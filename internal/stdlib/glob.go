package stdlib

import (
	"path/filepath"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/glob", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "match", &object.Builtin{Name: "glob.match", Fn: globMatch})
		setHashMember(exports, "find", &object.Builtin{Name: "glob.find", Fn: globFind})
		return exports, nil
	})
}

func globMatch(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "glob.match requires pattern and path")
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "glob.match expects string pattern")
	}
	path, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "glob.match expects string path")
	}

	matched, err := filepath.Match(pattern.Value, path.Value)
	if err != nil {
		return object.NewError(pos, "glob.match: %v", err)
	}
	return &object.Boolean{Value: matched}
}

func globFind(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "glob.find requires pattern")
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "glob.find expects string pattern")
	}

	matches, err := filepath.Glob(pattern.Value)
	if err != nil {
		return object.NewError(pos, "glob.find: %v", err)
	}

	elements := make([]object.Object, len(matches))
	for i, match := range matches {
		elements[i] = &object.String{Value: match}
	}
	return &object.Array{Elements: elements}
}
