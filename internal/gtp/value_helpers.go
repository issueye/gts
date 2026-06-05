package gtp

import "fmt"

func (v Value) StringValue() (string, bool) {
	s, ok := v.Value.(string)
	return s, ok
}

func (v Value) NumberValue() (float64, bool) {
	n, ok := v.Value.(float64)
	return n, ok
}

func (v Value) BoolValue() (bool, bool) {
	b, ok := v.Value.(bool)
	return b, ok
}

func Field(obj Value, key string) (Value, bool) {
	if obj.Type != "object" || obj.Fields == nil {
		return Undefined(), false
	}
	value, ok := obj.Fields[key]
	return value, ok
}

func StringField(obj Value, key string) (string, bool) {
	value, ok := Field(obj, key)
	if !ok {
		return "", false
	}
	return value.StringValue()
}

func Plain(value Value) any {
	switch value.Type {
	case "undefined", "null":
		return nil
	case "boolean", "number", "string", "bytes":
		return value.Value
	case "array":
		items := make([]any, len(value.Items))
		for i, item := range value.Items {
			items[i] = Plain(item)
		}
		return items
	case "object":
		fields := make(map[string]any, len(value.Fields))
		for key, item := range value.Fields {
			fields[key] = Plain(item)
		}
		return fields
	case "resource":
		return map[string]any{"id": value.ID, "kind": value.Kind, "methods": value.Methods}
	case "error":
		return map[string]any{"name": value.Name, "message": value.Message}
	default:
		return value.Value
	}
}

func NumberField(obj Value, key string) (float64, bool) {
	value, ok := Field(obj, key)
	if !ok {
		return 0, false
	}
	return value.NumberValue()
}

func RequiredObjectArg(args []Value, index int, name string) (Value, *Error) {
	if index < 0 || index >= len(args) {
		return Value{}, TypeError("%s is required", name)
	}
	if args[index].Type != "object" {
		return Value{}, TypeError("%s must be an object", name)
	}
	return args[index], nil
}

func TypeError(format string, args ...any) *Error {
	return &Error{Name: "TypeError", Message: fmt.Sprintf(format, args...), Code: "TYPE_ERROR"}
}

func HostError(format string, args ...any) *Error {
	return &Error{Name: "HostError", Message: fmt.Sprintf(format, args...), Code: "HOST_ERROR"}
}

func NotFoundError(format string, args ...any) *Error {
	return &Error{Name: "ReferenceError", Message: fmt.Sprintf(format, args...), Code: "NOT_FOUND"}
}

func OKResult(id string, result Value) Frame {
	ok := true
	return Frame{Version: Version, ID: id, Type: "result", OK: &ok, Result: &result}
}

func ErrorResult(id string, err *Error) Frame {
	ok := false
	return Frame{Version: Version, ID: id, Type: "result", OK: &ok, Error: err}
}
