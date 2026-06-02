package stdlib

import (
	"io"
	"net"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func socketClientConnect(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "socket.connect requires host and port")
	}
	host, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "socket.connect: host must be a string")
	}
	port, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "socket.connect: port must be a number")
	}
	addr := host.Value + ":" + formatInt(int(port.Value))
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return object.NewError(pos, "socket.connect: %v", err)
	}
	return newSocketConnObject(conn)
}

func newSocketConnObject(conn net.Conn) object.Object {
	connObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(connObj, "_conn", &object.GoObject{Value: conn})
	setHashMember(connObj, "remoteAddr", &object.String{Value: conn.RemoteAddr().String()})
	setHashMember(connObj, "localAddr", &object.String{Value: conn.LocalAddr().String()})
	setHashMember(connObj, "write", &object.Builtin{
		Name: "socket.write",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "socket.write requires data")
			}
			data := []byte(args[0].Inspect())
			n, err := conn.Write(data)
			if err != nil {
				return object.NewError(pos, "socket.write: %v", err)
			}
			return &object.Number{Value: float64(n)}
		},
	})
	setHashMember(connObj, "send", &object.Builtin{
		Name: "socket.send",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "socket.send requires data")
			}
			data := []byte(args[0].Inspect())
			n, err := conn.Write(data)
			if err != nil {
				return object.NewError(pos, "socket.send: %v", err)
			}
			return &object.Number{Value: float64(n)}
		},
	})
	setHashMember(connObj, "read", &object.Builtin{
		Name: "socket.read",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			bufSize := 4096
			if len(args) >= 1 {
				if n, ok := args[0].(*object.Number); ok {
					bufSize = int(n.Value)
				}
			}
			buf := make([]byte, bufSize)
			n, err := conn.Read(buf)
			if n > 0 {
				return &object.String{Value: string(buf[:n])}
			}
			if err == io.EOF {
				return object.NULL
			}
			return object.NewError(pos, "socket.read: %v", err)
		},
	})
	setHashMember(connObj, "recv", &object.Builtin{
		Name: "socket.recv",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if n > 0 {
				return &object.String{Value: string(buf[:n])}
			}
			if err == io.EOF {
				return object.NULL
			}
			return object.NewError(pos, "socket.recv: %v", err)
		},
	})
	setHashMember(connObj, "close", &object.Builtin{
		Name: "socket.close",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			conn.Close()
			return object.UNDEFINED
		},
	})
	setHashMember(connObj, "setDeadline", &object.Builtin{
		Name: "socket.setDeadline",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "socket.setDeadline requires timeout in ms")
			}
			ms, ok := args[0].(*object.Number)
			if !ok {
				return object.NewError(pos, "socket.setDeadline: argument must be a number (ms)")
			}
			conn.SetDeadline(time.Now().Add(time.Duration(ms.Value) * time.Millisecond))
			return object.UNDEFINED
		},
	})
	return connObj
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
