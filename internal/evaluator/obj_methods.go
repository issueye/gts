package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerObject(env *object.Environment) {
	env.VM().SetGlobalConst("Object", objectConstructorObject())
}

func objectConstructorObject() object.Object {
	return orderedHash(
		hashEntry("create", &object.Builtin{Name: "Object.create", Fn: builtinObjCreate}),
		hashEntry("keys", &object.Builtin{Name: "Object.keys", Fn: builtinObjKeys}),
		hashEntry("values", &object.Builtin{Name: "Object.values", Fn: builtinObjValues}),
		hashEntry("entries", &object.Builtin{Name: "Object.entries", Fn: builtinObjEntries}),
		hashEntry("fromEntries", &object.Builtin{Name: "Object.fromEntries", Fn: builtinObjFromEntries}),
		hashEntry("assign", &object.Builtin{Name: "Object.assign", Fn: builtinObjAssign}),
		hashEntry("freeze", &object.Builtin{Name: "Object.freeze", Fn: builtinObjFreeze}),
		hashEntry("isFrozen", &object.Builtin{Name: "Object.isFrozen", Fn: builtinObjIsFrozen}),
		hashEntry("seal", &object.Builtin{Name: "Object.seal", Fn: builtinObjSeal}),
		hashEntry("isSealed", &object.Builtin{Name: "Object.isSealed", Fn: builtinObjIsSealed}),
		hashEntry("getPrototypeOf", &object.Builtin{Name: "Object.getPrototypeOf", Fn: builtinObjGetPrototypeOf}),
		hashEntry("setPrototypeOf", &object.Builtin{Name: "Object.setPrototypeOf", Fn: builtinObjSetPrototypeOf}),
		hashEntry("hasOwn", &object.Builtin{Name: "Object.hasOwn", Fn: builtinObjHasOwn}),
		hashEntry("is", &object.Builtin{Name: "Object.is", Fn: builtinObjIs}),
		hashEntry("defineProperty", &object.Builtin{Name: "Object.defineProperty", Fn: builtinObjDefineProperty}),
		hashEntry("getOwnPropertyDescriptor", &object.Builtin{Name: "Object.getOwnPropertyDescriptor", Fn: builtinObjGetOwnPropertyDescriptor}),
		hashEntry("getOwnPropertyNames", &object.Builtin{Name: "Object.getOwnPropertyNames", Fn: builtinObjKeys}),
	)
}

func builtinObjCreate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	if len(args) > 0 {
		switch proto := args[0].(type) {
		case *object.Hash:
			obj.Proto = proto
		case *object.Null, *object.Undefined:
		default:
			return object.NewError(pos, "TypeError: Object.create proto must be an object or null")
		}
	}
	if len(args) > 1 {
		if props, ok := args[1].(*object.Hash); ok {
			for _, pair := range props.OrderedPairs() {
				name := pair.Key.Inspect()
				if desc, ok := pair.Value.(*object.Hash); ok {
					if value := getHashKey(desc, &object.String{Value: "value"}); value != object.UNDEFINED {
						obj.SetMember(&object.String{Value: name}, value)
					}
				}
			}
		}
	}
	return obj
}

func builtinObjKeys(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.keys requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			elems = append(elems, pair.Key)
		}
		return &object.Array{Elements: elems}
	case *object.Instance:
		elems := make([]object.Object, 0, len(v.Props))
		for k := range v.Props {
			elems = append(elems, &object.String{Value: k})
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjValues(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.values requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			elems = append(elems, pair.Value)
		}
		return &object.Array{Elements: elems}
	case *object.Instance:
		elems := make([]object.Object, 0, len(v.Props))
		for _, val := range v.Props {
			elems = append(elems, val)
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjEntries(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.entries requires an argument")
	}
	switch v := args[0].(type) {
	case *object.Hash:
		elems := make([]object.Object, 0, len(v.Pairs))
		for _, pair := range v.OrderedPairs() {
			entry := &object.Array{Elements: []object.Object{pair.Key, pair.Value}}
			elems = append(elems, entry)
		}
		return &object.Array{Elements: elems}
	default:
		return &object.Array{Elements: nil}
	}
}

func builtinObjFromEntries(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.fromEntries requires an array")
	}
	entries, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "TypeError: Object.fromEntries requires an array")
	}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for _, entryObj := range entries.Elements {
		entry, ok := entryObj.(*object.Array)
		if !ok || len(entry.Elements) < 2 {
			return object.NewError(pos, "TypeError: Object.fromEntries entries must be [key, value] arrays")
		}
		key := &object.String{Value: entry.Elements[0].Inspect()}
		obj.SetMember(key, entry.Elements[1])
	}
	return obj
}

func builtinObjAssign(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "Object.assign requires at least 1 argument")
	}
	target, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "Object.assign target must be an object")
	}
	if target.Frozen {
		return object.NewError(pos, "TypeError: cannot assign to frozen object")
	}
	for _, src := range args[1:] {
		srcHash, ok := src.(*object.Hash)
		if !ok {
			continue
		}
		for _, pair := range srcHash.OrderedPairs() {
			k := hashKey(pair.Key)
			if target.Sealed {
				if _, ok := target.Pairs[k]; !ok {
					return object.NewError(pos, "TypeError: cannot add property to sealed object")
				}
			}
			target.Set(k, pair)
		}
	}
	return target
}

func builtinObjFreeze(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	if obj, ok := args[0].(*object.Hash); ok {
		obj.Frozen = true
		obj.Sealed = true
		return obj
	}
	return args[0]
}

func builtinObjIsFrozen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	obj, ok := args[0].(*object.Hash)
	return object.NativeBool(ok && obj.Frozen)
}

func builtinObjSeal(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	if obj, ok := args[0].(*object.Hash); ok {
		obj.Sealed = true
		return obj
	}
	return args[0]
}

func builtinObjIsSealed(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	obj, ok := args[0].(*object.Hash)
	return object.NativeBool(ok && obj.Sealed)
}

func builtinObjGetPrototypeOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.UNDEFINED
	}
	if obj, ok := args[0].(*object.Hash); ok && obj.Proto != nil {
		return obj.Proto
	}
	return object.NULL
}

func builtinObjSetPrototypeOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "Object.setPrototypeOf requires object and prototype")
	}
	obj, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "TypeError: Object.setPrototypeOf target must be an object")
	}
	switch proto := args[1].(type) {
	case *object.Hash:
		obj.Proto = proto
	case *object.Null:
		obj.Proto = nil
	default:
		return object.NewError(pos, "TypeError: Object.setPrototypeOf prototype must be an object or null")
	}
	return obj
}

func builtinObjHasOwn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.FALSE
	}
	name, ok := args[1].(*object.String)
	if !ok {
		return object.FALSE
	}
	switch v := args[0].(type) {
	case *object.Hash:
		key := hashKey(name)
		_, ok := v.Pairs[key]
		return object.NativeBool(ok)
	case *object.Instance:
		_, ok := v.Props[name.Value]
		return object.NativeBool(ok)
	}
	return object.FALSE
}

func builtinObjIs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.FALSE
	}
	if a, ok := args[0].(*object.Number); ok {
		if b, ok := args[1].(*object.Number); ok {
			if a.Value == 0 && b.Value == 0 {
				return object.NativeBool(!mathSignBitMismatch(a.Value, b.Value))
			}
			if a.Value != a.Value && b.Value != b.Value {
				return object.TRUE
			}
		}
	}
	return object.NativeBool(strictEqual(args[0], args[1]))
}

func builtinObjDefineProperty(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 3 {
		return object.NewError(pos, "Object.defineProperty requires object, key and descriptor")
	}
	obj, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "TypeError: Object.defineProperty target must be an object")
	}
	if obj.Frozen {
		return object.NewError(pos, "TypeError: cannot define property on frozen object")
	}
	key := &object.String{Value: args[1].Inspect()}
	if obj.Sealed {
		if _, ok := obj.Pairs[hashKey(key)]; !ok {
			return object.NewError(pos, "TypeError: cannot add property to sealed object")
		}
	}
	desc, ok := args[2].(*object.Hash)
	if !ok {
		return object.NewError(pos, "TypeError: Object.defineProperty descriptor must be an object")
	}
	value := getHashKey(desc, &object.String{Value: "value"})
	if value == object.UNDEFINED {
		value = object.UNDEFINED
	}
	obj.SetMember(key, value)
	return obj
}

func builtinObjGetOwnPropertyDescriptor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.UNDEFINED
	}
	obj, ok := args[0].(*object.Hash)
	if !ok {
		return object.UNDEFINED
	}
	key := &object.String{Value: args[1].Inspect()}
	pair, ok := obj.Pairs[hashKey(key)]
	if !ok {
		return object.UNDEFINED
	}
	desc := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	desc.SetMember(&object.String{Value: "value"}, pair.Value)
	desc.SetMember(&object.String{Value: "writable"}, object.NativeBool(!obj.Frozen))
	desc.SetMember(&object.String{Value: "enumerable"}, object.TRUE)
	desc.SetMember(&object.String{Value: "configurable"}, object.NativeBool(!obj.Sealed))
	return desc
}

func mathSignBitMismatch(a, b float64) bool {
	return (1/a < 0) != (1/b < 0)
}
