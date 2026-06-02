package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerObject(env *object.Environment) {
	env.Set("Object", &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hk("keys"):    {Key: &object.String{Value: "keys"}, Value: &object.Builtin{Name: "Object.keys", Fn: builtinObjKeys}},
			hk("values"):  {Key: &object.String{Value: "values"}, Value: &object.Builtin{Name: "Object.values", Fn: builtinObjValues}},
			hk("entries"): {Key: &object.String{Value: "entries"}, Value: &object.Builtin{Name: "Object.entries", Fn: builtinObjEntries}},
			hk("assign"):  {Key: &object.String{Value: "assign"}, Value: &object.Builtin{Name: "Object.assign", Fn: builtinObjAssign}},
			hk("hasOwn"):  {Key: &object.String{Value: "hasOwn"}, Value: &object.Builtin{Name: "Object.hasOwn", Fn: builtinObjHasOwn}},
		},
	})
}

func builtinObjKeys(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.keys requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.Pairs {
			elems = append(elems, pair.Key)
		}
		return &object.Array{Elements: elems}
	case *object.Instance:
		elems := make([]object.Object, 0, len(v.Props))
		for k := range v.Props {
			elems = append(elems, &object.String{Value: k})
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjValues(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.values requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.Pairs {
			elems = append(elems, pair.Value)
		}
		return &object.Array{Elements: elems}
	case *object.Instance:
		elems := make([]object.Object, 0, len(v.Props))
		for _, val := range v.Props {
			elems = append(elems, val)
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjEntries(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.entries requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.Pairs {
			entry := &object.Array{Elements: []object.Object{pair.Key, pair.Value}}
			elems = append(elems, entry)
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjAssign(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.assign requires at least 1 argument")
	}
	target, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "Object.assign target must be an object")
	}
	for _, src := range args[1:] {
		srcHash, ok := src.(*object.Hash)
		if !ok {
			continue
		}
		for k, v := range srcHash.Pairs {
			target.Pairs[k] = v
		}
	}
	return target
}

func builtinObjHasOwn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.FALSE
	}
	name, ok := args[1].(*object.String)
	if !ok {
		return object.FALSE
	}
	switch v := args[0].(type) {
	case *object.Hash:
		key := hashKey(name)
		_, ok := v.Pairs[key]
		return object.NativeBool(ok)
	case *object.Instance:
		_, ok := v.Props[name.Value]
		return object.NativeBool(ok)
	}
	return object.FALSE
}
