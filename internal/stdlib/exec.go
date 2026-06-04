package stdlib

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type spawnedProcess struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	cancel     context.CancelFunc
	waitOnce   sync.Once
	waitResult object.Object
	waitErr    error
}

func init() {
	module.RegisterNative("@std/exec", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initExecModule(exports)
		return exports, nil
	})
}

func initExecModule(exports *object.Hash) {
	setHashMember(exports, "run", &object.Builtin{Name: "exec.run", Fn: execRun})
	setHashMember(exports, "output", &object.Builtin{Name: "exec.output", Fn: execOutput})
	setHashMember(exports, "start", &object.Builtin{Name: "exec.start", Fn: execStart})
	setHashMember(exports, "spawn", &object.Builtin{Name: "exec.spawn", Fn: execSpawn})
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
	return processResult(exitCode, stdout.String(), stderr.String())
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

func execSpawn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "exec.spawn requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "exec.spawn: first argument must be a string (command name)")
	}
	cmdArgs, opts, errObj := spawnArgsAndOptions(pos, args[1:])
	if errObj != nil {
		return errObj
	}
	var cancel context.CancelFunc
	ctx := context.Background()
	if timeoutMs, ok := hashNumber(opts, "timeoutMs"); ok && timeoutMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	}
	cmd := exec.CommandContext(ctx, cmdName.Value, cmdArgs...)
	if opts != nil {
		applyCommandOptions(cmd, opts)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return object.NewError(pos, "exec.spawn: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return object.NewError(pos, "exec.spawn: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return object.NewError(pos, "exec.spawn: %v", err)
	}
	if err := cmd.Start(); err != nil {
		if cancel != nil {
			cancel()
		}
		return object.NewError(pos, "exec.spawn: %v", err)
	}
	proc := &spawnedProcess{cmd: cmd, stdin: stdin, cancel: cancel}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "pid", &object.Number{Value: float64(cmd.Process.Pid)})
	setHashMember(obj, "stdout", newReadableStream(stdout, stdout))
	setHashMember(obj, "stderr", newReadableStream(stderr, stderr))
	setHashMember(obj, "stdin", stdinObject(proc))
	setHashMember(obj, "write", &object.Builtin{Name: "process.write", Fn: spawnWrite, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "writeln", &object.Builtin{Name: "process.writeln", Fn: spawnWriteln, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "closeStdin", &object.Builtin{Name: "process.closeStdin", Fn: spawnCloseStdin, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "wait", &object.Builtin{Name: "process.wait", Fn: spawnWait, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "kill", &object.Builtin{Name: "process.kill", Fn: spawnKill, Extra: &object.GoObject{Value: proc}})
	return obj
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
			return processResult(exitCode, stdout.String(), stderr.String())
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

func spawnWrite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundSpawnedProcess(pos, env, "process.write")
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return object.NewError(pos, "process.write requires data")
	}
	if proc.stdin == nil {
		return object.NewError(pos, "process.write: stdin is closed")
	}
	text := objectToText(args[0])
	n, err := proc.stdin.Write([]byte(text))
	if err != nil {
		return object.NewError(pos, "process.write: %v", err)
	}
	return &object.Number{Value: float64(n)}
}

func spawnWriteln(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return spawnWrite(env, pos, &object.String{Value: "\n"})
	}
	return spawnWrite(env, pos, &object.String{Value: objectToText(args[0]) + "\n"})
}

func spawnCloseStdin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundSpawnedProcess(pos, env, "process.closeStdin")
	if errObj != nil {
		return errObj
	}
	if proc.stdin == nil {
		return object.UNDEFINED
	}
	err := proc.stdin.Close()
	proc.stdin = nil
	if err != nil {
		return object.NewError(pos, "process.closeStdin: %v", err)
	}
	return object.UNDEFINED
}

func spawnWait(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundSpawnedProcess(pos, env, "process.wait")
	if errObj != nil {
		return errObj
	}
	proc.waitOnce.Do(func() {
		if proc.stdin != nil {
			_ = proc.stdin.Close()
			proc.stdin = nil
		}
		err := proc.cmd.Wait()
		if proc.cancel != nil {
			proc.cancel()
		}
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				proc.waitErr = err
				return
			}
		}
		proc.waitResult = processResult(exitCode, "", "")
	})
	if proc.waitErr != nil {
		return object.NewError(pos, "process.wait: %v", proc.waitErr)
	}
	if proc.waitResult == nil {
		return object.NewError(pos, "process.wait: missing wait result")
	}
	return proc.waitResult
}

func spawnKill(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundSpawnedProcess(pos, env, "process.kill")
	if errObj != nil {
		return errObj
	}
	if proc.cmd.Process == nil {
		return object.NewError(pos, "process.kill: process has not started")
	}
	if err := proc.cmd.Process.Kill(); err != nil {
		return object.NewError(pos, "process.kill: %v", err)
	}
	if proc.cancel != nil {
		proc.cancel()
	}
	return object.UNDEFINED
}

func stdinObject(proc *spawnedProcess) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "write", &object.Builtin{Name: "stdin.write", Fn: spawnWrite, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "writeln", &object.Builtin{Name: "stdin.writeln", Fn: spawnWriteln, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "close", &object.Builtin{Name: "stdin.close", Fn: spawnCloseStdin, Extra: &object.GoObject{Value: proc}})
	return obj
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
			for _, pair := range h.OrderedPairs() {
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
			return processResult(exitCode, stdout.String(), stderr.String())
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

func spawnArgsAndOptions(pos ast.Position, args []object.Object) ([]string, *object.Hash, *object.Error) {
	if len(args) == 0 {
		return nil, nil, nil
	}
	if arr, ok := args[0].(*object.Array); ok {
		cmdArgs := toStringSlice(arr.Elements)
		if len(args) >= 2 {
			if opts, ok := args[1].(*object.Hash); ok {
				return cmdArgs, opts, nil
			}
			return nil, nil, object.NewError(pos, "exec.spawn: options must be an object")
		}
		return cmdArgs, nil, nil
	}
	if opts, ok := args[0].(*object.Hash); ok {
		return nil, opts, nil
	}
	return extractArgs(args), nil, nil
}

func applyCommandOptions(cmd *exec.Cmd, opts *object.Hash) {
	if dir, ok := hashString(opts, "cwd"); ok {
		cmd.Dir = dir
	} else if dir, ok := hashString(opts, "dir"); ok {
		cmd.Dir = dir
	}
	if envObj, ok := hashValue(opts, "env"); ok {
		if h, ok := envObj.(*object.Hash); ok {
			envVars := os.Environ()
			for _, pair := range h.OrderedPairs() {
				envVars = upsertEnv(envVars, objectToMapKey(pair.Key), objectToText(pair.Value))
			}
			cmd.Env = envVars
		}
	}
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func hashNumber(hash *object.Hash, key string) (float64, bool) {
	if hash == nil {
		return 0, false
	}
	value, ok := hashValue(hash, key)
	if !ok {
		return 0, false
	}
	n, ok := value.(*object.Number)
	if !ok {
		return 0, false
	}
	return n.Value, true
}

func processResult(exitCode int, stdout, stderr string) *object.Hash {
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "stdout", &object.String{Value: stdout})
	setHashMember(result, "stderr", &object.String{Value: stderr})
	setHashMember(result, "exitCode", &object.Number{Value: float64(exitCode)})
	setHashMember(result, "success", object.NativeBool(exitCode == 0))
	return result
}

func boundSpawnedProcess(pos ast.Position, env *object.Environment, name string) (*spawnedProcess, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing process receiver", name)
	}
	proc, ok := goObj.Value.(*spawnedProcess)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid process receiver", name)
	}
	return proc, nil
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
