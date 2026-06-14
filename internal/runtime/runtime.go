// Package runtime provides the reusable execution unit shared by the CLI,
// the embeddable SDK, and per-request isolated VMs in @std/web.
//
// A Session owns one VirtualMachine together with its bounded goroutine pool,
// module cache, resolver, and the host wiring (require/import/evaluator) needed
// to run a script. Every Session is fully isolated from every other Session:
// VM, ObjectManager, module.Cache, global constants, and async wait group are
// per-instance, so two Sessions loading the same source file get independent
// module-top-level state.
//
// NewSession is the constructor for a fresh, fully-wired session. A warm
// InstancePool (see pool.go) reuses Sessions that have already loaded a
// bootstrap script, which is how @std/web isolated mode avoids re-parsing the
// app on every request.
package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
	"github.com/issueye/goscript/internal/stdlib"
)

// NativeResolver resolves a native module specifier (e.g. "@std/web",
// "@go/foo") to its exports object within a given environment. Hosts use this
// hook to layer in their own native modules (SDK host modules, CLI plugins)
// before falling back to the global native registry.
//
// Returning (nil, false, nil) means "not handled by this host"; the Session
// then consults the global registry via module.GetNative.
type NativeResolver func(env *object.Environment, specifier string) (object.Object, bool, error)

// Options configures a Session.
type Options struct {
	// Workers caps the number of goroutines the Session's async pool runs
	// concurrently. Values <= 0 default to runtime.NumCPU().
	Workers int

	// Timeout bounds any single Drain / LoadEntry / await. Zero means no
	// timeout (block indefinitely).
	Timeout time.Duration

	// CheckTypes enables optional type checking in the VM.
	CheckTypes bool

	// WorkingDir is chdir'd into for the duration of script execution when
	// non-empty. Module resolution is rooted at RootDir regardless.
	WorkingDir string

	// RootDir is the project root used for module resolution. When empty it is
	// derived from WorkingDir.
	RootDir string

	// Argv sets process.argv for the script. The first element should be the
	// executable, the second the entry path, followed by script arguments.
	Argv []string

	// NativeResolver is an optional host hook consulted before the global
	// native registry.
	NativeResolver NativeResolver
}

// Session is one isolated execution unit: VM + pool + cache + resolver.
//
// The zero value is not usable; construct via NewSession.
type Session struct {
	opts     Options
	vm       *object.VirtualMachine
	pool     *async.Pool
	cache    *module.Cache
	resolver *module.Resolver
	rootDir  string

	// loading tracks modules whose source is currently being evaluated down the
	// require() call stack, so a cycle (A -> B -> A) surfaces as an error
	// instead of infinite recursion. Guarded by loadingMu. The CLI runner
	// previously owned this; it now lives on the Session so every consumer
	// (CLI, SDK, @std/web isolated) gets cycle detection for free.
	loading   map[string]bool
	loadingMu sync.Mutex

	closeOnce sync.Once
}

// NewSession constructs a fresh, fully-wired Session. The VM is configured with
// a bounded spawner, typecheck flag, argv, evaluator, importer, and module
// cache.
func NewSession(opts Options) *Session {
	if opts.Workers < 1 {
		opts.Workers = goruntimeNumCPU()
	}
	vm := object.NewVirtualMachine()
	vm.SetTypeCheck(opts.CheckTypes)
	pool := async.NewPool(opts.Workers)
	vm.SetSpawner(pool.Go)
	vm.SetArgv(normalizeArgv(opts.Argv))
	if !vm.HasEvaluator() {
		vm.SetEvaluator(func(node any, env *object.Environment) object.Object {
			return evaluator.Eval(node.(ast.Node), env)
		})
	}
	rootDir := opts.RootDir
	if rootDir == "" {
		rootDir = module.FindProjectRoot(opts.WorkingDir)
	}
	return &Session{
		opts:     opts,
		vm:       vm,
		pool:     pool,
		resolver: module.NewResolver(rootDir),
		rootDir:  rootDir,
		cache:    module.NewCacheWithVM(vm),
		loading:  make(map[string]bool),
	}
}

// VM returns the session's virtual machine.
func (s *Session) VM() *object.VirtualMachine { return s.vm }

// SetBootstrapSource records the entry script source on this session's VM.
// @std/web isolated mode reads it at app.listen() time to replay the app
// definition in per-request VMs. Hosts set this just before evaluating the
// entry script.
func (s *Session) SetBootstrapSource(source string) {
	if s == nil || s.vm == nil {
		return
	}
	s.vm.SetBootstrapSource(source)
}

// Pool returns the session's async goroutine pool.
func (s *Session) Pool() *async.Pool { return s.pool }

// Cache returns the session's module cache.
func (s *Session) Cache() *module.Cache { return s.cache }

// Resolver returns the session's module resolver.
func (s *Session) Resolver() *module.Resolver { return s.resolver }

// RootDir returns the project root used for module resolution.
func (s *Session) RootDir() string { return s.rootDir }

// Options returns the options the session was constructed with.
func (s *Session) Options() Options { return s.opts }

// NewEnvironment returns a fresh root environment bound to this session's VM
// with module exports scaffolding installed.
func (s *Session) NewEnvironment() *object.Environment {
	env := s.vm.NewEnvironment()
	module.SetupExports(env)
	return env
}

// Configure wires require/import for an environment rooted at baseDir. This
// registers the global require builtin and the VM's importer. Hosts should
// call it once per root environment before evaluating entry source.
func (s *Session) Configure(env *object.Environment, baseDir string) {
	env.ModuleDir = baseDir
	require := func(specifier string) (object.Object, error) {
		return s.requireFrom(env, specifier)
	}
	evaluator.RegisterBuiltinsWithCache(env, require)
	env.VM().SetImportFunc(func(importEnv *object.Environment, specifier string) (object.Object, error) {
		return s.requireFrom(importEnv, specifier)
	})
}

// EvalSource lexes, parses, and evaluates source under env. It handles
// top-level Promise results (blocking on settle, subject to Timeout) and
// translates runtime errors into Go errors.
func (s *Session) EvalSource(source, file string, env *object.Environment) (object.Object, error) {
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
		result, err = s.waitPromise(promise, "top-level promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

// LoadEntry runs a bootstrap script and, when autoMain is true, calls a
// top-level main() function. WorkingDir is honored via chdir for the duration.
//
// This is the shared "load an app / script entry" path used by the CLI, SDK,
// and (via InstancePool) the @std/web isolated request handler.
func (s *Session) LoadEntry(source, file string, autoMain bool) (object.Object, error) {
	baseDir := s.opts.WorkingDir
	if baseDir == "" {
		if file != "" && !strings.HasPrefix(file, "<") {
			baseDir = filepath.Dir(file)
		} else if wd, err := os.Getwd(); err == nil {
			baseDir = wd
		}
	}
	restore, err := enterWorkingDir(s.opts.WorkingDir)
	if err != nil {
		return nil, err
	}
	defer restore()
	env := s.NewEnvironment()
	s.Configure(env, baseDir)
	result, err := s.EvalSource(source, file, env)
	if err != nil {
		return nil, err
	}
	if autoMain {
		result, err = s.callMain(env, file)
		if err != nil {
			return nil, err
		}
	}
	if err := s.Drain(); err != nil {
		return nil, err
	}
	return result, nil
}

// Drain waits for outstanding async work: first VM async tasks (timers,
// promises), then the worker pool. Subject to Timeout.
func (s *Session) Drain() error {
	if s.vm != nil {
		if err := s.waitGroup("async tasks", s.vm.WaitAsync); err != nil {
			return err
		}
	}
	if s.pool != nil {
		if err := s.waitGroup("worker pool", s.pool.Wait); err != nil {
			return err
		}
	}
	return nil
}

// Close releases session-owned resources. It is idempotent. After Close the
// session must not be used. Terminal sessions bound to the VM are torn down so
// an isolated VM does not leak pty readers.
func (s *Session) Close() {
	s.closeOnce.Do(func() {
		if s.vm != nil {
			stdlib.StopTerminalSessionsForVM(s.vm)
		}
	})
}

// callMain invokes a top-level main() if present, mirroring CLI/SDK behavior.
func (s *Session) callMain(env *object.Environment, file string) (object.Object, error) {
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
		result, err = s.waitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

// requireFrom resolves and loads a module within this session. Native modules
// consult the host NativeResolver first, then the global registry. Source
// modules are cached per-session so repeated require() returns the same env.
func (s *Session) requireFrom(env *object.Environment, specifier string) (object.Object, error) {
	baseDir := env.ModuleDir
	if baseDir == "" {
		baseDir = s.rootDir
	}
	resolved, err := s.resolver.Resolve(specifier, module.ResolveOptions{ProjectRoot: s.rootDir, BaseDir: baseDir})
	if err != nil {
		return nil, err
	}
	if resolved.Kind == module.ModuleKindNative {
		if s.opts.NativeResolver != nil {
			native, ok, err := s.opts.NativeResolver(env, specifier)
			if err != nil {
				return nil, err
			}
			if ok {
				return native, nil
			}
		}
		native, ok := module.GetNative(specifier, env)
		if !ok {
			return nil, fmt.Errorf("native module %s is not registered", specifier)
		}
		return native, nil
	}
	return s.requireResolved(resolved)
}

func (s *Session) requireResolved(resolved module.ResolvedModule) (object.Object, error) {
	cacheKey := resolved.ID
	if cacheKey == "" {
		cacheKey = resolved.Path
	}
	if cached := s.cache.Get(cacheKey); cached != nil {
		return module.GetExports(cached), nil
	}
	// Cycle detection: if this module is already being evaluated somewhere up
	// the require() call stack, fail fast. A->B->A must not recurse forever.
	s.loadingMu.Lock()
	if s.loading[cacheKey] {
		s.loadingMu.Unlock()
		return nil, fmt.Errorf("circular dependency detected: %s", resolved.Path)
	}
	s.loading[cacheKey] = true
	s.loadingMu.Unlock()
	defer func() {
		s.loadingMu.Lock()
		delete(s.loading, cacheKey)
		s.loadingMu.Unlock()
	}()

	env := s.cache.GetOrCreate(cacheKey)
	module.SetupExports(env)
	s.Configure(env, module.ResolvedModuleDir(resolved))
	source, err := module.ReadResolvedSource(resolved)
	if err != nil {
		return nil, err
	}
	if _, err := s.EvalSource(source, resolved.Path, env); err != nil {
		return nil, err
	}
	return module.GetExports(env), nil
}

// WaitPromise blocks until p settles and returns its value (or reason). Hosts
// use this to bridge script-returned Promises into synchronous Go control flow,
// subject to the session's Timeout. Exposed (vs the internal waitPromise) so
// the SDK and the @std/web layer can share one await discipline.
func (s *Session) WaitPromise(p *object.Promise, label string) (object.Object, error) {
	return s.waitPromise(p, label)
}

func (s *Session) waitPromise(p *object.Promise, label string) (object.Object, error) {
	if s.opts.Timeout <= 0 {
		return p.Wait(), nil
	}
	done := make(chan object.Object, 1)
	go func() { done <- p.Wait() }()
	select {
	case result := <-done:
		return result, nil
	case <-time.After(s.opts.Timeout):
		return nil, fmt.Errorf("%s timed out after %s", label, s.opts.Timeout)
	}
}

func (s *Session) waitGroup(label string, wait func()) error {
	if s.opts.Timeout <= 0 {
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
	case <-time.After(s.opts.Timeout):
		return fmt.Errorf("%s timed out after %s", label, s.opts.Timeout)
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
	return func() { _ = os.Chdir(oldWd) }, nil
}

func normalizeArgv(argv []string) []string {
	out := append([]string{}, argv...)
	if len(out) == 0 {
		out = []string{defaultArgv0()}
	}
	return out
}

func defaultArgv0() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return "goscript"
}
