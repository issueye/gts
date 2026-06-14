package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/proj"
	"github.com/issueye/goscript/internal/runtime"
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
//
// Internally a Runtime wraps a runtime.Session, which owns the VM, async pool,
// module cache, and resolver. Host-specific concerns (host-registered modules,
// host method CallContext binding) live here; the Session owns the generic
// loader. This keeps the SDK API stable while sharing the loader with the CLI
// and @std/web isolated mode.
type Runtime struct {
	opts Options
	sess *runtime.Session

	modulesMu sync.RWMutex
	modules   map[string]Module
}

// NewRuntime creates a runtime. Call Close when the host is done with it.
func NewRuntime(opts Options) *Runtime {
	r := &Runtime{
		opts:    opts,
		modules: make(map[string]Module),
	}
	r.sess = runtime.NewSession(runtime.Options{
		Workers:    opts.Workers,
		Timeout:    opts.Timeout,
		CheckTypes: opts.CheckTypes,
		WorkingDir: opts.WorkingDir,
		Argv:       opts.Argv,
		// Host modules are layered in via this resolver, ahead of the global
		// native registry. The closure captures `r` so host methods receive the
		// correct *Runtime in their CallContext.
		NativeResolver: r.resolveHostModule,
	})
	return r
}

// resolveHostModule is the Session's NativeResolver hook. It consults the
// runtime's host-registered modules; returning (nil, false, nil) falls through
// to the global native registry inside the Session.
func (r *Runtime) resolveHostModule(env *object.Environment, specifier string) (object.Object, bool, error) {
	r.modulesMu.RLock()
	mod, ok := r.modules[specifier]
	r.modulesMu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	exports, err := moduleExports(r, mod, env)
	if err != nil {
		return nil, true, err
	}
	return exports, true, nil
}

// Close waits for outstanding async work and releases runtime-owned resources.
func (r *Runtime) Close() error {
	if r == nil {
		return nil
	}
	var err error
	if r.sess != nil {
		err = r.sess.Drain()
		r.sess.Close()
	}
	return err
}

// VM returns the runtime's virtual machine. Exposed for advanced hosts that
// need to install low-level hooks (spawner, scheduler, evaluator).
func (r *Runtime) VM() *object.VirtualMachine {
	if r == nil || r.sess == nil {
		return nil
	}
	return r.sess.VM()
}

// Session returns the underlying runtime.Session. Exposed so hosts that need
// the loader directly (e.g. to build per-request sessions for @std/web) can
// reach it.
func (r *Runtime) Session() *runtime.Session { return r.sess }

// RunSource evaluates source code with a synthetic file name.
func (r *Runtime) RunSource(source, file string) (Value, error) {
	if file == "" {
		file = "<embedded>"
	}
	return r.withTimeout(func() (Value, error) {
		env := r.sess.NewEnvironment()
		baseDir := r.baseDirFor(file)
		r.sess.Configure(env, baseDir)
		result, err := r.sess.EvalSource(source, file, env)
		if err != nil {
			return nil, err
		}
		if err := r.sess.Drain(); err != nil {
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
	return r.withTimeout(func() (Value, error) {
		env := r.sess.NewEnvironment()
		r.sess.Configure(env, filepath.Dir(absPath))
		r.sess.SetBootstrapSource(string(source))
		result, err := r.sess.EvalSource(string(source), absPath, env)
		if err != nil {
			return nil, err
		}
		if autoMain {
			result, err = r.callMain(env, absPath)
			if err != nil {
				return nil, err
			}
		}
		if err := r.sess.Drain(); err != nil {
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

// CallExport loads a script file, reads one exported function, and calls it
// with the provided arguments.
func (r *Runtime) CallExport(path, exportName string, args ...Value) (Value, error) {
	if exportName == "" {
		return nil, fmt.Errorf("export name is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	source, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return r.withTimeout(func() (Value, error) {
		env := r.sess.NewEnvironment()
		r.sess.Configure(env, filepath.Dir(absPath))
		if _, err := r.sess.EvalSource(string(source), absPath, env); err != nil {
			return nil, err
		}
		fn, err := exportedValue(module.GetExports(env), exportName)
		if err != nil {
			return nil, err
		}
		result, err := r.callValue(fn, env, args, absPath)
		if err != nil {
			return nil, err
		}
		if err := r.sess.Drain(); err != nil {
			return nil, err
		}
		return result, nil
	})
}

func (r *Runtime) baseDirFor(file string) string {
	if r.opts.WorkingDir != "" {
		return r.opts.WorkingDir
	}
	if strings.HasPrefix(file, "<") {
		if wd, err := os.Getwd(); err == nil {
			return wd
		}
		return ""
	}
	return filepath.Dir(file)
}

func (r *Runtime) withTimeout(fn func() (Value, error)) (Value, error) {
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
		return nil, fmt.Errorf("script execution timed out after %s", r.opts.Timeout)
	}
}

// asyncPool is retained for backwards-compatible access from hosts that may
// reach into it; it mirrors the Session's pool.
func (r *Runtime) asyncPool() *async.Pool {
	if r == nil || r.sess == nil {
		return nil
	}
	return r.sess.Pool()
}

// ensure stdlib is referenced even if all prior call sites moved to the
// Session (StopTerminalSessionsForVM is invoked via Session.Close, but keep
// the import alive for hosts relying on stdlib symbols re-exported elsewhere).
var _ = stdlib.StopTerminalSessionsForVM

// errRuntimeClosed is a sentinel for closed-runtimes; currently unused but kept
// to make future Close discipline explicit.
var errRuntimeClosed = errors.New("runtime is closed")
