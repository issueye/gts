package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/http/server", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initHTTPServerModule(exports)
		return exports, nil
	})
}

func initHTTPServerModule(exports *object.Hash) {
	setHashMember(exports, "createServer", &object.Builtin{Name: "http.createServer", Fn: httpServerCreateServer})
}
