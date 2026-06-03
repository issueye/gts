package typechecker

import (
	"math"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// Check verifies that value satisfies anno. A nil annotation is treated as any.
func Check(pos ast.Position, anno *ast.TypeAnnotation, value object.Object) *object.Error {
	if Matches(anno, value) {
		return nil
	}
	return object.NewError(pos, "TypeError: expected %s, got %s", typeName(anno), valueName(value))
}

// Matches reports whether value satisfies anno.
func Matches(anno *ast.TypeAnnotation, value object.Object) bool {
	if anno == nil {
		return true
	}
	if anno.Optional && isNilLike(value) {
		return true
	}
	switch anno.Kind {
	case ast.TK_PRIMITIVE:
		return matchesPrimitive(anno.Name, value)
	case ast.TK_ARRAY:
		arr, ok := value.(*object.Array)
		if !ok {
			return false
		}
		for _, elem := range arr.Elements {
			if !Matches(anno.ArrayOf, elem) {
				return false
			}
		}
		return true
	case ast.TK_UNION:
		for _, item := range anno.Union {
			if item == anno {
				if matchesPrimitive(anno.Name, value) {
					return true
				}
				continue
			}
			if Matches(item, value) {
				return true
			}
		}
		return false
	case ast.TK_OBJECT:
		hash, ok := value.(*object.Hash)
		if !ok {
			return false
		}
		for key, propType := range anno.Properties {
			prop, ok := hash.Pairs[object.HashKey{Type: object.STRING_OBJ, Value: key}]
			if !ok || !Matches(propType, prop.Value) {
				return false
			}
		}
		return true
	case ast.TK_FUNCTION:
		return value != nil && value.Type() == object.FUNCTION_OBJ
	default:
		return true
	}
}

func matchesPrimitive(name string, value object.Object) bool {
	switch strings.ToLower(name) {
	case "", "any":
		return true
	case "number", "float":
		return value != nil && value.Type() == object.NUMBER_OBJ
	case "int":
		num, ok := value.(*object.Number)
		return ok && math.Trunc(num.Value) == num.Value
	case "string":
		return value != nil && value.Type() == object.STRING_OBJ
	case "boolean", "bool":
		return value != nil && value.Type() == object.BOOLEAN_OBJ
	case "null":
		return value != nil && value.Type() == object.NULL_OBJ
	case "undefined", "void":
		return value != nil && value.Type() == object.UNDEFINED_OBJ
	case "object":
		return value != nil && value.Type() == object.OBJECT_OBJ
	case "array":
		return value != nil && value.Type() == object.ARRAY_OBJ
	case "function":
		return value != nil && value.Type() == object.FUNCTION_OBJ
	default:
		return true
	}
}

func isNilLike(value object.Object) bool {
	if value == nil {
		return true
	}
	switch value.Type() {
	case object.NULL_OBJ, object.UNDEFINED_OBJ:
		return true
	default:
		return false
	}
}

func typeName(anno *ast.TypeAnnotation) string {
	if anno == nil {
		return "any"
	}
	if anno.Kind == ast.TK_UNION {
		parts := make([]string, 0, len(anno.Union))
		for _, item := range anno.Union {
			if item == anno {
				parts = append(parts, anno.Name)
				continue
			}
			parts = append(parts, typeName(item))
		}
		return strings.Join(parts, " | ")
	}
	if anno.Kind == ast.TK_ARRAY {
		return typeName(anno.ArrayOf) + "[]"
	}
	if anno.Optional {
		return anno.Name + "?"
	}
	return anno.String()
}

func valueName(value object.Object) string {
	if value == nil {
		return "undefined"
	}
	return strings.ToLower(strings.TrimSuffix(string(value.Type()), "_OBJ"))
}
