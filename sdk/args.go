package sdk

import (
	"fmt"

	"github.com/issueye/goscript/internal/object"
)

// Args provides named argument readers for host methods.
type Args struct {
	Pos    Position
	Method string
	Values []Value
}

// NewArgs creates an argument reader.
func NewArgs(ctx CallContext, values []Value) Args {
	return Args{Pos: ctx.Pos, Method: ctx.Method, Values: values}
}

// Len returns the number of passed arguments.
func (a Args) Len() int { return len(a.Values) }

// Value returns the raw argument at index, or undefined if missing.
func (a Args) Value(index int) Value {
	if index < 0 || index >= len(a.Values) {
		return object.UNDEFINED
	}
	return a.Values[index]
}

// String reads a required string argument.
func (a Args) String(index int, name string) (string, error) {
	value := a.Value(index)
	if got, ok := AsString(value); ok {
		return got, nil
	}
	return "", a.typeError(name, "string", value)
}

// StringDefault reads an optional string argument.
func (a Args) StringDefault(index int, name, fallback string) (string, error) {
	value := a.Value(index)
	if value == object.UNDEFINED || value == object.NULL {
		return fallback, nil
	}
	if got, ok := AsString(value); ok {
		return got, nil
	}
	return "", a.typeError(name, "string", value)
}

// Number reads a required number argument.
func (a Args) Number(index int, name string) (float64, error) {
	value := a.Value(index)
	if got, ok := AsNumber(value); ok {
		return got, nil
	}
	return 0, a.typeError(name, "number", value)
}

// NumberDefault reads an optional number argument.
func (a Args) NumberDefault(index int, name string, fallback float64) (float64, error) {
	value := a.Value(index)
	if value == object.UNDEFINED || value == object.NULL {
		return fallback, nil
	}
	if got, ok := AsNumber(value); ok {
		return got, nil
	}
	return 0, a.typeError(name, "number", value)
}

// Bool reads a required boolean argument.
func (a Args) Bool(index int, name string) (bool, error) {
	value := a.Value(index)
	if got, ok := AsBool(value); ok {
		return got, nil
	}
	return false, a.typeError(name, "boolean", value)
}

// BoolDefault reads an optional boolean argument.
func (a Args) BoolDefault(index int, name string, fallback bool) (bool, error) {
	value := a.Value(index)
	if value == object.UNDEFINED || value == object.NULL {
		return fallback, nil
	}
	if got, ok := AsBool(value); ok {
		return got, nil
	}
	return false, a.typeError(name, "boolean", value)
}

// Object reads a required object argument.
func (a Args) Object(index int, name string) (*object.Hash, error) {
	value := a.Value(index)
	if got, ok := AsObject(value); ok {
		return got, nil
	}
	return nil, a.typeError(name, "object", value)
}

// Array reads a required array argument.
func (a Args) Array(index int, name string) ([]Value, error) {
	value := a.Value(index)
	if got, ok := AsArray(value); ok {
		return got, nil
	}
	return nil, a.typeError(name, "array", value)
}

func (a Args) typeError(name, want string, value Value) error {
	if name == "" {
		name = "argument"
	}
	got := "missing"
	if value != nil {
		got = string(value.Type())
	}
	prefix := name
	if a.Method != "" {
		prefix = a.Method + ": " + name
	}
	return Errorf("TypeError", "%s must be a %s, got %s", prefix, want, got)
}

// HashValue reads a string-keyed member from an object.
func HashValue(hash *object.Hash, key string) (Value, bool) {
	if hash == nil {
		return object.UNDEFINED, false
	}
	pair, ok := hash.Pairs[object.HashKeyFor(&object.String{Value: key})]
	if !ok {
		return object.UNDEFINED, false
	}
	return pair.Value, true
}

// HashString reads a string member from an object.
func HashString(hash *object.Hash, key string) (string, error) {
	value, ok := HashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return "", fmt.Errorf("%s is required", key)
	}
	got, ok := AsString(value)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return got, nil
}
