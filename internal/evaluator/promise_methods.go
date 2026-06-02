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
	next := object.NewPromise()
	AsyncWG.Add(1)
	Go(func() {
		defer AsyncWG.Done()
		value := promise.Wait()
		state := promise.State()

		if onFinally != nil && onFinally != object.UNDEFINED {
			finalResult := applyFunction(onFinally, env, nil, pos)
			if finalPromise, ok := finalResult.(*object.Promise); ok {
				finalResult = finalPromise.Wait()
				if finalPromise.State() == object.PROMISE_REJECTED {
					next.Reject(finalResult)
					return
				}
			}
			if object.IsRuntimeError(finalResult) {
				next.Reject(finalResult)
				return
			}
		}

		if state == object.PROMISE_FULFILLED {
			if onFulfilled == nil || onFulfilled == object.UNDEFINED {
				next.Resolve(value)
				return
			}
			result := applyFunction(onFulfilled, env, []object.Object{value}, pos)
			settleChainedPromise(next, result)
			return
		}

		if onRejected == nil || onRejected == object.UNDEFINED {
			next.Reject(value)
			return
		}
		result := applyFunction(onRejected, env, []object.Object{catchablePromiseReason(value)}, pos)
		settleChainedPromise(next, result)
	})
	return next
}

func settleChainedPromise(next *object.Promise, result object.Object) {
	if promise, ok := result.(*object.Promise); ok {
		settled := promise.Wait()
		if promise.State() == object.PROMISE_REJECTED {
			next.Reject(settled)
			return
		}
		result = settled
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
