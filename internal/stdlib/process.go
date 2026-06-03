package stdlib

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

var processStartedAt = time.Now()

func init() {
	module.RegisterNative("@std/process", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initProcessModule(exports, env)
		return exports, nil
	})
}

func initProcessModule(exports *object.Hash, env *object.Environment) {
	argv := runtimeArgv(env)
	setHashMember(exports, "argv", strSliceToArray(argv))
	if len(argv) > 0 {
		setHashMember(exports, "argv0", &object.String{Value: argv[0]})
	} else {
		setHashMember(exports, "argv0", &object.String{Value: ""})
	}
	setHashMember(exports, "pid", &object.Number{Value: float64(os.Getpid())})
	setHashMember(exports, "env", envObject())
	setHashMember(exports, "cwd", &object.Builtin{Name: "process.cwd", Fn: processCwd})
	setHashMember(exports, "chdir", &object.Builtin{Name: "process.chdir", Fn: processChdir})
	setHashMember(exports, "execPath", &object.Builtin{Name: "process.execPath", Fn: processExecPath})
	setHashMember(exports, "getenv", &object.Builtin{Name: "process.getenv", Fn: processGetenv})
	setHashMember(exports, "envObject", &object.Builtin{Name: "process.envObject", Fn: processEnvObject})
	setHashMember(exports, "uptime", &object.Builtin{Name: "process.uptime", Fn: processUptime})
	setHashMember(exports, "hrtime", &object.Builtin{Name: "process.hrtime", Fn: processHrtime})
	setHashMember(exports, "setenv", &object.Builtin{Name: "process.setenv", Fn: processSetenv})
	setHashMember(exports, "unsetenv", &object.Builtin{Name: "process.unsetenv", Fn: processUnsetenv})
	setHashMember(exports, "exit", &object.Builtin{Name: "process.exit", Fn: processExit})
	setHashMember(exports, "version", &object.String{Value: "0.1.0-dev"})
}

func runtimeArgv(env *object.Environment) []string {
	if env != nil {
		if argv := env.VM().Argv(); len(argv) > 0 {
			return argv
		}
	}
	return append([]string{}, os.Args...)
}

func processCwd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cwd, err := os.Getwd()
	if err != nil {
		return object.NewError(pos, "process.cwd: %v", err)
	}
	return &object.String{Value: cwd}
}

func processChdir(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "process.chdir", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if err := os.Chdir(path); err != nil {
		return object.NewError(pos, "process.chdir: %v", err)
	}
	return object.UNDEFINED
}

func processExecPath(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, err := os.Executable()
	if err != nil {
		return object.NewError(pos, "process.execPath: %v", err)
	}
	return &object.String{Value: path}
}

func processGetenv(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	name, errObj := requiredString(pos, "process.getenv", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	value, ok := os.LookupEnv(name)
	if !ok {
		if len(args) >= 2 {
			return args[1]
		}
		return object.UNDEFINED
	}
	return &object.String{Value: value}
}

func processEnvObject(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return envObject()
}

func processUptime(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: time.Since(processStartedAt).Seconds()}
}

func processHrtime(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	elapsed := time.Since(processStartedAt)
	seconds := int64(elapsed / time.Second)
	nanos := int64(elapsed % time.Second)
	if len(args) >= 1 {
		if prev, ok := args[0].(*object.Array); ok && len(prev.Elements) >= 2 {
			if secObj, ok := prev.Elements[0].(*object.Number); ok {
				if nanoObj, ok := prev.Elements[1].(*object.Number); ok {
					baseSeconds := int64(secObj.Value)
					baseNanos := int64(nanoObj.Value)
					seconds -= baseSeconds
					nanos -= baseNanos
					if nanos < 0 {
						seconds--
						nanos += int64(time.Second)
					}
				}
			}
		}
	}
	return &object.Array{Elements: []object.Object{
		&object.Number{Value: float64(seconds)},
		&object.Number{Value: float64(nanos)},
	}}
}

func processSetenv(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	name, errObj := requiredString(pos, "process.setenv", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	value, errObj := requiredString(pos, "process.setenv", args, 1, "value")
	if errObj != nil {
		return errObj
	}
	if err := os.Setenv(name, value); err != nil {
		return object.NewError(pos, "process.setenv: %v", err)
	}
	return object.UNDEFINED
}

func processUnsetenv(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	name, errObj := requiredString(pos, "process.unsetenv", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	if err := os.Unsetenv(name); err != nil {
		return object.NewError(pos, "process.unsetenv: %v", err)
	}
	return object.UNDEFINED
}

func processExit(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	code := 0
	if len(args) >= 1 {
		switch v := args[0].(type) {
		case *object.Number:
			code = int(v.Value)
		case *object.String:
			parsed, err := strconv.Atoi(v.Value)
			if err != nil {
				return object.NewError(pos, "process.exit: code must be a number")
			}
			code = parsed
		default:
			return object.NewError(pos, "process.exit: code must be a number")
		}
	}
	os.Exit(code)
	return object.UNDEFINED
}

func envObject() *object.Hash {
	env := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		setHashMember(env, parts[0], &object.String{Value: value})
	}
	return env
}
