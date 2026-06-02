package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/socket/server", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSocketServerModule(exports)
		return exports, nil
	})
}

func initSocketServerModule(exports *object.Hash) {
	setHashMember(exports, "listen", &object.Builtin{Name: "socket.listen", Fn: socketServerListen})
	setHashMember(exports, "createServer", &object.Builtin{Name: "socket.createServer", Fn: socketServerListen})
}
