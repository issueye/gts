package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
	"github.com/issueye/goscript/internal/proj"
	_ "github.com/issueye/goscript/internal/stdlib"
)

const version = "0.1.0-dev"
const defaultTimeout = 10 * time.Second

type options struct {
	checkTypes bool
	workers    int
	timeout    time.Duration
}

type runner struct {
	opts  options
	pool  *async.Pool
	cache *module.Cache
	vm    *object.VirtualMachine
}

type runOptions struct {
	autoMain bool
}

func main() {
	code := run(os.Args[1:])
	os.Exit(code)
}

func run(args []string) int {
	fs := flag.NewFlagSet("gs", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	opts := options{}
	fs.BoolVar(&opts.checkTypes, "check-types", false, "enable optional type checking")
	fs.IntVar(&opts.workers, "workers", runtime.NumCPU(), "maximum async worker count")
	fs.DurationVar(&opts.timeout, "timeout", defaultTimeout, "maximum script runtime; use 0 to disable")
	showVersion := fs.Bool("version", false, "print version")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(os.Stdout, "GoScript", version)
		return 0
	}
	if opts.checkTypes {
		fmt.Fprintln(os.Stderr, "type checking is not implemented yet; run without --check-types")
		return 2
	}

	r := newRunner(opts)
	rest := fs.Args()
	if len(rest) == 0 {
		printUsage(fs)
		return 2
	}

	var err error
	switch rest[0] {
	case "run":
		err = r.runProject(".")
	case "help", "-h", "--help":
		printUsage(fs)
		return 0
	default:
		err = r.runFile(rest[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newRunner(opts options) *runner {
	if opts.workers < 1 {
		opts.workers = 1
	}
	pool := async.NewPool(opts.workers)
	evaluator.SetPool(pool)
	return &runner{
		opts: opts,
		pool: pool,
	}
}

func printUsage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "Usage:\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] <script.gs>\n")
	fmt.Fprintf(fs.Output(), "  gs [flags] run\n\n")
	fmt.Fprintf(fs.Output(), "Flags:\n")
	fs.PrintDefaults()
}

func (r *runner) runProject(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	cfg := proj.Load(filepath.Join(absDir, "project.toml"))
	return r.runFileWithOptions(filepath.Join(absDir, cfg.Entry), runOptions{autoMain: true})
}

func (r *runner) runFile(path string) error {
	autoMain := strings.EqualFold(filepath.Base(path), "main.gs")
	return r.runFileWithOptions(path, runOptions{autoMain: autoMain})
}

func (r *runner) runFileWithOptions(path string, opts runOptions) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return r.withTimeout("script execution", func() error {
		if _, err := r.evalFile(absPath, opts); err != nil {
			return err
		}
		if err := r.waitGroup("async tasks", evaluator.AsyncWG.Wait); err != nil {
			return err
		}
		return r.waitGroup("worker pool", r.pool.Wait)
	})
}

func (r *runner) evalFile(absPath string, opts runOptions) (object.Object, error) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	r.vm = object.NewVirtualMachine()
	r.cache = module.NewCacheWithManager(r.vm.ObjectManager())
	env := r.vm.NewEnvironment()
	module.SetupExports(env)
	r.configureModuleLoaders(env, filepath.Dir(absPath))
	result, err := r.evalSource(string(src), absPath, env)
	if err != nil {
		return nil, err
	}
	if opts.autoMain {
		return r.callMain(env, absPath)
	}
	return result, nil
}

func (r *runner) evalSource(src, file string, env *object.Environment) (object.Object, error) {
	l := lexer.New(src)
	p := parser.New(l, file)
	program := p.ParseProgram()

	var parseErrors []string
	parseErrors = append(parseErrors, l.Errors()...)
	parseErrors = append(parseErrors, program.Errors...)
	if len(parseErrors) > 0 {
		return nil, errors.New(strings.Join(parseErrors, "\n"))
	}

	result := evaluator.Eval(program, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.waitPromise(promise, "top-level promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

func (r *runner) callMain(env *object.Environment, file string) (object.Object, error) {
	mainFn, ok := env.Get("main")
	if !ok {
		return object.UNDEFINED, nil
	}
	if _, ok := mainFn.(*object.Function); !ok {
		return nil, fmt.Errorf("%s: top-level main is not a function", file)
	}

	pos := ast.Position{File: file}
	call := &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "main",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "main"},
	}
	result := evaluator.Eval(call, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.waitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

func (r *runner) requireFunc(baseDir string) evaluator.RequireFn {
	return func(path string) (object.Object, error) {
		r.ensureRuntime()
		if native, ok := module.GetNative(path); ok {
			return native, nil
		}

		absPath := module.ResolvePath(path, baseDir)
		if !filepath.IsAbs(absPath) {
			var err error
			absPath, err = filepath.Abs(absPath)
			if err != nil {
				return nil, err
			}
		}
		if cached := r.cache.Get(absPath); cached != nil {
			return module.GetExports(cached), nil
		}

		env := r.cache.GetOrCreate(absPath)
		module.SetupExports(env)
		r.configureModuleLoaders(env, filepath.Dir(absPath))

		src, err := os.ReadFile(absPath)
		if err != nil {
			return nil, err
		}
		if _, err := r.evalSource(string(src), absPath, env); err != nil {
			return nil, err
		}
		return module.GetExports(env), nil
	}
}

func (r *runner) ensureRuntime() {
	if r.vm == nil {
		r.vm = object.NewVirtualMachine()
	}
	if r.cache == nil {
		r.cache = module.NewCacheWithManager(r.vm.ObjectManager())
	}
}

func (r *runner) configureModuleLoaders(env *object.Environment, baseDir string) {
	r.ensureRuntime()
	require := r.requireFunc(baseDir)
	evaluator.RegisterBuiltinsWithCache(env, require)
	evaluator.SetImportFunc(func(env *object.Environment, path string) (object.Object, error) {
		return require(path)
	})
}

func (r *runner) waitPromise(p *object.Promise, label string) (object.Object, error) {
	if r.opts.timeout <= 0 {
		return p.Wait(), nil
	}

	done := make(chan object.Object, 1)
	go func() {
		done <- p.Wait()
	}()
	select {
	case result := <-done:
		return result, nil
	case <-time.After(r.opts.timeout):
		return nil, fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}

func (r *runner) waitGroup(label string, wait func()) error {
	if r.opts.timeout <= 0 {
		wait()
		return nil
	}

	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(r.opts.timeout):
		return fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}

func (r *runner) withTimeout(label string, fn func() error) error {
	if r.opts.timeout <= 0 {
		return fn()
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	timer := time.NewTimer(r.opts.timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return err
	case <-timer.C:
		return fmt.Errorf("%s timed out after %s", label, r.opts.timeout)
	}
}
