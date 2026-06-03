package stdlib

import (
	"os"
	"path/filepath"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/path", func(env *object.Environment) (object.Object, error) {
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
	setHashMember(exports, "isAbs", &object.Builtin{Name: "path.isAbs", Fn: pathIsAbs})
	setHashMember(exports, "toSlash", &object.Builtin{Name: "path.toSlash", Fn: pathToSlash})
	setHashMember(exports, "fromSlash", &object.Builtin{Name: "path.fromSlash", Fn: pathFromSlash})
	setHashMember(exports, "matches", &object.Builtin{Name: "path.matches", Fn: pathMatches})
	setHashMember(exports, "parse", &object.Builtin{Name: "path.parse", Fn: pathParse})
	setHashMember(exports, "format", &object.Builtin{Name: "path.format", Fn: pathFormat})
	setHashMember(exports, "splitList", &object.Builtin{Name: "path.splitList", Fn: pathSplitList})
	setHashMember(exports, "sep", &object.String{Value: string(os.PathSeparator)})
	setHashMember(exports, "delimiter", &object.String{Value: string(os.PathListSeparator)})
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

func pathIsAbs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s, errObj := requiredString(pos, "path.isAbs", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	return object.NativeBool(filepath.IsAbs(s))
}

func pathToSlash(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s, errObj := requiredString(pos, "path.toSlash", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: filepath.ToSlash(s)}
}

func pathFromSlash(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s, errObj := requiredString(pos, "path.fromSlash", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: filepath.FromSlash(s)}
}

func pathMatches(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	pattern, errObj := requiredString(pos, "path.matches", args, 0, "pattern")
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "path.matches", args, 1, "path")
	if errObj != nil {
		return errObj
	}
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return object.NewError(pos, "path.matches: %v", err)
	}
	return object.NativeBool(matched)
}

func pathParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s, errObj := requiredString(pos, "path.parse", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	dir := filepath.Dir(s)
	base := filepath.Base(s)
	ext := filepath.Ext(s)
	name := base[:len(base)-len(ext)]
	root := ""
	volume := filepath.VolumeName(s)
	if volume != "" {
		root = volume + string(os.PathSeparator)
	} else if filepath.IsAbs(s) {
		root = string(os.PathSeparator)
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "root", &object.String{Value: root})
	setHashMember(out, "dir", &object.String{Value: dir})
	setHashMember(out, "base", &object.String{Value: base})
	setHashMember(out, "name", &object.String{Value: name})
	setHashMember(out, "ext", &object.String{Value: ext})
	return out
}

func pathFormat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "path.format requires a path object")
	}
	opts, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "path.format: path object must be an object")
	}
	if dir, ok := hashString(opts, "dir"); ok && dir != "" {
		if base, ok := hashString(opts, "base"); ok && base != "" {
			return &object.String{Value: filepath.Join(dir, base)}
		}
		name, _ := hashString(opts, "name")
		ext, _ := hashString(opts, "ext")
		return &object.String{Value: filepath.Join(dir, name+ext)}
	}
	root, _ := hashString(opts, "root")
	if base, ok := hashString(opts, "base"); ok && base != "" {
		return &object.String{Value: filepath.Join(root, base)}
	}
	name, _ := hashString(opts, "name")
	ext, _ := hashString(opts, "ext")
	return &object.String{Value: filepath.Join(root, name+ext)}
}

func pathSplitList(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s, errObj := requiredString(pos, "path.splitList", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	return strSliceToArray(filepath.SplitList(s))
}

func hashString(hash *object.Hash, key string) (string, bool) {
	value, ok := hashValue(hash, key)
	if !ok {
		return "", false
	}
	s, ok := value.(*object.String)
	if !ok {
		return "", false
	}
	return s.Value, true
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
