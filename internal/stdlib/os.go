package stdlib

import (
	"os"
	"os/user"
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
	setHashMember(exports, "eol", &object.String{Value: lineEnding()})
	setHashMember(exports, "type", &object.Builtin{Name: "os.type", Fn: osType})
	setHashMember(exports, "release", &object.Builtin{Name: "os.release", Fn: osRelease})
	setHashMember(exports, "homedir", &object.Builtin{Name: "os.homedir", Fn: osHomedir})
	setHashMember(exports, "tmpdir", &object.Builtin{Name: "os.tmpdir", Fn: osTmpdir})
	setHashMember(exports, "hostname", &object.Builtin{Name: "os.hostname", Fn: osHostname})
	setHashMember(exports, "cpus", &object.Builtin{Name: "os.cpus", Fn: osCPUs})
	setHashMember(exports, "userInfo", &object.Builtin{Name: "os.userInfo", Fn: osUserInfo})
}

func osType(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	switch runtime.GOOS {
	case "windows":
		return &object.String{Value: "Windows_NT"}
	case "darwin":
		return &object.String{Value: "Darwin"}
	case "linux":
		return &object.String{Value: "Linux"}
	default:
		return &object.String{Value: runtime.GOOS}
	}
}

func osRelease(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: runtime.GOOS + "/" + runtime.GOARCH}
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

func osCPUs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: float64(runtime.NumCPU())}
}

func osUserInfo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	current, err := user.Current()
	if err != nil {
		return object.NewError(pos, "os.userInfo: %v", err)
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "uid", &object.String{Value: current.Uid})
	setHashMember(out, "gid", &object.String{Value: current.Gid})
	setHashMember(out, "username", &object.String{Value: current.Username})
	setHashMember(out, "name", &object.String{Value: current.Name})
	setHashMember(out, "homedir", &object.String{Value: current.HomeDir})
	return out
}

func lineEnding() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}
