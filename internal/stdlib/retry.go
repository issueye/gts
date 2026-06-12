package stdlib

import (
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/retry", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "run", &object.Builtin{Name: "retry.run", Fn: retryRun})
		setHashMember(exports, "exponential", &object.Builtin{Name: "retry.exponential", Fn: retryExponential})
		return exports, nil
	})
}

func retryRun(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "retry.run requires function")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "retry.run expects function")
	}

	times := 3
	delay := 0
	backoff := 1.0

	if len(args) > 1 {
		if opts, ok := args[1].(*object.Hash); ok {
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "times"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					times = int(n.Value)
				}
			}
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "delay"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					delay = int(n.Value)
				}
			}
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "backoff"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					backoff = n.Value
				}
			}
		}
	}

	var lastErr object.Object
	currentDelay := delay

	for i := 0; i < times; i++ {
		result := evaluator.Eval(fn.Body, fn.Env)
		if result.Type() != object.ERROR_OBJ {
			return result
		}
		lastErr = result

		if i < times-1 && currentDelay > 0 {
			time.Sleep(time.Duration(currentDelay) * time.Millisecond)
			currentDelay = int(float64(currentDelay) * backoff)
		}
	}

	return lastErr
}

func retryExponential(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "retry.exponential requires function")
	}

	times := 5
	initialDelay := 1000

	if len(args) > 1 {
		if opts, ok := args[1].(*object.Hash); ok {
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "times"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					times = int(n.Value)
				}
			}
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "initialDelay"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					initialDelay = int(n.Value)
				}
			}
		}
	}

	newArgs := []object.Object{
		args[0],
		&object.Hash{Pairs: make(map[object.HashKey]object.HashPair)},
	}
	opts := newArgs[1].(*object.Hash)
	setHashMember(opts, "times", &object.Number{Value: float64(times)})
	setHashMember(opts, "delay", &object.Number{Value: float64(initialDelay)})
	setHashMember(opts, "backoff", &object.Number{Value: 2.0})

	return retryRun(env, pos, newArgs...)
}
