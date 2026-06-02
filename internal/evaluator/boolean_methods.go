package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var booleanObjectMethods = map[string]object.BuiltinFunc{
	"valueOf":  builtinBooleanObjectValueOf,
	"toString": builtinBooleanObjectToString,
}

func constructBooleanObject(args []object.Object) *object.BooleanObject {
	value := false
	if len(args) > 0 {
		value = object.IsTruthy(args[0])
	}
	return &object.BooleanObject{Value: value}
}

func builtinBooleanObjectValueOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	box, ok := env.Extra.(*object.BooleanObject)
	if !ok {
		return object.NewError(pos, "TypeError: Boolean.valueOf requires Boolean receiver")
	}
	return object.NativeBool(box.Value)
}

func builtinBooleanObjectToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	box, ok := env.Extra.(*object.BooleanObject)
	if !ok {
		return object.NewError(pos, "TypeError: Boolean.toString requires Boolean receiver")
	}
	if box.Value {
		return &object.String{Value: "true"}
	}
	return &object.String{Value: "false"}
}
