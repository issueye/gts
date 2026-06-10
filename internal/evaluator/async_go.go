package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// builtinGo spawns a goroutine to execute a function asynchronously.
// Usage: go(fn, ...args)
func builtinGo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "go() requires a function argument")
	}

	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "go() argument must be a function, got %s", args[0].Type())
	}

	// 提取传递给函数的参数
	fnArgs := []object.Object{}
	if len(args) > 1 {
		fnArgs = args[1:]
	}

	// 使用 VM 的异步机制
	vm := env.VM()
	vm.AsyncAdd(1)

	go func() {
		defer vm.AsyncDone()
		defer func() {
			if r := recover(); r != nil {
				// 捕获 panic，避免 goroutine 崩溃影响主程序
			}
		}()

		// 执行函数
		applyFunction(fn, env, fnArgs, pos)
	}()

	return object.NULL
}
