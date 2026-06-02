package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/net/http/client", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initHTTPClientModule(exports)
		return exports, nil
	})
}

func initHTTPClientModule(exports *object.Hash) {
	setHashMember(exports, "get", &object.Builtin{Name: "http.get", Fn: httpClientGet})
	setHashMember(exports, "post", &object.Builtin{Name: "http.post", Fn: httpClientPost})
	setHashMember(exports, "request", &object.Builtin{Name: "http.request", Fn: httpClientRequest})
	setHashMember(exports, "fetch", &object.Builtin{Name: "http.fetch", Fn: httpClientRequest})
}
