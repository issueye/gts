package evaluator

import (
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/object"
)

var AsyncWG sync.WaitGroup

// Pool is the global goroutine pool used by async functions and timers.
// Set from CLI before any script execution.
var Pool *async.Pool

// SetPool sets the goroutine pool for async operations.
func SetPool(p *async.Pool) { Pool = p }

// Go runs a function in the async pool, or a raw goroutine if pool is nil.
func Go(fn func()) {
	if Pool != nil {
		Pool.Go(fn)
	} else {
		go fn()
	}
}

func registerAsync(env *object.Environment) {
	env.Set("Promise", &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hk("__promiseConstructor"): {Key: &object.String{Value: "__promiseConstructor"}, Value: object.TRUE},
			hk("resolve"):              {Key: &object.String{Value: "resolve"}, Value: &object.Builtin{Name: "Promise.resolve", Fn: builtinPromiseResolve}},
			hk("reject"):               {Key: &object.String{Value: "reject"}, Value: &object.Builtin{Name: "Promise.reject", Fn: builtinPromiseReject}},
			hk("all"):                  {Key: &object.String{Value: "all"}, Value: &object.Builtin{Name: "Promise.all", Fn: builtinPromiseAll}},
		},
	})
	env.Set("setTimeout", &object.Builtin{Name: "setTimeout", Fn: builtinSetTimeout})
	env.Set("setInterval", &object.Builtin{Name: "setInterval", Fn: builtinSetInterval})
	env.Set("sleep", &object.Builtin{Name: "sleep", Fn: builtinSleep})
}

func constructPromise(env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: Promise constructor requires an executor function")
	}
	executor, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "TypeError: Promise executor must be a function")
	}
	promise := object.NewPromise()
	resolve := &object.Builtin{Name: "Promise.resolveExecutor", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		var value object.Object = object.UNDEFINED
		if len(args) > 0 {
			value = args[0]
		}
		if nested, ok := value.(*object.Promise); ok {
			value = nested.Wait()
			if nested.State() == object.PROMISE_REJECTED {
				promise.Reject(value)
				return object.UNDEFINED
			}
		}
		promise.Resolve(value)
		return object.UNDEFINED
	}}
	reject := &object.Builtin{Name: "Promise.rejectExecutor", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		var reason object.Object = object.UNDEFINED
		if len(args) > 0 {
			reason = args[0]
		}
		promise.Reject(reason)
		return object.UNDEFINED
	}}
	result := applyFunction(executor, env, []object.Object{resolve, reject}, pos)
	if object.IsRuntimeError(result) {
		promise.Reject(result)
	}
	return promise
}

func builtinPromiseResolve(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := object.NewPromise()
	if len(args) > 0 {
		p.Resolve(args[0])
	}
	return p
}

func builtinPromiseReject(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := object.NewPromise()
	if len(args) > 0 {
		p.Reject(args[0])
	}
	return p
}

func builtinPromiseAll(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := object.NewPromise()
	if len(args) < 1 {
		p.Resolve(object.UNDEFINED)
		return p
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		p.Reject(object.NewError(pos, "Promise.all requires an array"))
		return p
	}
	if len(arr.Elements) == 0 {
		p.Resolve(&object.Array{Elements: nil})
		return p
	}
	results := make([]object.Object, len(arr.Elements))
	remaining := len(arr.Elements)
	var mu sync.Mutex
	for i, elem := range arr.Elements {
		idx := i
		go func(el object.Object) {
			if pr, ok := el.(*object.Promise); ok {
				val := pr.Wait()
				if pr.State() == object.PROMISE_REJECTED {
					if p.State() == object.PROMISE_PENDING {
						p.Reject(val)
					}
					return
				}
				mu.Lock()
				results[idx] = val
				remaining--
				if remaining == 0 {
					p.Resolve(&object.Array{Elements: results})
				}
				mu.Unlock()
			} else {
				mu.Lock()
				results[idx] = el
				remaining--
				if remaining == 0 {
					p.Resolve(&object.Array{Elements: results})
				}
				mu.Unlock()
			}
		}(elem)
	}
	return p
}

func builtinSetTimeout(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "setTimeout requires a function and delay")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "setTimeout first arg must be a function")
	}
	delay, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "setTimeout second arg must be a number (ms)")
	}
	AsyncWG.Add(1)
	time.AfterFunc(time.Duration(delay.Value)*time.Millisecond, func() {
		defer AsyncWG.Done()
		Go(func() {
			scope := fn.Env.NewScope()
			Eval(fn.Body, scope)
		})
	})
	return object.UNDEFINED
}

func builtinSetInterval(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "setInterval requires a function and delay")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "setInterval first arg must be a function")
	}
	delay, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "setInterval second arg must be a number (ms)")
	}
	go func() {
		ticker := time.NewTicker(time.Duration(delay.Value) * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			Go(func() {
				scope := fn.Env.NewScope()
				Eval(fn.Body, scope)
			})
		}
	}()
	return object.UNDEFINED
}

func builtinSleep(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	ms, ok := args[0].(*object.Number)
	if !ok {
		return object.UNDEFINED
	}
	time.Sleep(time.Duration(ms.Value) * time.Millisecond)
	return object.UNDEFINED
}
