package stdlib

import (
	"sync"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type prometheusMetrics struct {
	mu      sync.RWMutex
	metrics map[string]float64
}

func init() {
	module.RegisterNative("@std/prometheus", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "create", &object.Builtin{Name: "prometheus.create", Fn: prometheusCreate})
		return exports, nil
	})
}

func prometheusCreate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	m := &prometheusMetrics{metrics: make(map[string]float64)}
	inst := &object.Instance{Props: make(map[string]object.Object)}

	inst.Props["inc"] = &object.Builtin{
		Name: "prometheus.inc",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "inc requires name")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "inc expects string name")
			}
			m.mu.Lock()
			m.metrics[name.Value]++
			m.mu.Unlock()
			return object.UNDEFINED
		},
	}

	inst.Props["set"] = &object.Builtin{
		Name: "prometheus.set",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "set requires name and value")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "set expects string name")
			}
			val, ok := args[1].(*object.Number)
			if !ok {
				return object.NewError(pos, "set expects number value")
			}
			m.mu.Lock()
			m.metrics[name.Value] = val.Value
			m.mu.Unlock()
			return object.UNDEFINED
		},
	}

	inst.Props["get"] = &object.Builtin{
		Name: "prometheus.get",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "get requires name")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "get expects string name")
			}
			m.mu.RLock()
			val := m.metrics[name.Value]
			m.mu.RUnlock()
			return &object.Number{Value: val}
		},
	}

	return inst
}
