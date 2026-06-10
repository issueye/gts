package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// makeChannel creates a new channel with optional capacity
func builtinMakeChannel(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	capacity := 0
	if len(args) > 0 {
		if num, ok := args[0].(*object.Number); ok {
			capacity = int(num.Value)
		}
	}
	ch := object.NewChannel(capacity)

	// Return object with methods
	return orderedHash(
		hashEntry("send", &object.Builtin{Name: "Channel.send", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "Channel.send requires a value")
			}
			ok := ch.Send(args[0])
			return object.NativeBool(ok)
		}}),
		hashEntry("recv", &object.Builtin{Name: "Channel.recv", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			val, ok := ch.Recv()
			if !ok {
				return object.UNDEFINED
			}
			return val
		}}),
		hashEntry("tryRecv", &object.Builtin{Name: "Channel.tryRecv", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			val, ok := ch.TryRecv()
			if !ok {
				return object.UNDEFINED
			}
			return val
		}}),
		hashEntry("close", &object.Builtin{Name: "Channel.close", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			ch.Close()
			return object.UNDEFINED
		}}),
		hashEntry("isClosed", &object.Builtin{Name: "Channel.isClosed", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			return object.NativeBool(ch.IsClosed())
		}}),
	)
}

// makeWaitGroup creates a new wait group
func builtinMakeWaitGroup(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	wg := object.NewWaitGroup()

	// Return object with methods
	return orderedHash(
		hashEntry("add", &object.Builtin{Name: "WaitGroup.add", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "WaitGroup.add requires a number argument")
			}
			if num, ok := args[0].(*object.Number); ok {
				wg.Add(int(num.Value))
			}
			return object.UNDEFINED
		}}),
		hashEntry("done", &object.Builtin{Name: "WaitGroup.done", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			wg.Done()
			return object.UNDEFINED
		}}),
		hashEntry("wait", &object.Builtin{Name: "WaitGroup.wait", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			wg.Wait()
			return object.UNDEFINED
		}}),
	)
}
