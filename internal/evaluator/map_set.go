package evaluator

import (
	"fmt"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var mapMethods map[string]object.BuiltinFunc
var setMethods map[string]object.BuiltinFunc

func init() {
	mapMethods = map[string]object.BuiltinFunc{
		"set":    builtinMapSet,
		"get":    builtinMapGet,
		"has":    builtinMapHas,
		"delete": builtinMapDelete,
		"clear":  builtinMapClear,
	}
	setMethods = map[string]object.BuiltinFunc{
		"add":    builtinSetAdd,
		"has":    builtinSetHas,
		"delete": builtinSetDelete,
		"clear":  builtinSetClear,
	}
}

func registerMapSet(env *object.Environment) {
	env.VM().SetGlobalConst("Map", callableBuiltinObject("Map", builtinMapConstructor, nil))
	env.VM().SetGlobalConst("Set", callableBuiltinObject("Set", builtinSetConstructor, nil))
}

func builtinMapConstructor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.ObjectManager().NewMapAt(pos)
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return m
	}
	iterable, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "TypeError: Map constructor requires an array of [key, value] entries")
	}
	for _, entryObj := range iterable.Elements {
		entry, ok := entryObj.(*object.Array)
		if !ok || len(entry.Elements) < 2 {
			return object.NewError(pos, "TypeError: Map entries must be [key, value] arrays")
		}
		key := mapSetKey(entry.Elements[0])
		m.Entries[key] = object.HashPair{Key: entry.Elements[0], Value: entry.Elements[1]}
	}
	return m
}

func builtinMapSet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.Extra.(*object.Map)
	if len(args) < 2 {
		return object.NewError(pos, "TypeError: Map.set requires key and value")
	}
	key := mapSetKey(args[0])
	m.Entries[key] = object.HashPair{Key: args[0], Value: args[1]}
	return m
}

func builtinMapGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.Extra.(*object.Map)
	if len(args) < 1 {
		return object.UNDEFINED
	}
	if pair, ok := m.Entries[mapSetKey(args[0])]; ok {
		return pair.Value
	}
	return object.UNDEFINED
}

func builtinMapHas(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.Extra.(*object.Map)
	if len(args) < 1 {
		return object.FALSE
	}
	_, ok := m.Entries[mapSetKey(args[0])]
	return object.NativeBool(ok)
}

func builtinMapDelete(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.Extra.(*object.Map)
	if len(args) < 1 {
		return object.FALSE
	}
	key := mapSetKey(args[0])
	_, ok := m.Entries[key]
	if ok {
		delete(m.Entries, key)
	}
	return object.NativeBool(ok)
}

func builtinMapClear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := env.Extra.(*object.Map)
	m.Entries = make(map[object.HashKey]object.HashPair)
	return object.UNDEFINED
}

func builtinSetConstructor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := env.ObjectManager().NewSetAt(pos)
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return s
	}
	iterable, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "TypeError: Set constructor requires an array")
	}
	for _, value := range iterable.Elements {
		s.Values[mapSetKey(value)] = value
	}
	return s
}

func builtinSetAdd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := env.Extra.(*object.Set)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: Set.add requires a value")
	}
	s.Values[mapSetKey(args[0])] = args[0]
	return s
}

func builtinSetHas(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := env.Extra.(*object.Set)
	if len(args) < 1 {
		return object.FALSE
	}
	_, ok := s.Values[mapSetKey(args[0])]
	return object.NativeBool(ok)
}

func builtinSetDelete(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := env.Extra.(*object.Set)
	if len(args) < 1 {
		return object.FALSE
	}
	key := mapSetKey(args[0])
	_, ok := s.Values[key]
	if ok {
		delete(s.Values, key)
	}
	return object.NativeBool(ok)
}

func builtinSetClear(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := env.Extra.(*object.Set)
	s.Values = make(map[object.HashKey]object.Object)
	return object.UNDEFINED
}

func mapSetKey(o object.Object) object.HashKey {
	switch o := o.(type) {
	case *object.String:
		return object.HashKey{Type: o.Type(), Value: o.Value}
	case *object.Number:
		return object.HashKey{Type: o.Type(), Value: fmt.Sprintf("%v", o.Value)}
	case *object.Boolean:
		if o.Value {
			return object.HashKey{Type: o.Type(), Value: "true"}
		}
		return object.HashKey{Type: o.Type(), Value: "false"}
	case *object.Null:
		return object.HashKey{Type: o.Type(), Value: "null"}
	case *object.Undefined:
		return object.HashKey{Type: o.Type(), Value: "undefined"}
	default:
		return object.HashKey{Type: o.Type(), Value: fmt.Sprintf("%p", o)}
	}
}
