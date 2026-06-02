package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/ws/client", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initWSClientModule(exports)
		return exports, nil
	})
}

func initWSClientModule(exports *object.Hash) {
	setHashMember(exports, "connect", &object.Builtin{Name: "ws.connect", Fn: wsClientConnect})
}
