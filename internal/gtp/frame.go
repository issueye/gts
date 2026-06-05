package gtp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/issueye/goscript/internal/object"
)

const Version = 1

type Frame struct {
	Version      int                        `json:"v"`
	ID           string                     `json:"id"`
	Type         string                     `json:"type"`
	Runtime      string                     `json:"runtime,omitempty"`
	Protocol     string                     `json:"protocol,omitempty"`
	Capabilities []string                   `json:"capabilities,omitempty"`
	Modules      any                        `json:"modules,omitempty"`
	Service      string                     `json:"service,omitempty"`
	Module       string                     `json:"module,omitempty"`
	Method       string                     `json:"method,omitempty"`
	Resource     string                     `json:"resource,omitempty"`
	Event        string                     `json:"event,omitempty"`
	Args         []Value                    `json:"args,omitempty"`
	DeadlineMS   int64                      `json:"deadlineMs,omitempty"`
	Target       string                     `json:"target,omitempty"`
	Reason       string                     `json:"reason,omitempty"`
	OK           *bool                      `json:"ok,omitempty"`
	Result       *Value                     `json:"result,omitempty"`
	Error        *Error                     `json:"error,omitempty"`
	Data         *Value                     `json:"data,omitempty"`
	Extra        map[string]json.RawMessage `json:"-"`
}

type Error struct {
	Name    string         `json:"name"`
	Message string         `json:"message"`
	Code    string         `json:"code,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type Value struct {
	Type     string           `json:"$t"`
	Value    any              `json:"-"`
	Encoding string           `json:"encoding,omitempty"`
	Special  string           `json:"special,omitempty"`
	ID       string           `json:"id,omitempty"`
	Kind     string           `json:"kind,omitempty"`
	Methods  []string         `json:"methods,omitempty"`
	Name     string           `json:"name,omitempty"`
	Message  string           `json:"message,omitempty"`
	Fields   map[string]Value `json:"-"`
	Items    []Value          `json:"-"`
}

func Undefined() Value { return Value{Type: "undefined"} }
func Null() Value      { return Value{Type: "null"} }
func Bool(v bool) Value {
	return Value{Type: "boolean", Value: v}
}
func Number(v float64) Value {
	if math.IsNaN(v) {
		return Value{Type: "number", Special: "NaN"}
	}
	if math.IsInf(v, 1) {
		return Value{Type: "number", Special: "Infinity"}
	}
	if math.IsInf(v, -1) {
		return Value{Type: "number", Special: "-Infinity"}
	}
	return Value{Type: "number", Value: v}
}
func String(v string) Value { return Value{Type: "string", Value: v} }
func Bytes(v []byte) Value {
	return Value{Type: "bytes", Encoding: "base64", Value: base64.StdEncoding.EncodeToString(v)}
}
func Array(items []Value) Value {
	return Value{Type: "array", Items: append([]Value{}, items...)}
}
func Object(fields map[string]Value) Value {
	out := make(map[string]Value, len(fields))
	for key, value := range fields {
		out[key] = value
	}
	return Value{Type: "object", Fields: out}
}
func Resource(id, kind string, methods []string) Value {
	return Value{Type: "resource", ID: id, Kind: kind, Methods: append([]string{}, methods...)}
}

func (v Value) MarshalJSON() ([]byte, error) {
	return appendValueJSON(nil, v), nil
}

func appendValueJSON(out []byte, v Value) []byte {
	out = append(out, `{"$t":`...)
	out = strconv.AppendQuote(out, v.Type)
	switch v.Type {
	case "array":
		out = append(out, `,"v":[`...)
		for i, item := range v.Items {
			if i > 0 {
				out = append(out, ',')
			}
			out = appendValueJSON(out, item)
		}
		out = append(out, ']')
	case "object":
		out = append(out, `,"v":{`...)
		i := 0
		for key, value := range v.Fields {
			if i > 0 {
				out = append(out, ',')
			}
			out = strconv.AppendQuote(out, key)
			out = append(out, ':')
			out = appendValueJSON(out, value)
			i++
		}
		out = append(out, '}')
	case "resource":
		if v.ID != "" {
			out = append(out, `,"id":`...)
			out = strconv.AppendQuote(out, v.ID)
		}
		if v.Kind != "" {
			out = append(out, `,"kind":`...)
			out = strconv.AppendQuote(out, v.Kind)
		}
		if len(v.Methods) > 0 {
			out = append(out, `,"methods":[`...)
			for i, method := range v.Methods {
				if i > 0 {
					out = append(out, ',')
				}
				out = strconv.AppendQuote(out, method)
			}
			out = append(out, ']')
		}
	case "error":
		if v.Name != "" {
			out = append(out, `,"name":`...)
			out = strconv.AppendQuote(out, v.Name)
		}
		if v.Message != "" {
			out = append(out, `,"message":`...)
			out = strconv.AppendQuote(out, v.Message)
		}
	case "number":
		if v.Special != "" {
			out = append(out, `,"special":`...)
			out = strconv.AppendQuote(out, v.Special)
		} else {
			out = append(out, `,"v":`...)
			out = appendNumberJSON(out, v.Value)
		}
	case "bytes":
		if v.Encoding != "" {
			out = append(out, `,"encoding":`...)
			out = strconv.AppendQuote(out, v.Encoding)
		}
		out = append(out, `,"v":`...)
		out = strconv.AppendQuote(out, fmt.Sprint(v.Value))
	case "string":
		out = append(out, `,"v":`...)
		out = strconv.AppendQuote(out, fmt.Sprint(v.Value))
	case "boolean":
		out = append(out, `,"v":`...)
		if b, ok := v.Value.(bool); ok && b {
			out = append(out, "true"...)
		} else {
			out = append(out, "false"...)
		}
	case "undefined", "null":
	default:
		if v.Value != nil {
			out = append(out, `,"v":`...)
			data, err := json.Marshal(v.Value)
			if err == nil {
				out = append(out, data...)
			} else {
				out = append(out, "null"...)
			}
		}
	}
	out = append(out, '}')
	return out
}

func appendNumberJSON(out []byte, value any) []byte {
	switch n := value.(type) {
	case float64:
		return strconv.AppendFloat(out, n, 'g', -1, 64)
	case float32:
		return strconv.AppendFloat(out, float64(n), 'g', -1, 32)
	case int:
		return strconv.AppendInt(out, int64(n), 10)
	case int64:
		return strconv.AppendInt(out, n, 10)
	case uint64:
		return strconv.AppendUint(out, n, 10)
	default:
		return strconv.AppendFloat(out, 0, 'g', -1, 64)
	}
}

func (v *Value) UnmarshalJSON(data []byte) error {
	var raw valueWireRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*v = Value{
		Type:     raw.Type,
		Encoding: raw.Encoding,
		Special:  raw.Special,
		ID:       raw.ID,
		Kind:     raw.Kind,
		Methods:  raw.Methods,
		Name:     raw.Name,
		Message:  raw.Message,
	}
	switch v.Type {
	case "array":
		if len(raw.Value) > 0 {
			return json.Unmarshal(raw.Value, &v.Items)
		}
	case "object":
		if len(raw.Value) > 0 {
			return json.Unmarshal(raw.Value, &v.Fields)
		}
	case "number":
		if v.Special != "" {
			return nil
		}
		if len(raw.Value) > 0 {
			var n float64
			if err := json.Unmarshal(raw.Value, &n); err != nil {
				return err
			}
			v.Value = n
		}
	case "string", "bytes":
		if len(raw.Value) > 0 {
			var s string
			if err := json.Unmarshal(raw.Value, &s); err != nil {
				return err
			}
			v.Value = s
		}
	case "boolean":
		if len(raw.Value) > 0 {
			var b bool
			if err := json.Unmarshal(raw.Value, &b); err != nil {
				return err
			}
			v.Value = b
		}
	}
	return nil
}

type valueWireRaw struct {
	Type     string          `json:"$t"`
	Value    json.RawMessage `json:"v"`
	Encoding string          `json:"encoding,omitempty"`
	Special  string          `json:"special,omitempty"`
	ID       string          `json:"id,omitempty"`
	Kind     string          `json:"kind,omitempty"`
	Methods  []string        `json:"methods,omitempty"`
	Name     string          `json:"name,omitempty"`
	Message  string          `json:"message,omitempty"`
}

func FromObject(obj object.Object) Value {
	switch v := obj.(type) {
	case *object.Undefined:
		return Undefined()
	case *object.Null:
		return Null()
	case *object.Boolean:
		return Bool(v.Value)
	case *object.Number:
		return Number(v.Value)
	case *object.String:
		return String(v.Value)
	case *object.Array:
		items := make([]Value, len(v.Elements))
		for i, item := range v.Elements {
			items[i] = FromObject(item)
		}
		return Array(items)
	case *object.Hash:
		fields := make(map[string]Value, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			if key, ok := pair.Key.(*object.String); ok {
				fields[key.Value] = FromObject(pair.Value)
			}
		}
		return Object(fields)
	case *object.Error:
		return Value{Type: "error", Name: v.Name, Message: v.Message}
	case *object.GoObject:
		return Resource(fmt.Sprintf("%p", v.Value), fmt.Sprintf("%T", v.Value), nil)
	default:
		return Null()
	}
}

func EncodeFrame(frame Frame) ([]byte, error) {
	if frame.Version == 0 {
		frame.Version = Version
	}
	return json.Marshal(frame)
}

func DecodeFrame(data []byte) (Frame, error) {
	var frame Frame
	err := json.Unmarshal(data, &frame)
	return frame, err
}
