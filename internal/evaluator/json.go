package evaluator

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerJSON(env *object.Environment) {
	env.Set("JSON", &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hk("stringify"): {Key: &object.String{Value: "stringify"}, Value: &object.Builtin{Name: "JSON.stringify", Fn: builtinJSONStringify}},
			hk("parse"):    {Key: &object.String{Value: "parse"}, Value: &object.Builtin{Name: "JSON.parse", Fn: builtinJSONParse}},
		},
	})
}

func hk(s string) object.HashKey { return hashKey(&object.String{Value: s}) }

func builtinJSONStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	result := toJSON(args[0])
	if result == "" {
		return &object.Error{Message: "JSON.stringify: unsupported value", Pos: pos}
	}
	return &object.String{Value: result}
}

func toJSON(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Null:
		return "null"
	case *object.Undefined:
		return "null"
	case *object.Boolean:
		if v.Value {
			return "true"
		}
		return "false"
	case *object.Number:
		return v.Inspect()
	case *object.String:
		b, _ := json.Marshal(v.Value)
		return string(b)
	case *object.Array:
		var b strings.Builder
		b.WriteByte('[')
		for i, e := range v.Elements {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(toJSON(e))
		}
		b.WriteByte(']')
		return b.String()
	case *object.Hash:
		var b strings.Builder
		b.WriteByte('{')
		i := 0
		for _, pair := range v.Pairs {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(toJSON(pair.Key))
			b.WriteByte(':')
			b.WriteString(toJSON(pair.Value))
			i++
		}
		b.WriteByte('}')
		return b.String()
	case *object.Instance:
		var b strings.Builder
		b.WriteByte('{')
		i := 0
		for k, val := range v.Props {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(toJSON(&object.String{Value: k}))
			b.WriteByte(':')
			b.WriteString(toJSON(val))
			i++
		}
		b.WriteByte('}')
		return b.String()
	default:
		return ""
	}
}

func builtinJSONParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "JSON.parse requires a string argument")
	}
	input, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "JSON.parse requires a string argument")
	}
	result, err := parseJSONValue(input.Value)
	if err != nil {
		return object.NewError(pos, "JSON.parse: %v", err)
	}
	return result
}

func parseJSONValue(s string) (object.Object, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("unexpected end of JSON input")
	}
	switch s[0] {
	case '{':
		return parseJSONObject(s)
	case '[':
		return parseJSONArray(s)
	case '"':
		// Use Go's json decoder for strings
		var str string
		if err := json.Unmarshal([]byte(s), &str); err != nil {
			return nil, err
		}
		return &object.String{Value: str}, nil
	case 't':
		if strings.HasPrefix(s, "true") {
			return object.TRUE, nil
		}
	case 'f':
		if strings.HasPrefix(s, "false") {
			return object.FALSE, nil
		}
	case 'n':
		if strings.HasPrefix(s, "null") {
			return object.NULL, nil
		}
	default:
		if s[0] == '-' || (s[0] >= '0' && s[0] <= '9') {
			// Number
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, err
			}
			return &object.Number{Value: f}, nil
		}
	}
	// Fallback to Go's decoder
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, err
	}
	return goToObject(raw), nil
}

func parseJSONObject(s string) (object.Object, error) {
	// Use Go JSON decoder for simplicity
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return goToObject(m), nil
}

func parseJSONArray(s string) (object.Object, error) {
	var a []interface{}
	if err := json.Unmarshal([]byte(s), &a); err != nil {
		return nil, err
	}
	return goToObject(a), nil
}

func goToObject(v interface{}) object.Object {
	switch v := v.(type) {
	case nil:
		return object.NULL
	case bool:
		return object.NativeBool(v)
	case float64:
		return &object.Number{Value: v}
	case string:
		return &object.String{Value: v}
	case map[string]interface{}:
		hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for k, val := range v {
			key := &object.String{Value: k}
			hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: goToObject(val)}
		}
		return hash
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, e := range v {
			elements[i] = goToObject(e)
		}
		return &object.Array{Elements: elements}
	default:
		return object.NULL
	}
}
