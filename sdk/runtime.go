package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
	"github.com/issueye/goscript/internal/proj"
	"github.com/issueye/goscript/internal/stdlib"
)

// Options configures a Runtime.
type Options struct {
	Workers    int
	Timeout    time.Duration
	CheckTypes bool
	WorkingDir string
	Argv       []string
}

// Runtime is an embeddable GoScript execution environment for Go programs.
type Runtime struct {
	opts     Options
	vm       *object.VirtualMachine
	pool     *async.Pool
	cache    *module.Cache
	resolver *module.Resolver
	rootDir  string
}

// NewRuntime creates a runtime. Call Close when the host is done with it.
func NewRuntime(opts Options) *Runtime {
	if opts.Workers < 1 {
		opts.Workers = runtime.NumCPU()
	}
	vm := object.NewVirtualMachine()
	vm.SetTypeCheck(opts.CheckTypes)
	pool := async.NewPool(opts.Workers)
	vm.SetSpawner(pool.Go)
	return &Runtime{
		opts: opts,
		vm:   vm,
		pool: pool,
	}
}

// Close waits for outstanding async work and releases runtime-owned resources.
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	if err := r.drain(); err != nil {
		return err
	}
	if r.vm != nil {
		stdlib.StopTerminalSessionsForVM(r.vm)
	}
	return nil
}

// RunSource evaluates source code with a synthetic file name.
func (r *Runtime) RunSource(source, file string) (Value, error) {
	if file == "" {
		file = "<embedded>"
	}
	baseDir := r.opts.WorkingDir
	if baseDir == "" {
		if strings.HasPrefix(file, "<") {
			baseDir, _ = os.Getwd()
		} else {
			baseDir = filepath.Dir(file)
		}
	}
	return r.withTimeout("script execution", func() (Value, error) {
		restore, err := enterWorkingDir(r.opts.WorkingDir)
		if err != nil {
			return nil, err
		}
		defer restore()
		r.configure(baseDir)
		env := r.vm.NewEnvironment()
		module.SetupExports(env)
		r.configureModuleLoaders(env, baseDir)
		result, err := r.evalSource(source, file, env)
		if err != nil {
			return nil, err
		}
		if err := r.drain(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// RunFile evaluates a .gs file. If autoMain is true, top-level main() is called
// after the file is loaded.
func (r *Runtime) RunFile(path string, autoMain bool) (Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return r.withTimeout("script execution", func() (Value, error) {
		restore, err := enterWorkingDir(r.opts.WorkingDir)
		if err != nil {
			return nil, err
		}
		defer restore()
		baseDir := filepath.Dir(absPath)
		r.configure(baseDir)
		env := r.vm.NewEnvironment()
		module.SetupExports(env)
		r.configureModuleLoaders(env, baseDir)
		result, err := r.evalSource(string(source), absPath, env)
		if err != nil {
			return nil, err
		}
		if autoMain {
			result, err = r.callMain(env, absPath)
			if err != nil {
				return nil, err
			}
		}
		if err := r.drain(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

// RunProject loads project.toml from dir and runs its entry with autoMain.
func (r *Runtime) RunProject(dir string) (Value, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg, err := proj.LoadStrict(filepath.Join(absDir, "project.toml"))
	if err != nil {
		return nil, err
	}
	return r.RunFile(filepath.Join(absDir, cfg.Entry), true)
}

func (r *Runtime) configure(baseDir string) {
	if r.vm == nil {
		r.vm = object.NewVirtualMachine()
	}
	r.vm.SetTypeCheck(r.opts.CheckTypes)
	if r.pool == nil {
		r.pool = async.NewPool(r.opts.Workers)
		r.vm.SetSpawner(r.pool.Go)
	}
	argv := append([]string{}, r.opts.Argv...)
	if len(argv) == 0 {
		argv = []string{executableArgv0()}
	}
	r.vm.SetArgv(argv)
	r.rootDir = module.FindProjectRoot(baseDir)
	r.resolver = module.NewResolver(r.rootDir)
	r.cache = module.NewCacheWithVM(r.vm)
}

func (r *Runtime) evalSource(source, file string, env *object.Environment) (Value, error) {
	l := lexer.New(source)
	p := parser.New(l, file)
	program := p.ParseProgram()
	parseErrors := append([]string{}, l.Errors()...)
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

func (r *Runtime) configureModuleLoaders(env *object.Environment, baseDir string) {
	env.ModuleDir = baseDir
	requireFromEnv := func(loadEnv *object.Environment, specifier string) (object.Object, error) {
		currentBaseDir := loadEnv.ModuleDir
		if currentBaseDir == "" {
			currentBaseDir = baseDir
		}
		resolved, err := r.resolver.Resolve(specifier, module.ResolveOptions{ProjectRoot: r.rootDir, BaseDir: currentBaseDir})
		if err != nil {
			return nil, err
		}
		if resolved.Kind == module.ModuleKindNative {
			native, ok := module.GetNative(specifier, loadEnv)
			if !ok {
				return nil, fmt.Errorf("native module %s is not registered", specifier)
			}
			return native, nil
		}
		return r.requireResolved(resolved)
	}
	evaluator.RegisterBuiltinsWithCache(env, func(specifier string) (object.Object, error) {
		return requireFromEnv(env, specifier)
	})
	env.VM().SetImportFunc(func(importEnv *object.Environment, specifier string) (object.Object, error) {
		return requireFromEnv(importEnv, specifier)
	})
}

func (r *Runtime) requireResolved(resolved module.ResolvedModule) (object.Object, error) {
	cacheKey := resolved.ID
	if cacheKey == "" {
		cacheKey = resolved.Path
	}
	if cached := r.cache.Get(cacheKey); cached != nil {
		return module.GetExports(cached), nil
	}
	env := r.cache.GetOrCreate(cacheKey)
	module.SetupExports(env)
	r.configureModuleLoaders(env, resolvedModuleDir(resolved))
	source, err := readResolvedSource(resolved)
	if err != nil {
		return nil, err
	}
	if _, err := r.evalSource(source, resolved.Path, env); err != nil {
		return nil, err
	}
	return module.GetExports(env), nil
}

func (r *Runtime) waitPromise(p *object.Promise, label string) (object.Object, error) {
	if r.opts.Timeout <= 0 {
		return p.Wait(), nil
	}
	done := make(chan object.Object, 1)
	go func() {
		done <- p.Wait()
	}()
	select {
	case result := <-done:
		return result, nil
	case <-time.After(r.opts.Timeout):
		return nil, fmt.Errorf("%s timed out after %s", label, r.opts.Timeout)
	}
}

func (r *Runtime) drain() error {
	if r.vm != nil {
		if err := r.waitGroup("async tasks", r.vm.WaitAsync); err != nil {
			return err
		}
	}
	if r.pool != nil {
		if err := r.waitGroup("worker pool", r.pool.Wait); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) waitGroup(label string, wait func()) error {
	if r.opts.Timeout <= 0 {
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
	case <-time.After(r.opts.Timeout):
		return fmt.Errorf("%s timed out after %s", label, r.opts.Timeout)
	}
}

func (r *Runtime) withTimeout(label string, fn func() (Value, error)) (Value, error) {
	if r.opts.Timeout <= 0 {
		return fn()
	}
	done := make(chan struct {
		value Value
		err   error
	}, 1)
	go func() {
		value, err := fn()
		done <- struct {
			value Value
			err   error
		}{value: value, err: err}
	}()
	timer := time.NewTimer(r.opts.Timeout)
	defer timer.Stop()
	select {
	case result := <-done:
		return result.value, result.err
	case <-timer.C:
		return nil, fmt.Errorf("%s timed out after %s", label, r.opts.Timeout)
	}
}

func enterWorkingDir(dir string) (func(), error) {
	if dir == "" {
		return func() {}, nil
	}
	oldWd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(dir); err != nil {
		return nil, err
	}
	return func() {
		_ = os.Chdir(oldWd)
	}, nil
}

func executableArgv0() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		return exe
	}
	return "goscript"
}
