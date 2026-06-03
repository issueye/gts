package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func registerObject(env *object.Environment) {
	env.VM().SetGlobalConst("Object", objectConstructorObject())
}

func objectConstructorObject() object.Object {
	return &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hk("create"):                   {Key: &object.String{Value: "create"}, Value: &object.Builtin{Name: "Object.create", Fn: builtinObjCreate}},
			hk("keys"):                     {Key: &object.String{Value: "keys"}, Value: &object.Builtin{Name: "Object.keys", Fn: builtinObjKeys}},
			hk("values"):                   {Key: &object.String{Value: "values"}, Value: &object.Builtin{Name: "Object.values", Fn: builtinObjValues}},
			hk("entries"):                  {Key: &object.String{Value: "entries"}, Value: &object.Builtin{Name: "Object.entries", Fn: builtinObjEntries}},
			hk("fromEntries"):              {Key: &object.String{Value: "fromEntries"}, Value: &object.Builtin{Name: "Object.fromEntries", Fn: builtinObjFromEntries}},
			hk("assign"):                   {Key: &object.String{Value: "assign"}, Value: &object.Builtin{Name: "Object.assign", Fn: builtinObjAssign}},
			hk("freeze"):                   {Key: &object.String{Value: "freeze"}, Value: &object.Builtin{Name: "Object.freeze", Fn: builtinObjFreeze}},
			hk("isFrozen"):                 {Key: &object.String{Value: "isFrozen"}, Value: &object.Builtin{Name: "Object.isFrozen", Fn: builtinObjIsFrozen}},
			hk("seal"):                     {Key: &object.String{Value: "seal"}, Value: &object.Builtin{Name: "Object.seal", Fn: builtinObjSeal}},
			hk("isSealed"):                 {Key: &object.String{Value: "isSealed"}, Value: &object.Builtin{Name: "Object.isSealed", Fn: builtinObjIsSealed}},
			hk("getPrototypeOf"):           {Key: &object.String{Value: "getPrototypeOf"}, Value: &object.Builtin{Name: "Object.getPrototypeOf", Fn: builtinObjGetPrototypeOf}},
			hk("setPrototypeOf"):           {Key: &object.String{Value: "setPrototypeOf"}, Value: &object.Builtin{Name: "Object.setPrototypeOf", Fn: builtinObjSetPrototypeOf}},
			hk("hasOwn"):                   {Key: &object.String{Value: "hasOwn"}, Value: &object.Builtin{Name: "Object.hasOwn", Fn: builtinObjHasOwn}},
			hk("is"):                       {Key: &object.String{Value: "is"}, Value: &object.Builtin{Name: "Object.is", Fn: builtinObjIs}},
			hk("defineProperty"):           {Key: &object.String{Value: "defineProperty"}, Value: &object.Builtin{Name: "Object.defineProperty", Fn: builtinObjDefineProperty}},
			hk("getOwnPropertyDescriptor"): {Key: &object.String{Value: "getOwnPropertyDescriptor"}, Value: &object.Builtin{Name: "Object.getOwnPropertyDescriptor", Fn: builtinObjGetOwnPropertyDescriptor}},
			hk("getOwnPropertyNames"):      {Key: &object.String{Value: "getOwnPropertyNames"}, Value: &object.Builtin{Name: "Object.getOwnPropertyNames", Fn: builtinObjKeys}},
		},
	}
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
			for _, pair := range props.Pairs {
				name := pair.Key.Inspect()
				if desc, ok := pair.Value.(*object.Hash); ok {
					if value := getHashKey(desc, &object.String{Value: "value"}); value != object.UNDEFINED {
						obj.Pairs[hashKey(&object.String{Value: name})] = object.HashPair{Key: &object.String{Value: name}, Value: value}
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
		for _, pair := range v.Pairs {
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
		for _, pair := range v.Pairs {
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
		for _, pair := range v.Pairs {
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
		obj.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: entry.Elements[1]}
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
		for k, v := range srcHash.Pairs {
			if target.Sealed {
				if _, ok := target.Pairs[k]; !ok {
					return object.NewError(pos, "TypeError: cannot add property to sealed object")
				}
			}
			target.Pairs[k] = v
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
	obj.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: value}
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
	desc.Pairs[hashKey(&object.String{Value: "value"})] = object.HashPair{Key: &object.String{Value: "value"}, Value: pair.Value}
	desc.Pairs[hashKey(&object.String{Value: "writable"})] = object.HashPair{Key: &object.String{Value: "writable"}, Value: object.NativeBool(!obj.Frozen)}
	desc.Pairs[hashKey(&object.String{Value: "enumerable"})] = object.HashPair{Key: &object.String{Value: "enumerable"}, Value: object.TRUE}
	desc.Pairs[hashKey(&object.String{Value: "configurable"})] = object.HashPair{Key: &object.String{Value: "configurable"}, Value: object.NativeBool(!obj.Sealed)}
	return desc
}

func mathSignBitMismatch(a, b float64) bool {
	return (1/a < 0) != (1/b < 0)
}
