package stdlib

import (
	"bytes"
	"os/exec"
	"syscall"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/exec", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initExecModule(exports)
		return exports, nil
	})
}

func initExecModule(exports *object.Hash) {
	setHashMember(exports, "run", &object.Builtin{Name: "exec.run", Fn: execRun})
	setHashMember(exports, "output", &object.Builtin{Name: "exec.output", Fn: execOutput})
	setHashMember(exports, "start", &object.Builtin{Name: "exec.start", Fn: execStart})
	setHashMember(exports, "command", &object.Builtin{Name: "exec.command", Fn: execCommand})
	setHashMember(exports, "combinedOutput", &object.Builtin{Name: "exec.combinedOutput", Fn: execCombinedOutput})
}

func execRun(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.run requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.run: first argument must be a string (command name)")
	}
	cmdArgs := extractArgs(args[1:])
	cmd := exec.Command(cmdName.Value, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = exitErr.ExitCode()
			}
		} else {
			return object.NewError(pos, "exec.run: %v", err)
		}
	}
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.Pairs[hashKey(&object.String{Value: "stdout"})] = object.HashPair{
		Key: &object.String{Value: "stdout"}, Value: &object.String{Value: stdout.String()},
	}
	result.Pairs[hashKey(&object.String{Value: "stderr"})] = object.HashPair{
		Key: &object.String{Value: "stderr"}, Value: &object.String{Value: stderr.String()},
	}
	result.Pairs[hashKey(&object.String{Value: "exitCode"})] = object.HashPair{
		Key: &object.String{Value: "exitCode"}, Value: &object.Number{Value: float64(exitCode)},
	}
	result.Pairs[hashKey(&object.String{Value: "success"})] = object.HashPair{
		Key: &object.String{Value: "success"}, Value: object.NativeBool(exitCode == 0),
	}
	return result
}

func execOutput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.output requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.output: first argument must be a string (command name)")
	}
	cmdArgs := extractArgs(args[1:])
	cmd := exec.Command(cmdName.Value, cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		return object.NewError(pos, "exec.output: %v", err)
	}
	return &object.String{Value: string(out)}
}

func execCombinedOutput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.combinedOutput requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.combinedOutput: first argument must be a string (command name)")
	}
	cmdArgs := extractArgs(args[1:])
	cmd := exec.Command(cmdName.Value, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return object.NewError(pos, "exec.combinedOutput: %v", err)
	}
	return &object.String{Value: string(out)}
}

func execStart(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.start requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.start: first argument must be a string (command name)")
	}
	cmdArgs := extractArgs(args[1:])
	cmd := exec.Command(cmdName.Value, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return object.NewError(pos, "exec.start: %v", err)
	}
	proc := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(proc, "pid", &object.Number{Value: float64(cmd.Process.Pid)})
	setHashMember(proc, "wait", &object.Builtin{
		Name: "process.wait",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			err := cmd.Wait()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return object.NewError(pos, "process.wait: %v", err)
				}
			}
			result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			result.Pairs[hashKey(&object.String{Value: "stdout"})] = object.HashPair{
				Key: &object.String{Value: "stdout"}, Value: &object.String{Value: stdout.String()},
			}
			result.Pairs[hashKey(&object.String{Value: "stderr"})] = object.HashPair{
				Key: &object.String{Value: "stderr"}, Value: &object.String{Value: stderr.String()},
			}
			result.Pairs[hashKey(&object.String{Value: "exitCode"})] = object.HashPair{
				Key: &object.String{Value: "exitCode"}, Value: &object.Number{Value: float64(exitCode)},
			}
			result.Pairs[hashKey(&object.String{Value: "success"})] = object.HashPair{
				Key: &object.String{Value: "success"}, Value: object.NativeBool(exitCode == 0),
			}
			return result
		},
	})
	setHashMember(proc, "kill", &object.Builtin{
		Name: "process.kill",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if err := cmd.Process.Kill(); err != nil {
				return object.NewError(pos, "process.kill: %v", err)
			}
			return object.UNDEFINED
		},
	})
	return proc
}

func execCommand(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.command requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.command: first argument must be a string (command name)")
	}
	cmdArgs := extractArgs(args[1:])
	cmd := exec.Command(cmdName.Value, cmdArgs...)
	cmdObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(cmdObj, "name", &object.String{Value: cmdName.Value})
	setHashMember(cmdObj, "args", strSliceToArray(cmdArgs))
	setHashMember(cmdObj, "setDir", &object.Builtin{
		Name: "cmd.setDir",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "cmd.setDir requires a directory path")
			}
			s, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "cmd.setDir: argument must be a string")
			}
			cmd.Dir = s.Value
			return object.UNDEFINED
		},
	})
	setHashMember(cmdObj, "setEnv", &object.Builtin{
		Name: "cmd.setEnv",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "cmd.setEnv requires an env object")
			}
			h, ok := args[0].(*object.Hash)
			if !ok {
				return object.NewError(pos, "cmd.setEnv: argument must be an object")
			}
			envVars := make([]string, 0, len(h.Pairs))
			for _, pair := range h.Pairs {
				envVars = append(envVars, pair.Key.Inspect()+"="+pair.Value.Inspect())
			}
			cmd.Env = envVars
			return object.UNDEFINED
		},
	})
	setHashMember(cmdObj, "run", &object.Builtin{
		Name: "cmd.run",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					return object.NewError(pos, "cmd.run: %v", err)
				}
			}
			result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			result.Pairs[hashKey(&object.String{Value: "stdout"})] = object.HashPair{
				Key: &object.String{Value: "stdout"}, Value: &object.String{Value: stdout.String()},
			}
			result.Pairs[hashKey(&object.String{Value: "stderr"})] = object.HashPair{
				Key: &object.String{Value: "stderr"}, Value: &object.String{Value: stderr.String()},
			}
			result.Pairs[hashKey(&object.String{Value: "exitCode"})] = object.HashPair{
				Key: &object.String{Value: "exitCode"}, Value: &object.Number{Value: float64(exitCode)},
			}
			return result
		},
	})
	setHashMember(cmdObj, "output", &object.Builtin{
		Name: "cmd.output",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			out, err := cmd.Output()
			if err != nil {
				return object.NewError(pos, "cmd.output: %v", err)
			}
			return &object.String{Value: string(out)}
		},
	})
	return cmdObj
}

func extractArgs(args []object.Object) []string {
	if len(args) == 0 {
		return nil
	}
	if len(args) == 1 {
		if arr, ok := args[0].(*object.Array); ok {
			return toStringSlice(arr.Elements)
		}
	}
	return toStringSlice(args)
}

func toStringSlice(args []object.Object) []string {
	result := make([]string, 0, len(args))
	for _, a := range args {
		if s, ok := a.(*object.String); ok {
			result = append(result, s.Value)
		} else {
			result = append(result, a.Inspect())
		}
	}
	return result
}

func strSliceToArray(strs []string) *object.Array {
	elements := make([]object.Object, len(strs))
	for i, s := range strs {
		elements[i] = &object.String{Value: s}
	}
	return &object.Array{Elements: elements}
}
