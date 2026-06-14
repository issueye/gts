package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var promiseMethods map[string]object.BuiltinFunc

func init() {
	promiseMethods = map[string]object.BuiltinFunc{
		"then":    builtinPromiseThen,
		"catch":   builtinPromiseCatch,
		"finally": builtinPromiseFinally,
	}
}

func builtinPromiseThen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	promise := env.Extra.(*object.Promise)
	var onFulfilled object.Object
	var onRejected object.Object
	if len(args) > 0 {
		onFulfilled = args[0]
	}
	if len(args) > 1 {
		onRejected = args[1]
	}
	return chainPromise(env, pos, promise, onFulfilled, onRejected, nil)
}

func builtinPromiseCatch(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	promise := env.Extra.(*object.Promise)
	var onRejected object.Object
	if len(args) > 0 {
		onRejected = args[0]
	}
	return chainPromise(env, pos, promise, nil, onRejected, nil)
}

func builtinPromiseFinally(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	promise := env.Extra.(*object.Promise)
	var onFinally object.Object
	if len(args) > 0 {
		onFinally = args[0]
	}
	return chainPromise(env, pos, promise, nil, nil, onFinally)
}

func chainPromise(env *object.Environment, pos ast.Position, promise *object.Promise, onFulfilled, onRejected, onFinally object.Object) *object.Promise {
	next := env.ObjectManager().NewPromise()
	vm := env.VM()
	vm.AsyncAdd(1)
	vm.Go(func() {
		value := promise.Wait()
		state := promise.State()
		if err := vm.Post(func() {
			defer vm.AsyncDone()
			settlePromiseContinuation(env, pos, next, value, state, onFulfilled, onRejected, onFinally)
		}); err != nil {
			vm.AsyncDone()
			next.Reject(object.NewError(pos, "Promise continuation scheduler error: %v", err))
		}
	})
	return next
}

func settlePromiseContinuation(env *object.Environment, pos ast.Position, next *object.Promise, value object.Object, state object.PromiseState, onFulfilled, onRejected, onFinally object.Object) {
	if onFinally != nil && onFinally != object.UNDEFINED {
		finalResult := applyFunction(onFinally, env, nil, pos)
		if finalPromise, ok := finalResult.(*object.Promise); ok {
			settleAfterFinallyPromise(env, pos, next, finalPromise, value, state, onFulfilled, onRejected)
			return
		}
		if object.IsRuntimeError(finalResult) {
			next.Reject(finalResult)
			return
		}
	}
	settlePromiseContinuationAfterFinally(env, pos, next, value, state, onFulfilled, onRejected)
}

func settleAfterFinallyPromise(env *object.Environment, pos ast.Position, next *object.Promise, finalPromise *object.Promise, value object.Object, state object.PromiseState, onFulfilled, onRejected object.Object) {
	vm := env.VM()
	vm.AsyncAdd(1)
	vm.Go(func() {
		finalResult := finalPromise.Wait()
		finalState := finalPromise.State()
		if err := vm.Post(func() {
			defer vm.AsyncDone()
			if finalState == object.PROMISE_REJECTED {
				next.Reject(finalResult)
				return
			}
			settlePromiseContinuationAfterFinally(env, pos, next, value, state, onFulfilled, onRejected)
		}); err != nil {
			vm.AsyncDone()
			next.Reject(object.NewError(pos, "Promise finally scheduler error: %v", err))
		}
	})
}

func settlePromiseContinuationAfterFinally(env *object.Environment, pos ast.Position, next *object.Promise, value object.Object, state object.PromiseState, onFulfilled, onRejected object.Object) {
	if state == object.PROMISE_FULFILLED {
		if onFulfilled == nil || onFulfilled == object.UNDEFINED {
			next.Resolve(value)
			return
		}
		result := applyFunction(onFulfilled, env, []object.Object{value}, pos)
		settleChainedPromise(env, pos, next, result)
		return
	}

	if onRejected == nil || onRejected == object.UNDEFINED {
		next.Reject(value)
		return
	}
	result := applyFunction(onRejected, env, []object.Object{catchablePromiseReason(value)}, pos)
	settleChainedPromise(env, pos, next, result)
}

func settleChainedPromise(env *object.Environment, pos ast.Position, next *object.Promise, result object.Object) {
	if promise, ok := result.(*object.Promise); ok {
		vm := env.VM()
		vm.AsyncAdd(1)
		vm.Go(func() {
			settled := promise.Wait()
			state := promise.State()
			if err := vm.Post(func() {
				defer vm.AsyncDone()
				if state == object.PROMISE_REJECTED {
					next.Reject(settled)
					return
				}
				next.Resolve(settled)
			}); err != nil {
				vm.AsyncDone()
				next.Reject(object.NewError(pos, "Promise chain scheduler error: %v", err))
			}
		})
		return
	}
	if object.IsRuntimeError(result) {
		next.Reject(result)
		return
	}
	next.Resolve(result)
}

func catchablePromiseReason(reason object.Object) object.Object {
	if err, ok := reason.(*object.Error); ok {
		caught := *err
		caught.Runtime = false
		return &caught
	}
	return reason
}
