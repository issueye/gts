package stdlib

import (
	"encoding/base64"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/encoding/base64", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initEncodingBase64Module(exports)
		return exports, nil
	})
}

func initEncodingBase64Module(exports *object.Hash) {
	setHashMember(exports, "encode", &object.Builtin{Name: "base64.encode", Fn: base64Encode})
	setHashMember(exports, "decode", &object.Builtin{Name: "base64.decode", Fn: base64Decode})
	setHashMember(exports, "encodeURL", &object.Builtin{Name: "base64.encodeURL", Fn: base64EncodeURL})
	setHashMember(exports, "decodeURL", &object.Builtin{Name: "base64.decodeURL", Fn: base64DecodeURL})
}

func base64Encode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return base64EncodeWith(pos, "base64.encode", base64.StdEncoding, args...)
}

func base64Decode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return base64DecodeWith(pos, "base64.decode", base64.StdEncoding, args...)
}

func base64EncodeURL(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return base64EncodeWith(pos, "base64.encodeURL", base64.RawURLEncoding, args...)
}

func base64DecodeURL(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return base64DecodeWith(pos, "base64.decodeURL", base64.RawURLEncoding, args...)
}

func base64EncodeWith(pos ast.Position, name string, encoding *base64.Encoding, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "%s requires value", name)
	}
	data, errObj := bufferBytesFromObject(pos, name, args[0], "utf8")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: encoding.EncodeToString(data)}
}

func base64DecodeWith(pos ast.Position, name string, encoding *base64.Encoding, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, name, args, 0, "text")
	if errObj != nil {
		return errObj
	}
	data, err := encoding.DecodeString(text)
	if err != nil {
		return object.NewError(pos, "%s: invalid base64 data", name)
	}
	if wantsBuffer(args, 1) {
		return newBufferObject(data)
	}
	return &object.String{Value: string(data)}
}

func wantsBuffer(args []object.Object, index int) bool {
	if len(args) <= index {
		return false
	}
	opts, ok := args[index].(*object.Hash)
	if !ok {
		return false
	}
	if value, ok := hashValue(opts, "asBuffer"); ok {
		if b, ok := value.(*object.Boolean); ok {
			return b.Value
		}
	}
	return false
}
