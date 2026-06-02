package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/socket/client", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSocketClientModule(exports)
		return exports, nil
	})
}

func initSocketClientModule(exports *object.Hash) {
	setHashMember(exports, "connect", &object.Builtin{Name: "socket.connect", Fn: socketClientConnect})
	setHashMember(exports, "dial", &object.Builtin{Name: "socket.dial", Fn: socketClientConnect})
}
