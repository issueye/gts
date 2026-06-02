package stdlib

import (
	"os"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/process", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initProcessModule(exports)
		return exports, nil
	})
}

func initProcessModule(exports *object.Hash) {
	setHashMember(exports, "argv", strSliceToArray(os.Args))
	setHashMember(exports, "pid", &object.Number{Value: float64(os.Getpid())})
	setHashMember(exports, "env", envObject())
	setHashMember(exports, "cwd", &object.Builtin{Name: "process.cwd", Fn: processCwd})
	setHashMember(exports, "chdir", &object.Builtin{Name: "process.chdir", Fn: processChdir})
	setHashMember(exports, "getenv", &object.Builtin{Name: "process.getenv", Fn: processGetenv})
	setHashMember(exports, "setenv", &object.Builtin{Name: "process.setenv", Fn: processSetenv})
	setHashMember(exports, "unsetenv", &object.Builtin{Name: "process.unsetenv", Fn: processUnsetenv})
	setHashMember(exports, "exit", &object.Builtin{Name: "process.exit", Fn: processExit})
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
