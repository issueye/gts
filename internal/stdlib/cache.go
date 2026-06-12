package stdlib

import (
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type cacheItem struct {
	value    object.Object
	expireAt time.Time
}

type cache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

func init() {
	module.RegisterNative("@std/cache", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "create", &object.Builtin{Name: "cache.create", Fn: cacheCreate})
		return exports, nil
	})
}

func cacheCreate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	c := &cache{items: make(map[string]*cacheItem)}
	inst := &object.Instance{Props: make(map[string]object.Object)}

	inst.Props["set"] = &object.Builtin{
		Name: "cache.set",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "set requires key and value")
			}
			key, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "key must be string")
			}

			ttl := time.Duration(0)
			if len(args) > 2 {
				if n, ok := args[2].(*object.Number); ok {
					ttl = time.Duration(n.Value) * time.Millisecond
				}
			}

			c.mu.Lock()
			item := &cacheItem{value: args[1]}
			if ttl > 0 {
				item.expireAt = time.Now().Add(ttl)
			}
			c.items[key.Value] = item
			c.mu.Unlock()
			return object.UNDEFINED
		},
	}

	inst.Props["get"] = &object.Builtin{
		Name: "cache.get",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "get requires key")
			}
			key, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "key must be string")
			}

			c.mu.RLock()
			item, exists := c.items[key.Value]
			c.mu.RUnlock()

			if !exists {
				return object.UNDEFINED
			}

			if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
				c.mu.Lock()
				delete(c.items, key.Value)
				c.mu.Unlock()
				return object.UNDEFINED
			}

			return item.value
		},
	}

	inst.Props["has"] = &object.Builtin{
		Name: "cache.has",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "has requires key")
			}
			key, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "key must be string")
			}

			c.mu.RLock()
			item, exists := c.items[key.Value]
			c.mu.RUnlock()

			if !exists {
				return object.FALSE
			}

			if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
				return object.FALSE
			}

			return object.TRUE
		},
	}

	inst.Props["delete"] = &object.Builtin{
		Name: "cache.delete",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "delete requires key")
			}
			key, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "key must be string")
			}

			c.mu.Lock()
			delete(c.items, key.Value)
			c.mu.Unlock()
			return object.UNDEFINED
		},
	}

	inst.Props["clear"] = &object.Builtin{
		Name: "cache.clear",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			c.mu.Lock()
			c.items = make(map[string]*cacheItem)
			c.mu.Unlock()
			return object.UNDEFINED
		},
	}

	inst.Props["size"] = &object.Builtin{
		Name: "cache.size",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			c.mu.RLock()
			size := len(c.items)
			c.mu.RUnlock()
			return &object.Number{Value: float64(size)}
		},
	}

	inst.Props["keys"] = &object.Builtin{
		Name: "cache.keys",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			c.mu.RLock()
			keys := make([]object.Object, 0, len(c.items))
			for k := range c.items {
				keys = append(keys, &object.String{Value: k})
			}
			c.mu.RUnlock()
			return &object.Array{Elements: keys}
		},
	}

	return inst
}
