package stdlib

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/timers", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTimersModule(exports)
		return exports, nil
	})
}

func initTimersModule(exports *object.Hash) {
	setHashMember(exports, "setTimeout", timerAlias("setTimeout"))
	setHashMember(exports, "clearTimeout", timerAlias("clearTimeout"))
	setHashMember(exports, "setInterval", timerAlias("setInterval"))
	setHashMember(exports, "clearInterval", timerAlias("clearInterval"))
	setHashMember(exports, "queueMicrotask", timerAlias("queueMicrotask"))
	setHashMember(exports, "sleep", timerAlias("sleep"))
}

func timerAlias(name string) *object.Builtin {
	return &object.Builtin{Name: "timers." + name, Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		obj, ok := env.Get(name)
		if !ok {
			return object.NewError(pos, "timers.%s: global builtin %s not found", name, name)
		}
		builtin, ok := obj.(*object.Builtin)
		if !ok {
			return object.NewError(pos, "timers.%s: global %s is not a builtin", name, name)
		}
		return builtin.Fn(env, pos, args...)
	}}
}
