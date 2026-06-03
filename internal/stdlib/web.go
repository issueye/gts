package stdlib

import (
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/web", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initWebModule(exports)
		return exports, nil
	})
	module.RegisterNative("@std/express", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initWebModule(exports)
		return exports, nil
	})
}

func initWebModule(exports *object.Hash) {
	setHashMember(exports, "createApp", &object.Builtin{Name: "web.createApp", Fn: webCreateApp})
	setHashMember(exports, "json", &object.Builtin{Name: "web.json", Fn: webJSON})
	setHashMember(exports, "text", &object.Builtin{Name: "web.text", Fn: webText})
	setHashMember(exports, "static", &object.Builtin{Name: "web.static", Fn: webStatic})
}
