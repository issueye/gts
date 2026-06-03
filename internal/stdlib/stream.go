package stdlib

import (
	"bufio"
	"io"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type readableStream struct {
	reader *bufio.Reader
	closer io.Closer
	closed bool
}

func init() {
	module.RegisterNative("@std/stream", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initStreamModule(exports)
		return exports, nil
	})
}

func initStreamModule(exports *object.Hash) {
	setHashMember(exports, "fromString", &object.Builtin{Name: "stream.fromString", Fn: streamFromString})
}

func streamFromString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "stream.fromString", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	return newReadableStream(strings.NewReader(text), nil)
}

func newReadableStream(reader io.Reader, closer io.Closer) *object.Hash {
	stream := &readableStream{reader: bufio.NewReader(reader), closer: closer}
	return readableStreamObject(stream)
}

func readableStreamObject(stream *readableStream) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__stream", &object.GoObject{Value: stream})
	setHashMember(obj, "read", &object.Builtin{Name: "stream.read", Fn: streamRead, Extra: &object.GoObject{Value: stream}})
	setHashMember(obj, "readText", &object.Builtin{Name: "stream.readText", Fn: streamReadText, Extra: &object.GoObject{Value: stream}})
	setHashMember(obj, "readLine", &object.Builtin{Name: "stream.readLine", Fn: streamReadLine, Extra: &object.GoObject{Value: stream}})
	setHashMember(obj, "readAll", &object.Builtin{Name: "stream.readAll", Fn: streamReadAll, Extra: &object.GoObject{Value: stream}})
	setHashMember(obj, "close", &object.Builtin{Name: "stream.close", Fn: streamClose, Extra: &object.GoObject{Value: stream}})
	return obj
}

func streamRead(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stream, errObj := boundStream(pos, env, "stream.read")
	if errObj != nil {
		return errObj
	}
	size := 8192
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "stream.read: size must be a number")
		}
		size = int(n.Value)
		if size < 1 {
			return object.NewError(pos, "stream.read: size must be positive")
		}
	}
	buf := make([]byte, size)
	n, err := stream.reader.Read(buf)
	if err == io.EOF {
		return object.NULL
	}
	if err != nil {
		return object.NewError(pos, "stream.read: %v", err)
	}
	elements := make([]object.Object, n)
	for i := 0; i < n; i++ {
		elements[i] = &object.Number{Value: float64(buf[i])}
	}
	return &object.Array{Elements: elements}
}

func streamReadText(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stream, errObj := boundStream(pos, env, "stream.readText")
	if errObj != nil {
		return errObj
	}
	size := 8192
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "stream.readText: size must be a number")
		}
		size = int(n.Value)
		if size < 1 {
			return object.NewError(pos, "stream.readText: size must be positive")
		}
	}
	buf := make([]byte, size)
	n, err := stream.reader.Read(buf)
	if err == io.EOF {
		return object.NULL
	}
	if err != nil {
		return object.NewError(pos, "stream.readText: %v", err)
	}
	return &object.String{Value: string(buf[:n])}
}

func streamReadLine(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stream, errObj := boundStream(pos, env, "stream.readLine")
	if errObj != nil {
		return errObj
	}
	line, err := stream.reader.ReadString('\n')
	if err == io.EOF {
		if line == "" {
			return object.NULL
		}
		return &object.String{Value: strings.TrimRight(line, "\r\n")}
	}
	if err != nil {
		return object.NewError(pos, "stream.readLine: %v", err)
	}
	return &object.String{Value: strings.TrimRight(line, "\r\n")}
}

func streamReadAll(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stream, errObj := boundStream(pos, env, "stream.readAll")
	if errObj != nil {
		return errObj
	}
	data, err := io.ReadAll(stream.reader)
	if err != nil {
		return object.NewError(pos, "stream.readAll: %v", err)
	}
	return &object.String{Value: string(data)}
}

func streamClose(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	stream, errObj := boundStream(pos, env, "stream.close")
	if errObj != nil {
		return errObj
	}
	if stream.closed {
		return object.UNDEFINED
	}
	stream.closed = true
	if stream.closer != nil {
		if err := stream.closer.Close(); err != nil {
			return object.NewError(pos, "stream.close: %v", err)
		}
	}
	return object.UNDEFINED
}

func boundStream(pos ast.Position, env *object.Environment, name string) (*readableStream, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing stream receiver", name)
	}
	stream, ok := goObj.Value.(*readableStream)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid stream receiver", name)
	}
	return stream, nil
}
