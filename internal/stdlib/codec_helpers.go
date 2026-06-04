package stdlib

import (
	"fmt"
	"math"
	"sort"

	"github.com/issueye/goscript/internal/object"
)

func goValueToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.NULL
	case bool:
		return object.NativeBool(v)
	case int:
		return &object.Number{Value: float64(v)}
	case int8:
		return &object.Number{Value: float64(v)}
	case int16:
		return &object.Number{Value: float64(v)}
	case int32:
		return &object.Number{Value: float64(v)}
	case int64:
		return &object.Number{Value: float64(v)}
	case uint:
		return &object.Number{Value: float64(v)}
	case uint8:
		return &object.Number{Value: float64(v)}
	case uint16:
		return &object.Number{Value: float64(v)}
	case uint32:
		return &object.Number{Value: float64(v)}
	case uint64:
		return &object.Number{Value: float64(v)}
	case float32:
		return &object.Number{Value: float64(v)}
	case float64:
		return &object.Number{Value: v}
	case string:
		return &object.String{Value: v}
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, item := range v {
			elements[i] = goValueToObject(item)
		}
		return &object.Array{Elements: elements}
	case []string:
		return strSliceToArray(v)
	case map[string]interface{}:
		out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for _, key := range sortedStringKeys(v) {
			setHashMember(out, key, goValueToObject(v[key]))
		}
		return out
	case map[string]string:
		out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			setHashMember(out, key, &object.String{Value: v[key]})
		}
		return out
	case map[interface{}]interface{}:
		out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for key, item := range v {
			setHashMember(out, objectToMapKey(goValueToObject(key)), goValueToObject(item))
		}
		return out
	default:
		return &object.String{Value: fmt.Sprint(v)}
	}
}

func objectToGoValue(value object.Object) interface{} {
	switch v := value.(type) {
	case *object.Null:
		return nil
	case *object.Undefined:
		return nil
	case *object.Boolean:
		return v.Value
	case *object.Number:
		if math.Trunc(v.Value) == v.Value {
			return int64(v.Value)
		}
		return v.Value
	case *object.String:
		return v.Value
	case *object.Array:
		out := make([]interface{}, len(v.Elements))
		for i, item := range v.Elements {
			out[i] = objectToGoValue(item)
		}
		return out
	case *object.Hash:
		out := make(map[string]interface{}, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			out[objectToMapKey(pair.Key)] = objectToGoValue(pair.Value)
		}
		return out
	default:
		return v.Inspect()
	}
}

func objectToMapKey(value object.Object) string {
	if s, ok := value.(*object.String); ok {
		return s.Value
	}
	return value.Inspect()
}

func sortedStringKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
