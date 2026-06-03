package stdlib

import (
	"encoding/hex"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/encoding/hex", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initEncodingHexModule(exports)
		return exports, nil
	})
}

func initEncodingHexModule(exports *object.Hash) {
	setHashMember(exports, "encode", &object.Builtin{Name: "hex.encode", Fn: hexEncode})
	setHashMember(exports, "decode", &object.Builtin{Name: "hex.decode", Fn: hexDecode})
}

func hexEncode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "hex.encode requires value")
	}
	data, errObj := bufferBytesFromObject(pos, "hex.encode", args[0], "utf8")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: hex.EncodeToString(data)}
}

func hexDecode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "hex.decode", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	data, err := hex.DecodeString(text)
	if err != nil {
		return object.NewError(pos, "hex.decode: invalid hex data")
	}
	if wantsBuffer(args, 1) {
		return newBufferObject(data)
	}
	return &object.String{Value: string(data)}
}
