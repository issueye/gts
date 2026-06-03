package evaluator

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerJSON(env *object.Environment) {
	env.VM().SetGlobalConst("JSON", jsonObject())
}

func jsonObject() object.Object {
	return &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hk("stringify"): {Key: &object.String{Value: "stringify"}, Value: &object.Builtin{Name: "JSON.stringify", Fn: builtinJSONStringify}},
			hk("parse"):     {Key: &object.String{Value: "parse"}, Value: &object.Builtin{Name: "JSON.parse", Fn: builtinJSONParse}},
		},
	}
}

func hk(s string) object.HashKey { return hashKey(&object.String{Value: s}) }

func builtinJSONStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	value := args[0]
	if len(args) > 1 && args[1] != object.NULL && args[1] != object.UNDEFINED {
		value = applyJSONReplacerTree(env, pos, args[1], &object.String{Value: ""}, value)
		if object.IsRuntimeError(value) {
			return value
		}
	}
	raw := toGoJSONValue(value)
	space := jsonIndent(args)
	var data []byte
	var err error
	if space != "" {
		data, err = json.MarshalIndent(raw, "", space)
	} else {
		data, err = json.Marshal(raw)
	}
	if err != nil {
		return object.NewError(pos, "JSON.stringify: %v", err)
	}
	return &object.String{Value: string(data)}
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

func toGoJSONValue(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.Null, *object.Undefined:
		return nil
	case *object.Boolean:
		return v.Value
	case *object.Number:
		if math.IsNaN(v.Value) || math.IsInf(v.Value, 0) {
			return nil
		}
		return v.Value
	case *object.String:
		return v.Value
	case *object.Array:
		items := make([]interface{}, len(v.Elements))
		for i, item := range v.Elements {
			items[i] = toGoJSONValue(item)
		}
		return items
	case *object.Hash:
		out := make(map[string]interface{}, len(v.Pairs))
		for _, pair := range v.Pairs {
			out[pair.Key.Inspect()] = toGoJSONValue(pair.Value)
		}
		return out
	case *object.Instance:
		out := make(map[string]interface{}, len(v.Props))
		for key, value := range v.Props {
			out[key] = toGoJSONValue(value)
		}
		return out
	default:
		return nil
	}
}

func jsonIndent(args []object.Object) string {
	if len(args) < 3 || args[2] == object.NULL || args[2] == object.UNDEFINED {
		return ""
	}
	switch s := args[2].(type) {
	case *object.Number:
		n := int(s.Value)
		if n < 0 {
			n = 0
		}
		if n > 10 {
			n = 10
		}
		return strings.Repeat(" ", n)
	case *object.String:
		if len(s.Value) > 10 {
			return s.Value[:10]
		}
		return s.Value
	default:
		return ""
	}
}

func applyJSONReplacer(env *object.Environment, pos ast.Position, replacer object.Object, key object.Object, value object.Object) object.Object {
	switch replacer.(type) {
	case *object.Function, *object.Builtin:
		return applyFunction(replacer, env, []object.Object{key, value}, pos)
	default:
		return value
	}
}

func applyJSONReplacerTree(env *object.Environment, pos ast.Position, replacer object.Object, key object.Object, value object.Object) object.Object {
	switch v := value.(type) {
	case *object.Array:
		elements := make([]object.Object, len(v.Elements))
		for i, item := range v.Elements {
			next := applyJSONReplacerTree(env, pos, replacer, &object.String{Value: strconv.Itoa(i)}, item)
			if object.IsRuntimeError(next) {
				return next
			}
			elements[i] = next
		}
		value = &object.Array{Elements: elements, Pos: v.Pos}
	case *object.Hash:
		pairs := make(map[object.HashKey]object.HashPair, len(v.Pairs))
		for hk, pair := range v.Pairs {
			next := applyJSONReplacerTree(env, pos, replacer, pair.Key, pair.Value)
			if object.IsRuntimeError(next) {
				return next
			}
			pairs[hk] = object.HashPair{Key: pair.Key, Value: next}
		}
		value = &object.Hash{Pairs: pairs, Proto: v.Proto, Frozen: v.Frozen, Sealed: v.Sealed, Pos: v.Pos}
	case *object.Instance:
		props := make(map[string]object.Object, len(v.Props))
		for name, item := range v.Props {
			next := applyJSONReplacerTree(env, pos, replacer, &object.String{Value: name}, item)
			if object.IsRuntimeError(next) {
				return next
			}
			props[name] = next
		}
		value = &object.Instance{Class: v.Class, Props: props, Pos: v.Pos}
	}
	return applyJSONReplacer(env, pos, replacer, key, value)
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
	if len(args) > 1 && args[1] != object.NULL && args[1] != object.UNDEFINED {
		result = applyJSONReviver(env, pos, args[1], &object.String{Value: ""}, result)
		if object.IsRuntimeError(result) {
			return result
		}
	}
	return result
}

func applyJSONReviver(env *object.Environment, pos ast.Position, reviver object.Object, key object.Object, value object.Object) object.Object {
	switch v := value.(type) {
	case *object.Array:
		for i, item := range v.Elements {
			next := applyJSONReviver(env, pos, reviver, &object.String{Value: strconv.Itoa(i)}, item)
			if object.IsRuntimeError(next) {
				return next
			}
			v.Elements[i] = next
		}
	case *object.Hash:
		for hk, pair := range v.Pairs {
			next := applyJSONReviver(env, pos, reviver, pair.Key, pair.Value)
			if object.IsRuntimeError(next) {
				return next
			}
			v.Pairs[hk] = object.HashPair{Key: pair.Key, Value: next}
		}
	}
	switch reviver.(type) {
	case *object.Function, *object.Builtin:
		return applyFunction(reviver, env, []object.Object{key, value}, pos)
	default:
		return value
	}
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
