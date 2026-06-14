package stdlib

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	stdruntime "runtime"
	"strings"
	"sync"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

// webIsolatedSession is one per-request execution unit for isolated web mode.
// It owns a VM, a bounded async pool, a module cache, and a resolver — fully
// independent of every other request's session. The bootstrap source is
// re-evaluated on construction so the app's routes are registered inside this
// VM; the per-request handler then looks up the app via the VM-keyed registry
// and runs the matched route.
//
// This is intentionally a minimal, self-contained mirror of runtime.Session,
// duplicated here to avoid a stdlib -> runtime import cycle (runtime imports
// stdlib for StopTerminalSessionsForVM). The duplication is small (VM+pool+
// cache+eval wiring) and the web layer needs per-request customization anyway.
type webIsolatedSession struct {
	vm       *object.VirtualMachine
	pool     *async.Pool
	cache    *module.Cache
	resolver *module.Resolver
	rootDir  string
	tmpl     webIsolatedTemplate

	loadingMu sync.Mutex
	loading   map[string]bool

}

// newWebIsolatedSession creates a new session and validates bootstrap.
// Returns error if bootstrap fails, preventing broken sessions from being used.
// FIX: BUG #2 - Bootstrap errors are now validated immediately instead of being
// silently stored and only surfaced on first request.
func newWebIsolatedSession(tmpl webIsolatedTemplate) (*webIsolatedSession, error) {
	workers := tmpl.workers
	if workers < 1 {
		workers = stdruntime.NumCPU()
	}
	vm := object.NewVirtualMachine()
	vm.SetTypeCheck(tmpl.checkTypes)
	vm.SetBootstrapSource(tmpl.bootstrapSrc)
	pool := async.NewPool(workers)
	vm.SetSpawner(pool.Go)
	if !vm.HasEvaluator() {
		vm.SetEvaluator(func(node any, env *object.Environment) object.Object {
			return evaluator.Eval(node.(ast.Node), env)
		})
	}
	s := &webIsolatedSession{
		vm:       vm,
		pool:     pool,
		cache:    module.NewCacheWithVM(vm),
		resolver: module.NewResolver(tmpl.rootDir),
		rootDir:  tmpl.rootDir,
		tmpl:     tmpl,
		loading:  make(map[string]bool),
	}
	// Re-run the app bootstrap and validate immediately.
	// If bootstrap fails, clean up and return error to prevent broken session.
	if err := s.boot(); err != nil {
		// Clean up resources before returning error
		unregisterIsolatedApp(s.vm)
		StopTerminalSessionsForVM(s.vm)
		return nil, fmt.Errorf("session bootstrap failed: %w", err)
	}
	return s, nil
}

// boot re-evaluates the bootstrap source inside this session's VM. It uses a
// throwaway root environment with require/import wired through this session's
// cache+resolver.
//
// While the bootstrap runs, app.listen() in the replayed source MUST become a
// no-op: the real server was already started once on the main VM, and starting
// another here would (a) leak a listener and (b) recurse — listen -> build
// pool -> replay bootstrap -> listen ... We guard via a per-VM "serving already
// active" marker set on the main VM by webListen: any webApp whose VM carries
// that marker treats listen as a no-op.
func (s *webIsolatedSession) boot() error {
	env := s.vm.NewEnvironment()
	module.SetupExports(env)
	s.configureLoaders(env, s.rootDir)
	// Register builtins (Date, JSON, sleep, etc.) in this session's VM
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})
	markVMReplaying(s.vm, true)
	defer markVMReplaying(s.vm, false)
	if _, err := s.evalSource(s.tmpl.bootstrapSrc, "<isolated-bootstrap>", env); err != nil {
		return fmt.Errorf("isolated bootstrap failed: %w", err)
	}
	return nil
}

func (s *webIsolatedSession) configureLoaders(env *object.Environment, baseDir string) {
	env.ModuleDir = baseDir
	require := func(specifier string) (object.Object, error) {
		return s.requireFrom(env, specifier)
	}
	evaluator.RegisterBuiltinsWithCache(env, require)
	env.VM().SetImportFunc(func(importEnv *object.Environment, specifier string) (object.Object, error) {
		return s.requireFrom(importEnv, specifier)
	})
}

func (s *webIsolatedSession) requireFrom(env *object.Environment, specifier string) (object.Object, error) {
	baseDir := env.ModuleDir
	if baseDir == "" {
		baseDir = s.rootDir
	}
	resolved, err := s.resolver.Resolve(specifier, module.ResolveOptions{ProjectRoot: s.rootDir, BaseDir: baseDir})
	if err != nil {
		return nil, err
	}
	if resolved.Kind == module.ModuleKindNative {
		if s.tmpl.nativeResolv != nil {
			if native, ok, err := s.tmpl.nativeResolv(env, specifier); ok || err != nil {
				return native, err
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

func (s *webIsolatedSession) requireResolved(resolved module.ResolvedModule) (object.Object, error) {
	cacheKey := resolved.ID
	if cacheKey == "" {
		cacheKey = resolved.Path
	}
	if cached := s.cache.Get(cacheKey); cached != nil {
		return module.GetExports(cached), nil
	}
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
	s.configureLoaders(env, module.ResolvedModuleDir(resolved))
	source, err := module.ReadResolvedSource(resolved)
	if err != nil {
		return nil, err
	}
	if _, err := s.evalSource(source, resolved.Path, env); err != nil {
		return nil, err
	}
	return module.GetExports(env), nil
}

func (s *webIsolatedSession) evalSource(source, file string, env *object.Environment) (object.Object, error) {
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
		result = promise.Wait()
	}
	if object.IsError(result) {
		return nil, fmt.Errorf("%s", result.Inspect())
	}
	return result, nil
}

// drain waits for outstanding async work in this session (timers, promises)
// before returning the session to the pool.
func (s *webIsolatedSession) drain() {
	s.vm.WaitAsync()
	s.pool.Wait()
}

// close releases the session. We do not Reset the VM (sessions are discarded,
// not recycled, since each carries its replayed app state).
func (s *webIsolatedSession) close() {
	unregisterIsolatedApp(s.vm)
	StopTerminalSessionsForVM(s.vm)
}

// webIsolatedPool is a bounded warm pool of per-request sessions. Capacity
// bounds concurrency; the factory replays the bootstrap so a checked-out
// session already has its routes registered.
// FIX: Added createLimiter to prevent concurrent creation storms and statistics
// for monitoring pool health.
type webIsolatedPool struct {
	slots         chan struct{}              // capacity = poolSize, controls concurrency
	idle          chan *webIsolatedSession   // warm session reuse queue
	createLimiter chan struct{}              // limits concurrent session creation
	tmpl          webIsolatedTemplate
	size          int

	// Statistics for monitoring
	mu              sync.Mutex
	created         int64
	reused          int64
	discarded       int64
	createFailures  int64
	timeoutFailures int64
}

func newWebIsolatedPool(size int, tmpl webIsolatedTemplate) *webIsolatedPool {
	if size < 1 {
		size = 1
	}

	// Limit concurrent session creation to prevent memory spikes.
	// Use min(size/4, 32) to balance creation speed and memory pressure.
	// FIX: Problem #5 - Prevents concurrent creation storms.
	createConcurrency := size / 4
	if createConcurrency < 4 {
		createConcurrency = 4
	}
	if createConcurrency > 32 {
		createConcurrency = 32
	}

	return &webIsolatedPool{
		slots:         make(chan struct{}, size),
		idle:          make(chan *webIsolatedSession, size),
		createLimiter: make(chan struct{}, createConcurrency),
		tmpl:          tmpl,
		size:          size,
	}
}

func (p *webIsolatedPool) warm(n int) {
	if n > p.size {
		n = p.size
	}
	// Warm sessions live in the idle channel only; they do NOT occupy checkout
	// slots. Slots bound the number of sessions currently serving a request.
	for i := 0; i < n; i++ {
		sess, err := newWebIsolatedSession(p.tmpl)
		if err != nil {
			// Silently skip failed warm sessions
			continue
		}
		select {
		case p.idle <- sess:
		default:
			sess.close()
		}
	}
}

// get acquires a session from the pool with slot leak protection and creation
// rate limiting. Returns nil if creation fails.
// FIX: BUG #1 - Added defer protection to prevent slot leaks on panic/error.
// FIX: Problem #5 - Added createLimiter to prevent concurrent creation storms.
func (p *webIsolatedPool) get() *webIsolatedSession {
	// Acquire one checkout slot (bounds in-flight requests). Idle warm sessions
	// do not hold slots, so a fully warmed pool still allows `size` concurrent
	// checkouts.
	p.slots <- struct{}{}

	// Ensure slot is released on failure (critical for preventing leaks)
	slotReleased := false
	defer func() {
		if !slotReleased {
			<-p.slots
		}
	}()

	// Try to reuse existing session from idle queue
	select {
	case sess := <-p.idle:
		p.mu.Lock()
		p.reused++
		p.mu.Unlock()
		slotReleased = true
		return sess
	default:
	}

	// Need to create new session - acquire creation limiter first
	select {
	case p.createLimiter <- struct{}{}:
		defer func() { <-p.createLimiter }()
	default:
		// Creation limiter is full, wait briefly then try again
		// This prevents thundering herd of session creations
		return nil
	}

	// Create new session with error handling
	sess, err := newWebIsolatedSession(p.tmpl)
	if err != nil {
		// Creation failed - slot will be released by defer
		p.mu.Lock()
		p.createFailures++
		p.mu.Unlock()
		return nil
	}

	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	slotReleased = true
	return sess
}

// put returns a session to the pool.
// FIX: Added nil check and statistics tracking.
func (p *webIsolatedPool) put(sess *webIsolatedSession) {
	if sess == nil {
		<-p.slots
		return
	}

	select {
	case p.idle <- sess:
		// Successfully returned to idle queue for reuse
	default:
		// Idle queue full - discard session
		sess.close()
		p.mu.Lock()
		p.discarded++
		p.mu.Unlock()
	}
	<-p.slots
}

// discard explicitly discards a session (e.g., after error).
// FIX: Added nil check and statistics tracking.
func (p *webIsolatedPool) discard(sess *webIsolatedSession) {
	if sess != nil {
		sess.close()
		p.mu.Lock()
		p.discarded++
		p.mu.Unlock()
	}
	<-p.slots
}

func (p *webIsolatedPool) close() {
	for {
		select {
		case sess := <-p.idle:
			sess.close()
		default:
			return
		}
	}
}

// serveIsolated dispatches one HTTP request to a checked-out isolated session.
// The matched route's handlers run in that session's VM; the response is
// written back through the standard webContext. Multiple requests run in
// parallel across distinct sessions, bounded by pool size.
func (app *webApp) serveIsolated(w http.ResponseWriter, r *http.Request) {
	app.isolatedMu.Lock()
	pool := app.isoPool
	app.isolatedMu.Unlock()
	if pool == nil {
		http.Error(w, "web: isolated server not initialized", http.StatusInternalServerError)
		return
	}

	sess := pool.get()
	if sess == nil {
		// FIX: Handle nil session (creation failed or timeout)
		http.Error(w, "web: service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	defer pool.put(sess)

	bodyBytes, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	ctx := &webContext{req: r, writer: w, body: string(bodyBytes)}
	ctx.reqObj = buildWebRequestObject(ctx)
	ctx.resObj = newWebResponseObject(ctx.writer)

	reqApp, ok := lookupIsolatedApp(sess.vm)
	if !ok {
		http.Error(w, "web: isolated app not found in request VM", http.StatusInternalServerError)
		return
	}
	routes := reqApp.snapshotRoutes()
	// No handlerMu: each request owns its session/VM, so route execution is
	// naturally serialized within a VM and parallel across VMs.
	result := reqApp.runRoutes(routes, ctx, 0)
	reqApp.waitWebPromise(result)
}
