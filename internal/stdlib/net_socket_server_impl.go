package stdlib

import (
	"net"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func socketServerListen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "socket.listen requires a port and connection handler")
	}
	port, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "socket.listen: first argument must be a port number")
	}
	handler, ok := args[1].(*object.Function)
	if !ok {
		return object.NewError(pos, "socket.listen: second argument must be a function")
	}
	addr := ":" + formatInt(int(port.Value))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return object.NewError(pos, "socket.listen: %v", err)
	}
	serverObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(serverObj, "port", &object.Number{Value: float64(int(port.Value))})
	setHashMember(serverObj, "address", &object.String{Value: addr})
	setHashMember(serverObj, "close", &object.Builtin{
		Name: "server.close",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			listener.Close()
			return object.UNDEFINED
		},
	})
	object.Spawn(func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connObj := newSocketConnObject(conn)
			scope := handler.Env.NewScope()
			if len(handler.Parameters) > 0 {
				scope.Set(handler.Parameters[0].Name, connObj)
			}
			object.Spawn(func() {
				object.EvalFn(handler.Body, scope)
			})
		}
	})
	return serverObj
}
