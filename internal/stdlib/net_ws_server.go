package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/ws/server", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initWSServerModule(exports)
		return exports, nil
	})
}

func initWSServerModule(exports *object.Hash) {
	setHashMember(exports, "createServer", &object.Builtin{Name: "ws.createServer", Fn: wsServerCreateServer})
	setHashMember(exports, "upgrade", &object.Builtin{Name: "ws.upgrade", Fn: wsServerUpgrade})
}
