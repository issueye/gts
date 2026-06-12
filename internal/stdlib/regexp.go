package stdlib

import (
	"regexp"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/regexp", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "escape", &object.Builtin{Name: "regexp.escape", Fn: regexpEscape})
		setHashMember(exports, "matchAll", &object.Builtin{Name: "regexp.matchAll", Fn: regexpMatchAll})
		setHashMember(exports, "split", &object.Builtin{Name: "regexp.split", Fn: regexpSplit})
		return exports, nil
	})
}

func regexpEscape(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "regexp.escape requires string")
	}
	str, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "regexp.escape expects string")
	}
	return &object.String{Value: regexp.QuoteMeta(str.Value)}
}

func regexpMatchAll(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "regexp.matchAll requires pattern and string")
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "regexp.matchAll expects string pattern")
	}
	str, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "regexp.matchAll expects string")
	}

	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return object.NewError(pos, "regexp.matchAll: %v", err)
	}

	matches := re.FindAllStringSubmatch(str.Value, -1)
	result := &object.Array{Elements: make([]object.Object, len(matches))}
	for i, match := range matches {
		matchArr := &object.Array{Elements: make([]object.Object, len(match))}
		for j, m := range match {
			matchArr.Elements[j] = &object.String{Value: m}
		}
		result.Elements[i] = matchArr
	}
	return result
}

func regexpSplit(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "regexp.split requires pattern and string")
	}
	pattern, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "regexp.split expects string pattern")
	}
	str, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "regexp.split expects string")
	}

	limit := -1
	if len(args) > 2 {
		if n, ok := args[2].(*object.Number); ok {
			limit = int(n.Value)
		}
	}

	re, err := regexp.Compile(pattern.Value)
	if err != nil {
		return object.NewError(pos, "regexp.split: %v", err)
	}

	parts := re.Split(str.Value, limit)
	result := &object.Array{Elements: make([]object.Object, len(parts))}
	for i, part := range parts {
		result.Elements[i] = &object.String{Value: part}
	}
	return result
}
