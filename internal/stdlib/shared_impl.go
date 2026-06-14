package stdlib

import (
	"sync"
	"sync/atomic"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/shared", func(env *object.Environment) (object.Object, error) {
		ns := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(ns, "counter", &object.Builtin{
			Name: "shared.counter",
			Fn:   func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
				return sharedCounter(env, pos, args...)
			},
		})
		setHashMember(ns, "map", &object.Builtin{
			Name: "shared.map",
			Fn:   func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
				return sharedMap(env, pos, args...)
			},
		})
		setHashMember(ns, "atomic", &object.Builtin{
			Name: "shared.atomic",
			Fn:   func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
				return sharedAtomic(env, pos, args...)
			},
		})
		return ns, nil
	})
}

// sharedCounters holds process-wide named counters for isolated-mode cross-
// request state. Each counter is an atomic int64.
var sharedCounters sync.Map // map[string]*int64

func sharedCounter(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "shared.counter: name required")
	}
	nameObj, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "shared.counter: name must be a string")
	}
	name := nameObj.Value

	rawPtr, _ := sharedCounters.LoadOrStore(name, new(int64))
	ptr := rawPtr.(*int64)

	counter := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(counter, "incr", &object.Builtin{
		Name: "counter.incr",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			delta := int64(1)
			if len(args) >= 1 {
				if n, ok := args[0].(*object.Number); ok {
					delta = int64(n.Value)
				}
			}
			newVal := atomic.AddInt64(ptr, delta)
			return &object.Number{Value: float64(newVal)}
		},
	})
	setHashMember(counter, "decr", &object.Builtin{
		Name: "counter.decr",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			delta := int64(1)
			if len(args) >= 1 {
				if n, ok := args[0].(*object.Number); ok {
					delta = int64(n.Value)
				}
			}
			newVal := atomic.AddInt64(ptr, -delta)
			return &object.Number{Value: float64(newVal)}
		},
	})
	setHashMember(counter, "get", &object.Builtin{
		Name: "counter.get",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			val := atomic.LoadInt64(ptr)
			return &object.Number{Value: float64(val)}
		},
	})
	setHashMember(counter, "set", &object.Builtin{
		Name: "counter.set",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "counter.set: value required")
			}
			n, ok := args[0].(*object.Number)
			if !ok {
				return object.NewError(pos, "counter.set: value must be a number")
			}
			atomic.StoreInt64(ptr, int64(n.Value))
			return object.UNDEFINED
		},
	})
	return counter
}

// sharedMaps holds process-wide named maps for isolated-mode shared state.
var sharedMaps sync.Map // map[string]*sync.Map

func sharedMap(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "shared.map: name required")
	}
	nameObj, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "shared.map: name must be a string")
	}
	name := nameObj.Value

	rawMap, _ := sharedMaps.LoadOrStore(name, &sync.Map{})
	m := rawMap.(*sync.Map)

	mapObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(mapObj, "get", &object.Builtin{
		Name: "map.get",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "map.get: key required")
			}
			keyStr, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "map.get: key must be a string")
			}
			val, ok := m.Load(keyStr.Value)
			if !ok {
				return object.UNDEFINED
			}
			return val.(object.Object)
		},
	})
	setHashMember(mapObj, "set", &object.Builtin{
		Name: "map.set",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "map.set: key and value required")
			}
			keyStr, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "map.set: key must be a string")
			}
			m.Store(keyStr.Value, args[1])
			return object.UNDEFINED
		},
	})
	setHashMember(mapObj, "delete", &object.Builtin{
		Name: "map.delete",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "map.delete: key required")
			}
			keyStr, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "map.delete: key must be a string")
			}
			m.Delete(keyStr.Value)
			return object.UNDEFINED
		},
	})
	setHashMember(mapObj, "has", &object.Builtin{
		Name: "map.has",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "map.has: key required")
			}
			keyStr, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "map.has: key must be a string")
			}
			_, ok = m.Load(keyStr.Value)
			if ok {
				return object.TRUE
			}
			return object.FALSE
		},
	})
	setHashMember(mapObj, "keys", &object.Builtin{
		Name: "map.keys",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			var keys []object.Object
			m.Range(func(key, value any) bool {
				keys = append(keys, &object.String{Value: key.(string)})
				return true
			})
			return &object.Array{Elements: keys}
		},
	})
	return mapObj
}

// sharedAtomics holds process-wide named atomic values (object.Object stored
// via atomic.Value). Used for isolated-mode CAS patterns.
var sharedAtomics sync.Map // map[string]*atomic.Value

func sharedAtomic(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "shared.atomic: name required")
	}
	nameObj, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "shared.atomic: name must be a string")
	}
	name := nameObj.Value

	var initialValue object.Object = object.UNDEFINED
	if len(args) >= 2 {
		initialValue = args[1]
	}

	rawVal, _ := sharedAtomics.LoadOrStore(name, &atomic.Value{})
	atomicVal := rawVal.(*atomic.Value)
	// If this is the first access and an initial value was provided, store it.
	if atomicVal.Load() == nil && initialValue != object.UNDEFINED {
		atomicVal.CompareAndSwap(nil, initialValue)
	}

	atomicObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(atomicObj, "get", &object.Builtin{
		Name: "atomic.get",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			val := atomicVal.Load()
			if val == nil {
				return object.UNDEFINED
			}
			return val.(object.Object)
		},
	})
	setHashMember(atomicObj, "set", &object.Builtin{
		Name: "atomic.set",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "atomic.set: value required")
			}
			atomicVal.Store(args[0])
			return object.UNDEFINED
		},
	})
	setHashMember(atomicObj, "compareAndSwap", &object.Builtin{
		Name: "atomic.compareAndSwap",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "atomic.compareAndSwap: expect and update required")
			}
			expect := args[0]
			update := args[1]
			// CAS on object.Object: compare by pointer identity (not deep equality).
			swapped := atomicVal.CompareAndSwap(expect, update)
			if swapped {
				return object.TRUE
			}
			return object.FALSE
		},
	})
	return atomicObj
}
