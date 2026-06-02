package stdlib

import (
	"os"
	"path/filepath"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/path", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initPathModule(exports)
		return exports, nil
	})
}

func initPathModule(exports *object.Hash) {
	setHashMember(exports, "join", &object.Builtin{Name: "path.join", Fn: pathJoin})
	setHashMember(exports, "resolve", &object.Builtin{Name: "path.resolve", Fn: pathResolve})
	setHashMember(exports, "relative", &object.Builtin{Name: "path.relative", Fn: pathRelative})
	setHashMember(exports, "normalize", &object.Builtin{Name: "path.normalize", Fn: pathNormalize})
	setHashMember(exports, "dirname", &object.Builtin{Name: "path.dirname", Fn: pathDirname})
	setHashMember(exports, "basename", &object.Builtin{Name: "path.basename", Fn: pathBasename})
	setHashMember(exports, "extname", &object.Builtin{Name: "path.extname", Fn: pathExtname})
	setHashMember(exports, "sep", &object.String{Value: string(os.PathSeparator)})
}

func pathJoin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	parts, err := stringArgs("path.join", args)
	if err != "" {
		return object.NewError(pos, "%s", err)
	}
	return &object.String{Value: filepath.Join(parts...)}
}

func pathResolve(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	parts, err := stringArgs("path.resolve", args)
	if err != "" {
		return object.NewError(pos, "%s", err)
	}
	joined := filepath.Join(parts...)
	if joined == "" {
		joined = "."
	}
	abs, goErr := filepath.Abs(joined)
	if goErr != nil {
		return object.NewError(pos, "path.resolve: %v", goErr)
	}
	return &object.String{Value: abs}
}

func pathRelative(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) != 2 {
		return object.NewError(pos, "path.relative requires from and to paths")
	}
	from, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "path.relative: first argument must be a string")
	}
	to, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "path.relative: second argument must be a string")
	}
	rel, err := filepath.Rel(from.Value, to.Value)
	if err != nil {
		return object.NewError(pos, "path.relative: %v", err)
	}
	return &object.String{Value: rel}
}

func pathNormalize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "path.normalize requires a path")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "path.normalize: argument must be a string")
	}
	return &object.String{Value: filepath.Clean(s.Value)}
}

func pathDirname(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "path.dirname requires a path")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "path.dirname: argument must be a string")
	}
	return &object.String{Value: filepath.Dir(s.Value)}
}

func pathBasename(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "path.basename requires a path")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "path.basename: argument must be a string")
	}
	return &object.String{Value: filepath.Base(s.Value)}
}

func pathExtname(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "path.extname requires a path")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "path.extname: argument must be a string")
	}
	return &object.String{Value: filepath.Ext(s.Value)}
}

func stringArgs(name string, args []object.Object) ([]string, string) {
	parts := make([]string, len(args))
	for i, arg := range args {
		s, ok := arg.(*object.String)
		if !ok {
			return nil, name + ": all arguments must be strings"
		}
		parts[i] = s.Value
	}
	return parts, ""
}
