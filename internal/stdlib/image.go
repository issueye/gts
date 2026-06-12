package stdlib

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/image", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "info", &object.Builtin{Name: "image.info", Fn: imageInfo})
		return exports, nil
	})
}

func imageInfo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "image.info requires path")
	}
	return object.NewError(pos, "image module: basic placeholder - full implementation requires external library")
}
