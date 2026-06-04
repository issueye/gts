package stdlib

import (
	"mime"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/mime", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initMimeModule(exports)
		return exports, nil
	})
}

func initMimeModule(exports *object.Hash) {
	setHashMember(exports, "typeByExtension", &object.Builtin{Name: "mime.typeByExtension", Fn: mimeTypeByExtension})
	setHashMember(exports, "extensionByType", &object.Builtin{Name: "mime.extensionByType", Fn: mimeExtensionByType})
	setHashMember(exports, "parseMediaType", &object.Builtin{Name: "mime.parseMediaType", Fn: mimeParseMediaType})
	setHashMember(exports, "formatMediaType", &object.Builtin{Name: "mime.formatMediaType", Fn: mimeFormatMediaType})
}

func mimeTypeByExtension(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	ext, errObj := requiredString(pos, "mime.typeByExtension", args, 0, "extension")
	if errObj != nil {
		return errObj
	}
	value := mime.TypeByExtension(ext)
	if value == "" {
		return object.UNDEFINED
	}
	return &object.String{Value: value}
}

func mimeExtensionByType(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	mediaType, errObj := requiredString(pos, "mime.extensionByType", args, 0, "type")
	if errObj != nil {
		return errObj
	}
	exts, err := mime.ExtensionsByType(mediaType)
	if err != nil {
		return object.NewError(pos, "mime.extensionByType: %v", err)
	}
	if len(exts) == 0 {
		return object.UNDEFINED
	}
	return &object.String{Value: exts[0]}
}

func mimeParseMediaType(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "mime.parseMediaType", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil {
		return object.NewError(pos, "mime.parseMediaType: %v", err)
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: mediaType})
	setHashMember(out, "params", goValueToObject(params))
	return out
}

func mimeFormatMediaType(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	mediaType, errObj := requiredString(pos, "mime.formatMediaType", args, 0, "type")
	if errObj != nil {
		return errObj
	}
	params := map[string]string{}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "mime.formatMediaType: params must be an object")
		}
		for _, pair := range hash.OrderedPairs() {
			params[pair.Key.Inspect()] = pair.Value.Inspect()
		}
	}
	value := mime.FormatMediaType(mediaType, params)
	if value == "" {
		return object.NewError(pos, "mime.formatMediaType: invalid media type")
	}
	return &object.String{Value: value}
}
