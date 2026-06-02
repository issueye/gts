package stdlib

import (
	"os"
	"runtime"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/os", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initOSModule(exports)
		return exports, nil
	})
}

func initOSModule(exports *object.Hash) {
	setHashMember(exports, "platform", &object.String{Value: runtime.GOOS})
	setHashMember(exports, "arch", &object.String{Value: runtime.GOARCH})
	setHashMember(exports, "homedir", &object.Builtin{Name: "os.homedir", Fn: osHomedir})
	setHashMember(exports, "tmpdir", &object.Builtin{Name: "os.tmpdir", Fn: osTmpdir})
	setHashMember(exports, "hostname", &object.Builtin{Name: "os.hostname", Fn: osHostname})
}

func osHomedir(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	dir, err := os.UserHomeDir()
	if err != nil {
		return object.NewError(pos, "os.homedir: %v", err)
	}
	return &object.String{Value: dir}
}

func osTmpdir(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: os.TempDir()}
}

func osHostname(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	name, err := os.Hostname()
	if err != nil {
		return object.NewError(pos, "os.hostname: %v", err)
	}
	return &object.String{Value: name}
}
