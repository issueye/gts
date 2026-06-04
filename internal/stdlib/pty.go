package stdlib

import (
	"bufio"
	"context"
	"io"
	"os"
	"sync"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type ptyProcess struct {
	pty        gopty.Pty
	cmd        *gopty.Cmd
	stream     *readableStream
	outputMu   sync.Mutex
	pending    []byte
	output     chan ptyOutputChunk
	outputOnce sync.Once
	cancel     context.CancelFunc
	waitOnce   sync.Once
	waitResult object.Object
	waitErr    error
}

type ptyOutputChunk struct {
	data []byte
	err  error
}

func init() {
	module.RegisterNative("@std/pty", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initPTYModule(exports)
		return exports, nil
	})
}

func initPTYModule(exports *object.Hash) {
	setHashMember(exports, "spawn", &object.Builtin{Name: "pty.spawn", Fn: ptySpawn})
	setHashMember(exports, "open", &object.Builtin{Name: "pty.open", Fn: ptySpawn})
}

func ptySpawn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "pty.spawn requires a command name")
	}
	cmdName, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "pty.spawn: first argument must be a string (command name)")
	}
	cmdArgs, opts, errObj := spawnArgsAndOptions(pos, args[1:])
	if errObj != nil {
		return errObj
	}
	pty, err := gopty.New()
	if err != nil {
		return object.NewError(pos, "pty.spawn: %v", err)
	}
	cols := 80
	rows := 24
	if opts != nil {
		if n, ok := hashNumber(opts, "cols"); ok && n > 0 {
			cols = int(n)
		}
		if n, ok := hashNumber(opts, "rows"); ok && n > 0 {
			rows = int(n)
		}
	}
	_ = pty.Resize(cols, rows)
	var cancel context.CancelFunc
	ctx := context.Background()
	if timeoutMs, ok := hashNumber(opts, "timeoutMs"); ok && timeoutMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	}
	cmd := pty.CommandContext(ctx, cmdName.Value, cmdArgs...)
	if opts != nil {
		applyPTYOptions(cmd, opts)
	}
	if err := cmd.Start(); err != nil {
		_ = pty.Close()
		if cancel != nil {
			cancel()
		}
		return object.NewError(pos, "pty.spawn: %v", err)
	}
	proc := &ptyProcess{
		pty:    pty,
		cmd:    cmd,
		stream: &readableStream{reader: bufio.NewReader(pty), closer: pty},
		cancel: cancel,
	}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	if cmd.Process != nil {
		setHashMember(obj, "pid", &object.Number{Value: float64(cmd.Process.Pid)})
	}
	setHashMember(obj, "name", &object.String{Value: pty.Name()})
	setHashMember(obj, "output", readableStreamObject(proc.stream))
	setHashMember(obj, "write", &object.Builtin{Name: "pty.write", Fn: ptyWrite, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "writeln", &object.Builtin{Name: "pty.writeln", Fn: ptyWriteln, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "read", &object.Builtin{Name: "pty.read", Fn: ptyRead, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "readText", &object.Builtin{Name: "pty.readText", Fn: ptyReadText, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "readLine", &object.Builtin{Name: "pty.readLine", Fn: ptyReadLine, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "readTextTimeout", &object.Builtin{Name: "pty.readTextTimeout", Fn: ptyReadTextTimeout, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "resize", &object.Builtin{Name: "pty.resize", Fn: ptyResize, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "wait", &object.Builtin{Name: "pty.wait", Fn: ptyWait, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "kill", &object.Builtin{Name: "pty.kill", Fn: ptyKill, Extra: &object.GoObject{Value: proc}})
	setHashMember(obj, "close", &object.Builtin{Name: "pty.close", Fn: ptyClose, Extra: &object.GoObject{Value: proc}})
	return obj
}

func applyPTYOptions(cmd *gopty.Cmd, opts *object.Hash) {
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

func ptyWrite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.write")
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return object.NewError(pos, "pty.write requires data")
	}
	n, err := proc.pty.Write([]byte(objectToText(args[0])))
	if err != nil {
		return object.NewError(pos, "pty.write: %v", err)
	}
	return &object.Number{Value: float64(n)}
}

func ptyWriteln(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return ptyWrite(env, pos, &object.String{Value: "\n"})
	}
	return ptyWrite(env, pos, &object.String{Value: objectToText(args[0]) + "\n"})
}

func ptyRead(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.read")
	if errObj != nil {
		return errObj
	}
	return streamRead(&object.Environment{Extra: &object.GoObject{Value: proc.stream}}, pos, args...)
}

func ptyReadText(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.readText")
	if errObj != nil {
		return errObj
	}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		return ptyReadTextTimeoutWithName(pos, proc, "pty.readText", args...)
	}
	return streamReadText(&object.Environment{Extra: &object.GoObject{Value: proc.stream}}, pos, args...)
}

func ptyReadLine(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.readLine")
	if errObj != nil {
		return errObj
	}
	return streamReadLine(&object.Environment{Extra: &object.GoObject{Value: proc.stream}}, pos, args...)
}

func ptyReadTextTimeout(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.readTextTimeout")
	if errObj != nil {
		return errObj
	}
	return ptyReadTextTimeoutWithName(pos, proc, "pty.readTextTimeout", args...)
}

func ptyReadTextTimeoutWithName(pos ast.Position, proc *ptyProcess, name string, args ...object.Object) object.Object {
	size := 8192
	timeoutMs := 0
	if len(args) >= 1 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "%s: size must be a number", name)
		}
		size = int(n.Value)
		if size < 1 {
			return object.NewError(pos, "%s: size must be positive", name)
		}
	}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "%s: timeoutMs must be a number", name)
		}
		timeoutMs = int(n.Value)
		if timeoutMs < 0 {
			return object.NewError(pos, "%s: timeoutMs must be non-negative", name)
		}
	}
	chunk, timedOut := proc.readOutputChunk(size, timeoutMs)
	if timedOut {
		return object.NULL
	}
	if chunk.err == io.EOF {
		return object.NULL
	}
	if chunk.err != nil {
		return object.NewError(pos, "%s: %v", name, chunk.err)
	}
	if len(chunk.data) > size {
		data := make([]byte, size)
		copy(data, chunk.data[:size])
		proc.pushBackOutput(chunk.data[size:])
		return &object.String{Value: string(data)}
	}
	return &object.String{Value: string(chunk.data)}
}

func ptyResize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.resize")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "pty.resize requires cols and rows")
	}
	cols, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "pty.resize: cols must be a number")
	}
	rows, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "pty.resize: rows must be a number")
	}
	if err := proc.pty.Resize(int(cols.Value), int(rows.Value)); err != nil {
		return object.NewError(pos, "pty.resize: %v", err)
	}
	return object.UNDEFINED
}

func ptyWait(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.wait")
	if errObj != nil {
		return errObj
	}
	proc.waitOnce.Do(func() {
		err := proc.cmd.Wait()
		if proc.cancel != nil {
			proc.cancel()
		}
		exitCode := 0
		if proc.cmd.ProcessState != nil {
			exitCode = proc.cmd.ProcessState.ExitCode()
		}
		if err != nil && exitCode == 0 {
			proc.waitErr = err
			return
		}
		proc.waitResult = processResult(exitCode, "", "")
	})
	if proc.waitErr != nil {
		return object.NewError(pos, "pty.wait: %v", proc.waitErr)
	}
	if proc.waitResult == nil {
		return object.NewError(pos, "pty.wait: missing wait result")
	}
	return proc.waitResult
}

func ptyKill(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.kill")
	if errObj != nil {
		return errObj
	}
	if proc.cmd.Process == nil {
		return object.NewError(pos, "pty.kill: process has not started")
	}
	if err := proc.cmd.Process.Kill(); err != nil {
		return object.NewError(pos, "pty.kill: %v", err)
	}
	if proc.cancel != nil {
		proc.cancel()
	}
	return object.UNDEFINED
}

func ptyClose(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	proc, errObj := boundPTYProcess(pos, env, "pty.close")
	if errObj != nil {
		return errObj
	}
	if err := proc.pty.Close(); err != nil {
		return object.NewError(pos, "pty.close: %v", err)
	}
	if proc.cancel != nil {
		proc.cancel()
	}
	return object.UNDEFINED
}

func (p *ptyProcess) ensureOutputReader() {
	p.outputOnce.Do(func() {
		p.output = make(chan ptyOutputChunk, 128)
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := p.pty.Read(buf)
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					p.output <- ptyOutputChunk{data: data}
				}
				if err != nil {
					p.output <- ptyOutputChunk{err: err}
					close(p.output)
					return
				}
			}
		}()
	})
}

func (p *ptyProcess) readOutputChunk(size, timeoutMs int) (ptyOutputChunk, bool) {
	p.ensureOutputReader()
	if pending := p.takePending(size); len(pending) > 0 {
		return ptyOutputChunk{data: pending}, false
	}
	if timeoutMs <= 0 {
		chunk, ok := <-p.output
		if !ok {
			return ptyOutputChunk{err: io.EOF}, false
		}
		return chunk, false
	}
	timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
	defer timer.Stop()
	select {
	case chunk, ok := <-p.output:
		if !ok {
			return ptyOutputChunk{err: io.EOF}, false
		}
		return chunk, false
	case <-timer.C:
		return ptyOutputChunk{}, true
	}
}

func (p *ptyProcess) pushBackOutput(data []byte) {
	if len(data) == 0 {
		return
	}
	p.outputMu.Lock()
	next := make([]byte, 0, len(data)+len(p.pending))
	next = append(next, data...)
	next = append(next, p.pending...)
	p.pending = next
	p.outputMu.Unlock()
}

func (p *ptyProcess) takePending(size int) []byte {
	p.outputMu.Lock()
	defer p.outputMu.Unlock()
	if len(p.pending) == 0 {
		return nil
	}
	if len(p.pending) <= size {
		data := p.pending
		p.pending = nil
		return data
	}
	data := make([]byte, size)
	copy(data, p.pending[:size])
	rest := make([]byte, len(p.pending)-size)
	copy(rest, p.pending[size:])
	p.pending = rest
	return data
}

func boundPTYProcess(pos ast.Position, env *object.Environment, name string) (*ptyProcess, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing pty receiver", name)
	}
	proc, ok := goObj.Value.(*ptyProcess)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid pty receiver", name)
	}
	return proc, nil
}
