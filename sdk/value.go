package sdk

import (
	"fmt"
	"sort"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// Value is a Go-facing alias for a GoScript runtime value.
type Value = object.Object

// Position identifies where a host call originated in script source.
type Position = ast.Position

// Undefined returns the GoScript undefined singleton.
func Undefined() Value { return object.UNDEFINED }

// Null returns the GoScript null singleton.
func Null() Value { return object.NULL }

// String creates a GoScript string.
func String(value string) Value { return &object.String{Value: value} }

// Number creates a GoScript number.
func Number(value float64) Value { return &object.Number{Value: value} }

// Bool creates a GoScript boolean.
func Bool(value bool) Value { return object.NativeBool(value) }

// Array creates a GoScript array from values.
func Array(values ...Value) Value {
	return &object.Array{Elements: append([]object.Object{}, values...)}
}

// Object creates a GoScript object from a string-keyed map.
func Object(values map[string]Value) Value {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair, len(values))}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out.SetMember(&object.String{Value: key}, values[key])
	}
	return out
}

// GoObject wraps an opaque Go value for methods that know how to unwrap it.
func GoObject(value any) Value { return &object.GoObject{Value: value} }

// RuntimeError is a Go error that carries a GoScript error name.
type RuntimeError struct {
	Name    string
	Message string
}

func (e RuntimeError) Error() string {
	if e.Name == "" {
		return e.Message
	}
	return e.Name + ": " + e.Message
}

// Error creates a runtime Error value with the given name and message.
func Error(pos Position, name, message string) *object.Error {
	return object.NewNamedError(pos, name, message)
}

// Errorf creates a named Go error that RegisterModule converts into a GoScript
// runtime Error at the call site.
func Errorf(name, format string, args ...any) error {
	return RuntimeError{Name: name, Message: fmt.Sprintf(format, args...)}
}

// ToValue converts common Go values into GoScript values.
func ToValue(value any) (Value, error) {
	switch v := value.(type) {
	case nil:
		return object.NULL, nil
	case object.Object:
		return v, nil
	case string:
		return &object.String{Value: v}, nil
	case bool:
		return object.NativeBool(v), nil
	case int:
		return &object.Number{Value: float64(v)}, nil
	case int8:
		return &object.Number{Value: float64(v)}, nil
	case int16:
		return &object.Number{Value: float64(v)}, nil
	case int32:
		return &object.Number{Value: float64(v)}, nil
	case int64:
		return &object.Number{Value: float64(v)}, nil
	case uint:
		return &object.Number{Value: float64(v)}, nil
	case uint8:
		return &object.Number{Value: float64(v)}, nil
	case uint16:
		return &object.Number{Value: float64(v)}, nil
	case uint32:
		return &object.Number{Value: float64(v)}, nil
	case uint64:
		if v > 9007199254740991 {
			return nil, fmt.Errorf("uint64 value %d exceeds safe integer range", v)
		}
		return &object.Number{Value: float64(v)}, nil
	case float32:
		return &object.Number{Value: float64(v)}, nil
	case float64:
		return &object.Number{Value: v}, nil
	case []object.Object:
		return &object.Array{Elements: append([]object.Object{}, v...)}, nil
	case []any:
		items := make([]object.Object, len(v))
		for i, item := range v {
			converted, err := ToValue(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return &object.Array{Elements: items}, nil
	case map[string]object.Object:
		values := make(map[string]Value, len(v))
		for key, item := range v {
			values[key] = item
		}
		return Object(values), nil
	case map[string]any:
		values := make(map[string]Value, len(v))
		for key, item := range v {
			converted, err := ToValue(item)
			if err != nil {
				return nil, err
			}
			values[key] = converted
		}
		return Object(values), nil
	default:
		return nil, fmt.Errorf("unsupported Go value type %T", value)
	}
}

// FromValue converts stable GoScript values into ordinary Go values.
func FromValue(value Value) any {
	switch v := value.(type) {
	case *object.Null, *object.Undefined:
		return nil
	case *object.Boolean:
		return v.Value
	case *object.Number:
		return v.Value
	case *object.String:
		return v.Value
	case *object.Array:
		items := make([]any, len(v.Elements))
		for i, item := range v.Elements {
			items[i] = FromValue(item)
		}
		return items
	case *object.Hash:
		out := make(map[string]any, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			if key, ok := pair.Key.(*object.String); ok {
				out[key.Value] = FromValue(pair.Value)
			}
		}
		return out
	default:
		return value
	}
}

// AsString reads a string argument.
func AsString(value Value) (string, bool) {
	v, ok := value.(*object.String)
	if !ok {
		return "", false
	}
	return v.Value, true
}

// AsNumber reads a number argument.
func AsNumber(value Value) (float64, bool) {
	v, ok := value.(*object.Number)
	if !ok {
		return 0, false
	}
	return v.Value, true
}

// AsBool reads a boolean argument.
func AsBool(value Value) (bool, bool) {
	v, ok := value.(*object.Boolean)
	if !ok {
		return false, false
	}
	return v.Value, true
}

// AsObject reads an object argument.
func AsObject(value Value) (*object.Hash, bool) {
	v, ok := value.(*object.Hash)
	return v, ok
}

// AsArray reads an array argument.
func AsArray(value Value) ([]Value, bool) {
	v, ok := value.(*object.Array)
	if !ok {
		return nil, false
	}
	items := make([]Value, len(v.Elements))
	copy(items, v.Elements)
	return items, true
}
