package evaluator

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func builtinNativeToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value := env.Extra
	if value == nil {
		return object.NewError(pos, "TypeError: toString requires a receiver")
	}
	return &object.String{Value: nativeToString(value)}
}

func nativeToString(value object.Object) string {
	switch v := value.(type) {
	case *object.String:
		return v.Value
	case *object.Array:
		return joinArrayElements(v, ",")
	default:
		return value.Inspect()
	}
}

func joinArrayElements(arr *object.Array, sep string) string {
	parts := make([]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		parts[i] = elem.Inspect()
	}
	return strings.Join(parts, sep)
}
