package evaluator

import (
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerAsync(env *object.Environment) {
	env.VM().SetGlobalConsts(map[string]object.Object{
		"Promise":        promiseConstructorObject(),
		"setTimeout":     &object.Builtin{Name: "setTimeout", Fn: builtinSetTimeout},
		"clearTimeout":   &object.Builtin{Name: "clearTimeout", Fn: builtinClearTimeout},
		"setInterval":    &object.Builtin{Name: "setInterval", Fn: builtinSetInterval},
		"clearInterval":  &object.Builtin{Name: "clearInterval", Fn: builtinClearInterval},
		"queueMicrotask": &object.Builtin{Name: "queueMicrotask", Fn: builtinQueueMicrotask},
		"sleep":          &object.Builtin{Name: "sleep", Fn: builtinSleep},
	})
}

func promiseConstructorObject() object.Object {
	return orderedHash(
		hashEntry("__promiseConstructor", object.TRUE),
		hashEntry("resolve", &object.Builtin{Name: "Promise.resolve", Fn: builtinPromiseResolve}),
		hashEntry("reject", &object.Builtin{Name: "Promise.reject", Fn: builtinPromiseReject}),
		hashEntry("all", &object.Builtin{Name: "Promise.all", Fn: builtinPromiseAll}),
		hashEntry("race", &object.Builtin{Name: "Promise.race", Fn: builtinPromiseRace}),
		hashEntry("allSettled", &object.Builtin{Name: "Promise.allSettled", Fn: builtinPromiseAllSettled}),
	)
}

func constructPromise(env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: Promise constructor requires an executor function")
	}
	executor, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "TypeError: Promise executor must be a function")
	}
	promise := env.ObjectManager().NewPromise()
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
	p := env.ObjectManager().NewPromise()
	if len(args) > 0 {
		p.Resolve(args[0])
	}
	return p
}

func builtinPromiseReject(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := env.ObjectManager().NewPromise()
	if len(args) > 0 {
		p.Reject(args[0])
	}
	return p
}

func builtinPromiseAll(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := env.ObjectManager().NewPromise()
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
		p.Resolve(env.ObjectManager().NewArray(nil))
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
					p.Resolve(env.ObjectManager().NewArray(results))
				}
				mu.Unlock()
			} else {
				mu.Lock()
				results[idx] = el
				remaining--
				if remaining == 0 {
					p.Resolve(env.ObjectManager().NewArray(results))
				}
				mu.Unlock()
			}
		}(elem)
	}
	return p
}

func builtinPromiseRace(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := env.ObjectManager().NewPromise()
	if len(args) < 1 {
		p.Resolve(object.UNDEFINED)
		return p
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		p.Reject(object.NewError(pos, "Promise.race requires an array"))
		return p
	}
	for _, elem := range arr.Elements {
		if pr, ok := elem.(*object.Promise); ok {
			if pr.State() != object.PROMISE_PENDING {
				val := pr.Wait()
				if pr.State() == object.PROMISE_REJECTED {
					p.Reject(val)
				} else {
					p.Resolve(val)
				}
				return p
			}
			continue
		}
		p.Resolve(elem)
		return p
	}
	for _, elem := range arr.Elements {
		go func(el object.Object) {
			pr := el.(*object.Promise)
			val := pr.Wait()
			if pr.State() == object.PROMISE_REJECTED {
				p.Reject(val)
				return
			}
			p.Resolve(val)
		}(elem)
	}
	return p
}

func builtinPromiseAllSettled(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	p := env.ObjectManager().NewPromise()
	if len(args) < 1 {
		p.Resolve(object.UNDEFINED)
		return p
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		p.Reject(object.NewError(pos, "Promise.allSettled requires an array"))
		return p
	}
	if len(arr.Elements) == 0 {
		p.Resolve(env.ObjectManager().NewArray(nil))
		return p
	}
	results := make([]object.Object, len(arr.Elements))
	remaining := len(arr.Elements)
	var mu sync.Mutex
	for i, elem := range arr.Elements {
		idx := i
		go func(el object.Object) {
			result := settledResult(env, "fulfilled", "value", el)
			if pr, ok := el.(*object.Promise); ok {
				val := pr.Wait()
				if pr.State() == object.PROMISE_REJECTED {
					result = settledResult(env, "rejected", "reason", val)
				} else {
					result = settledResult(env, "fulfilled", "value", val)
				}
			}
			mu.Lock()
			results[idx] = result
			remaining--
			if remaining == 0 {
				p.Resolve(env.ObjectManager().NewArray(results))
			}
			mu.Unlock()
		}(elem)
	}
	return p
}

func settledResult(env *object.Environment, status, field string, value object.Object) *object.Hash {
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.SetMember(&object.String{Value: "status"}, &object.String{Value: status})
	result.SetMember(&object.String{Value: field}, value)
	env.ObjectManager().Register(result)
	return result
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
	callArgs := append([]object.Object(nil), args[2:]...)
	var done sync.Once
	vm := env.VM()
	vm.AsyncAdd(1)
	timer := time.AfterFunc(time.Duration(delay.Value)*time.Millisecond, func() {
		vm.Go(func() {
			defer done.Do(vm.AsyncDone)
			callTimerFunction(fn, callArgs)
		})
	})
	id := &object.TimerId{ID: vm.NextTimerID()}
	env.ObjectManager().Register(id)
	id.Cancel = func() {
		if timer.Stop() {
			done.Do(vm.AsyncDone)
		}
	}
	return id
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
	callArgs := append([]object.Object(nil), args[2:]...)
	stop := make(chan struct{})
	var done sync.Once
	vm := env.VM()
	vm.AsyncAdd(1)
	vm.Go(func() {
		defer done.Do(vm.AsyncDone)
		ticker := time.NewTicker(time.Duration(delay.Value) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				vm.Go(func() {
					callTimerFunction(fn, callArgs)
				})
			case <-stop:
				return
			}
		}
	})
	id := &object.TimerId{ID: vm.NextTimerID()}
	env.ObjectManager().Register(id)
	id.Cancel = func() { closeOnce(env, stop, &done) }
	return id
}

func builtinClearTimeout(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	clearTimer(args)
	return object.UNDEFINED
}

func builtinClearInterval(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	clearTimer(args)
	return object.UNDEFINED
}

func builtinQueueMicrotask(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "queueMicrotask requires a function")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "queueMicrotask first arg must be a function")
	}
	vm := env.VM()
	vm.AsyncAdd(1)
	vm.Go(func() {
		defer vm.AsyncDone()
		callTimerFunction(fn, nil)
	})
	return object.UNDEFINED
}

func clearTimer(args []object.Object) {
	if len(args) < 1 {
		return
	}
	if id, ok := args[0].(*object.TimerId); ok && id.Cancel != nil {
		id.Cancel()
	}
}

func closeOnce(env *object.Environment, stop chan struct{}, done *sync.Once) {
	done.Do(func() {
		close(stop)
		env.VM().AsyncDone()
	})
}

func callTimerFunction(fn *object.Function, args []object.Object) object.Object {
	scope := fn.Env.NewScope()
	for i, p := range fn.Parameters {
		if i < len(args) {
			if p.Spread {
				rest := make([]object.Object, len(args)-i)
				copy(rest, args[i:])
				scope.Set(p.Name, fn.Env.ObjectManager().NewArray(rest))
				break
			}
			scope.Set(p.Name, args[i])
		} else {
			scope.Set(p.Name, object.UNDEFINED)
		}
	}
	return Eval(fn.Body, scope)
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
