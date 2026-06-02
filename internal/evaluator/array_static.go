package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerArray(env *object.Environment) {
	env.Set("Array", callableBuiltinObject("Array", builtinArrayConstructor, map[string]object.Object{
		"isArray": &object.Builtin{Name: "Array.isArray", Fn: builtinArrayIsArray},
		"of":      &object.Builtin{Name: "Array.of", Fn: builtinArrayOf},
		"from":    &object.Builtin{Name: "Array.from", Fn: builtinArrayFrom},
	}))
}

func builtinArrayConstructor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 1 {
		if n, ok := args[0].(*object.Number); ok {
			length := int(n.Value)
			if length < 0 {
				return object.NewError(pos, "RangeError: invalid array length")
			}
			elements := make([]object.Object, length)
			for i := range elements {
				elements[i] = object.UNDEFINED
			}
			return &object.Array{Elements: elements}
		}
	}
	elements := make([]object.Object, len(args))
	copy(elements, args)
	return &object.Array{Elements: elements}
}

func builtinArrayIsArray(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	_, ok := args[0].(*object.Array)
	return object.NativeBool(ok)
}

func builtinArrayOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	elements := make([]object.Object, len(args))
	copy(elements, args)
	return &object.Array{Elements: elements}
}

func builtinArrayFrom(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Array{Elements: nil}
	}
	var elements []object.Object
	switch src := args[0].(type) {
	case *object.Array:
		elements = make([]object.Object, len(src.Elements))
		copy(elements, src.Elements)
	case *object.String:
		for _, r := range src.Value {
			elements = append(elements, &object.String{Value: string(r)})
		}
	case *object.Hash:
		if lengthObj := getHashKey(src, &object.String{Value: "length"}); lengthObj != object.UNDEFINED {
			if length, ok := lengthObj.(*object.Number); ok {
				for i := 0; i < int(length.Value); i++ {
					value := getHashKey(src, &object.Number{Value: float64(i)})
					elements = append(elements, value)
				}
			}
		}
	default:
		return object.NewError(pos, "TypeError: Array.from requires array, string or array-like object")
	}
	if len(args) > 1 {
		mapFn := args[1]
		for i, elem := range elements {
			result := applyFunction(mapFn, env, []object.Object{elem, &object.Number{Value: float64(i)}}, pos)
			if object.IsRuntimeError(result) {
				return result
			}
			elements[i] = result
		}
	}
	return &object.Array{Elements: elements}
}
