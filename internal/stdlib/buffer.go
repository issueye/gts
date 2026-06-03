package stdlib

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

const bufferDataKey = "__buffer"

func init() {
	module.RegisterNative("@std/buffer", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initBufferModule(exports)
		return exports, nil
	})
}

func initBufferModule(exports *object.Hash) {
	setHashMember(exports, "from", &object.Builtin{Name: "buffer.from", Fn: bufferFrom})
	setHashMember(exports, "alloc", &object.Builtin{Name: "buffer.alloc", Fn: bufferAlloc})
	setHashMember(exports, "byteLength", &object.Builtin{Name: "buffer.byteLength", Fn: bufferByteLength})
	setHashMember(exports, "concat", &object.Builtin{Name: "buffer.concat", Fn: bufferConcat})
	setHashMember(exports, "isBuffer", &object.Builtin{Name: "buffer.isBuffer", Fn: bufferIsBuffer})
}

func bufferFrom(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "buffer.from requires value")
	}
	encoding, errObj := optionalBufferEncoding(pos, "buffer.from", args, 1)
	if errObj != nil {
		return errObj
	}
	data, errObj := bufferBytesFromObject(pos, "buffer.from", args[0], encoding)
	if errObj != nil {
		return errObj
	}
	return newBufferObject(data)
}

func bufferAlloc(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	size, errObj := requiredInt(pos, "buffer.alloc", args, 0, "size")
	if errObj != nil {
		return errObj
	}
	if size < 0 {
		return object.NewError(pos, "buffer.alloc: size must be non-negative")
	}
	data := make([]byte, size)
	if len(args) >= 2 && size > 0 {
		fill, errObj := bufferFillBytes(pos, args[1])
		if errObj != nil {
			return errObj
		}
		for i := range data {
			data[i] = fill[i%len(fill)]
		}
	}
	return newBufferObject(data)
}

func bufferByteLength(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "buffer.byteLength requires value")
	}
	encoding, errObj := optionalBufferEncoding(pos, "buffer.byteLength", args, 1)
	if errObj != nil {
		return errObj
	}
	data, errObj := bufferBytesFromObject(pos, "buffer.byteLength", args[0], encoding)
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(len(data))}
}

func bufferConcat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "buffer.concat requires buffers")
	}
	buffers, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "buffer.concat: buffers must be an array")
	}
	total := 0
	parts := make([][]byte, len(buffers.Elements))
	for i, item := range buffers.Elements {
		data, ok := bufferData(item)
		if !ok {
			return object.NewError(pos, "buffer.concat: buffers[%d] must be a Buffer", i)
		}
		parts[i] = data
		total += len(data)
	}
	out := make([]byte, 0, total)
	for _, part := range parts {
		out = append(out, part...)
	}
	return newBufferObject(out)
}

func bufferIsBuffer(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	_, ok := bufferData(args[0])
	return object.NativeBool(ok)
}

func bufferToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := boundBuffer(pos, env, "buffer.toString")
	if errObj != nil {
		return errObj
	}
	encoding, errObj := optionalBufferEncoding(pos, "buffer.toString", args, 0)
	if errObj != nil {
		return errObj
	}
	text, errObj := encodeBufferBytes(pos, "buffer.toString", data, encoding)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: text}
}

func bufferToArray(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := boundBuffer(pos, env, "buffer.toArray")
	if errObj != nil {
		return errObj
	}
	elements := make([]object.Object, len(data))
	for i, b := range data {
		elements[i] = &object.Number{Value: float64(b)}
	}
	return &object.Array{Elements: elements}
}

func bufferSlice(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := boundBuffer(pos, env, "buffer.slice")
	if errObj != nil {
		return errObj
	}
	start := 0
	end := len(data)
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "buffer.slice: start must be a number")
		}
		start = normalizeBufferIndex(int(n.Value), len(data))
	}
	if len(args) >= 2 {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "buffer.slice: end must be a number")
		}
		end = normalizeBufferIndex(int(n.Value), len(data))
	}
	if end < start {
		end = start
	}
	return newBufferObject(data[start:end])
}

func newBufferObject(data []byte) *object.Hash {
	buf := append([]byte(nil), data...)
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	holder := &object.GoObject{Value: buf}
	setHashMember(obj, bufferDataKey, holder)
	setHashMember(obj, "length", &object.Number{Value: float64(len(buf))})
	setHashMember(obj, "toString", &object.Builtin{Name: "buffer.toString", Fn: bufferToString, Extra: holder})
	setHashMember(obj, "toArray", &object.Builtin{Name: "buffer.toArray", Fn: bufferToArray, Extra: holder})
	setHashMember(obj, "slice", &object.Builtin{Name: "buffer.slice", Fn: bufferSlice, Extra: holder})
	return obj
}

func bufferData(value object.Object) ([]byte, bool) {
	hash, ok := value.(*object.Hash)
	if !ok {
		return nil, false
	}
	raw, ok := hashValue(hash, bufferDataKey)
	if !ok {
		return nil, false
	}
	goObj, ok := raw.(*object.GoObject)
	if !ok {
		return nil, false
	}
	data, ok := goObj.Value.([]byte)
	if !ok {
		return nil, false
	}
	return data, true
}

func boundBuffer(pos ast.Position, env *object.Environment, name string) ([]byte, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing buffer receiver", name)
	}
	data, ok := goObj.Value.([]byte)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid buffer receiver", name)
	}
	return data, nil
}

func bufferBytesFromObject(pos ast.Position, name string, value object.Object, encoding string) ([]byte, *object.Error) {
	switch v := value.(type) {
	case *object.String:
		return decodeBufferString(pos, name, v.Value, encoding)
	case *object.Array:
		out := make([]byte, len(v.Elements))
		for i, item := range v.Elements {
			n, ok := item.(*object.Number)
			if !ok {
				return nil, object.NewError(pos, "%s: array item %d must be a number", name, i)
			}
			out[i] = byte(int(n.Value) & 0xff)
		}
		return out, nil
	default:
		if data, ok := bufferData(value); ok {
			return append([]byte(nil), data...), nil
		}
		return nil, object.NewError(pos, "%s: value must be a string, array, or Buffer", name)
	}
}

func bufferFillBytes(pos ast.Position, value object.Object) ([]byte, *object.Error) {
	switch v := value.(type) {
	case *object.Number:
		return []byte{byte(int(v.Value) & 0xff)}, nil
	case *object.String:
		data, errObj := decodeBufferString(pos, "buffer.alloc", v.Value, "utf8")
		if errObj != nil {
			return nil, errObj
		}
		if len(data) == 0 {
			return []byte{0}, nil
		}
		return data, nil
	default:
		if data, ok := bufferData(value); ok {
			if len(data) == 0 {
				return []byte{0}, nil
			}
			return data, nil
		}
		return nil, object.NewError(pos, "buffer.alloc: fill must be a number, string, or Buffer")
	}
}

func optionalBufferEncoding(pos ast.Position, name string, args []object.Object, index int) (string, *object.Error) {
	if len(args) <= index {
		return "utf8", nil
	}
	encoding, ok := args[index].(*object.String)
	if !ok {
		return "", object.NewError(pos, "%s: encoding must be a string", name)
	}
	normalized := normalizeBufferEncoding(encoding.Value)
	if normalized == "" {
		return "", object.NewError(pos, "%s: unsupported encoding %q", name, encoding.Value)
	}
	return normalized, nil
}

func normalizeBufferEncoding(encoding string) string {
	switch strings.ToLower(strings.ReplaceAll(encoding, "-", "")) {
	case "", "utf8", "utf":
		return "utf8"
	case "hex":
		return "hex"
	case "base64":
		return "base64"
	default:
		return ""
	}
}

func decodeBufferString(pos ast.Position, name, value, encoding string) ([]byte, *object.Error) {
	switch normalizeBufferEncoding(encoding) {
	case "utf8":
		return []byte(value), nil
	case "hex":
		data, err := hex.DecodeString(value)
		if err != nil {
			return nil, object.NewError(pos, "%s: invalid hex data", name)
		}
		return data, nil
	case "base64":
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return nil, object.NewError(pos, "%s: invalid base64 data", name)
		}
		return data, nil
	default:
		return nil, object.NewError(pos, "%s: unsupported encoding %q", name, encoding)
	}
}

func encodeBufferBytes(pos ast.Position, name string, data []byte, encoding string) (string, *object.Error) {
	switch normalizeBufferEncoding(encoding) {
	case "utf8":
		if !utf8.Valid(data) {
			return string(data), nil
		}
		return string(data), nil
	case "hex":
		return hex.EncodeToString(data), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(data), nil
	default:
		return "", object.NewError(pos, "%s: unsupported encoding %q", name, encoding)
	}
}

func normalizeBufferIndex(index, length int) int {
	if index < 0 {
		index += length
	}
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}
