package stdlib

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"golang.org/x/term"
)

type terminalRawMode struct {
	fd    int
	state *term.State
}

type terminalSession struct {
	mu              sync.Mutex
	vm              *object.VirtualMachine
	raw             *terminalRawMode
	bracketedPaste  bool
	mouse           bool
	alternateScreen bool
	cursorHidden    bool
	onInput         *object.Function
	onResize        *object.Function
	onError         *object.Function
	restoreOnError  bool
	restoreOnExit   bool
	moduleEvent     string
	events          chan terminalEvent
	stop            chan struct{}
	stopped         bool
	asyncRegistered bool
	lastCols        int
	lastRows        int
}

type terminalEvent struct {
	kind string
	data string
	cols int
	rows int
}

var activeTerminalSessions = struct {
	sync.Mutex
	sessions map[*terminalSession]struct{}
}{sessions: make(map[*terminalSession]struct{})}

func init() {
	module.RegisterNative("@std/terminal", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTerminalModule(exports)
		return exports, nil
	})
}

func initTerminalModule(exports *object.Hash) {
	setHashMember(exports, "isTTY", &object.Builtin{Name: "terminal.isTTY", Fn: terminalIsTTY})
	setHashMember(exports, "size", &object.Builtin{Name: "terminal.size", Fn: terminalSize})
	setHashMember(exports, "read", &object.Builtin{Name: "terminal.read", Fn: terminalRead})
	setHashMember(exports, "write", &object.Builtin{Name: "terminal.write", Fn: terminalWrite})
	setHashMember(exports, "writeln", &object.Builtin{Name: "terminal.writeln", Fn: terminalWriteln})
	setHashMember(exports, "setRawMode", &object.Builtin{Name: "terminal.setRawMode", Fn: terminalSetRawMode})
	setHashMember(exports, "start", &object.Builtin{Name: "terminal.start", Fn: terminalStart})
	setHashMember(exports, "onInput", &object.Builtin{Name: "terminal.onInput", Fn: terminalOnInput})
	setHashMember(exports, "offInput", &object.Builtin{Name: "terminal.offInput", Fn: terminalOffInput})
	setHashMember(exports, "onResize", &object.Builtin{Name: "terminal.onResize", Fn: terminalOnResize})
	setHashMember(exports, "offResize", &object.Builtin{Name: "terminal.offResize", Fn: terminalOffResize})
	setHashMember(exports, "hideCursor", &object.Builtin{Name: "terminal.hideCursor", Fn: terminalHideCursor})
	setHashMember(exports, "showCursor", &object.Builtin{Name: "terminal.showCursor", Fn: terminalShowCursor})
	setHashMember(exports, "clearScreen", &object.Builtin{Name: "terminal.clearScreen", Fn: terminalClearScreen})
	setHashMember(exports, "clearLine", &object.Builtin{Name: "terminal.clearLine", Fn: terminalClearLine})
	setHashMember(exports, "clearFromCursor", &object.Builtin{Name: "terminal.clearFromCursor", Fn: terminalClearFromCursor})
	setHashMember(exports, "moveTo", &object.Builtin{Name: "terminal.moveTo", Fn: terminalMoveTo})
	setHashMember(exports, "moveBy", &object.Builtin{Name: "terminal.moveBy", Fn: terminalMoveBy})
	setHashMember(exports, "setTitle", &object.Builtin{Name: "terminal.setTitle", Fn: terminalSetTitle})
	setHashMember(exports, "style", &object.Builtin{Name: "terminal.style", Fn: terminalStyle})
	setHashMember(exports, "hyperlink", &object.Builtin{Name: "terminal.hyperlink", Fn: terminalHyperlink})
	setHashMember(exports, "enterAlternateScreen", &object.Builtin{Name: "terminal.enterAlternateScreen", Fn: terminalEnterAlternateScreen})
	setHashMember(exports, "leaveAlternateScreen", &object.Builtin{Name: "terminal.leaveAlternateScreen", Fn: terminalLeaveAlternateScreen})
	setHashMember(exports, "enableMouse", &object.Builtin{Name: "terminal.enableMouse", Fn: terminalEnableMouse})
	setHashMember(exports, "disableMouse", &object.Builtin{Name: "terminal.disableMouse", Fn: terminalDisableMouse})
	setHashMember(exports, "enableBracketedPaste", &object.Builtin{Name: "terminal.enableBracketedPaste", Fn: terminalEnableBracketedPaste})
	setHashMember(exports, "disableBracketedPaste", &object.Builtin{Name: "terminal.disableBracketedPaste", Fn: terminalDisableBracketedPaste})
}

func terminalIsTTY(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	fd := int(os.Stdout.Fd())
	if len(args) >= 1 {
		if s, ok := args[0].(*object.String); ok {
			fd = terminalFD(s.Value)
		}
	}
	return object.NativeBool(term.IsTerminal(fd))
}

func terminalSize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	fd := int(os.Stdout.Fd())
	if len(args) >= 1 {
		if s, ok := args[0].(*object.String); ok {
			fd = terminalFD(s.Value)
		}
	}
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		cols, rows = terminalSizeFromEnv()
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "cols", &object.Number{Value: float64(cols)})
	setHashMember(out, "rows", &object.Number{Value: float64(rows)})
	return out
}

func terminalRead(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	size := 1
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "terminal.read: size must be a number")
		}
		size = int(n.Value)
		if size < 1 {
			return object.NewError(pos, "terminal.read: size must be positive")
		}
	}
	buf := make([]byte, size)
	n, err := os.Stdin.Read(buf)
	if err != nil {
		return object.NewError(pos, "terminal.read: %v", err)
	}
	return &object.String{Value: string(buf[:n])}
}

func terminalWrite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.write requires text")
	}
	n, err := os.Stdout.Write([]byte(objectToText(args[0])))
	if err != nil {
		return object.NewError(pos, "terminal.write: %v", err)
	}
	return &object.Number{Value: float64(n)}
}

func terminalWriteln(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return terminalWrite(env, pos, &object.String{Value: "\n"})
	}
	return terminalWrite(env, pos, &object.String{Value: objectToText(args[0]) + "\n"})
}

func terminalSetRawMode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	enabled := true
	if len(args) >= 1 {
		b, ok := args[0].(*object.Boolean)
		if !ok {
			return object.NewError(pos, "terminal.setRawMode: enabled must be a boolean")
		}
		enabled = b.Value
	}
	fd := int(os.Stdin.Fd())
	if !enabled {
		return object.UNDEFINED
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return object.NewError(pos, "terminal.setRawMode: %v", err)
	}
	mode := &terminalRawMode{fd: fd, state: state}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "restore", &object.Builtin{Name: "terminal.restoreRawMode", Fn: terminalRestoreRawMode, Extra: &object.GoObject{Value: mode}})
	return obj
}

func terminalStart(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	opts, errObj := terminalStartOptions(pos, args)
	if errObj != nil {
		return errObj
	}
	return terminalStartSession(env, pos, opts, "")
}

func terminalStartSession(env *object.Environment, pos ast.Position, opts terminalOptions, moduleEvent string) object.Object {
	session := &terminalSession{
		vm:             env.VM(),
		onInput:        opts.onInput,
		onResize:       opts.onResize,
		onError:        opts.onError,
		restoreOnError: opts.restoreOnError,
		restoreOnExit:  opts.restoreOnExit,
		moduleEvent:    moduleEvent,
		events:         make(chan terminalEvent, 256),
		stop:           make(chan struct{}),
	}
	session.lastCols, session.lastRows = terminalGetSize()

	if opts.raw {
		raw, err := terminalMakeRaw()
		if err != nil {
			return object.NewError(pos, "terminal.start: %v", err)
		}
		session.raw = raw
	}
	if opts.bracketedPaste {
		if _, err := os.Stdout.Write([]byte("\x1b[?2004h")); err != nil {
			session.restore()
			return object.NewError(pos, "terminal.start: %v", err)
		}
		session.bracketedPaste = true
	}
	if opts.mouse {
		if _, err := os.Stdout.Write([]byte("\x1b[?1000h\x1b[?1002h\x1b[?1006h")); err != nil {
			session.restore()
			return object.NewError(pos, "terminal.start: %v", err)
		}
		session.mouse = true
	}
	if opts.alternateScreen {
		if _, err := os.Stdout.Write([]byte("\x1b[?1049h")); err != nil {
			session.restore()
			return object.NewError(pos, "terminal.start: %v", err)
		}
		session.alternateScreen = true
	}
	if opts.hideCursor {
		if _, err := os.Stdout.Write([]byte("\x1b[?25l")); err != nil {
			session.restore()
			return object.NewError(pos, "terminal.start: %v", err)
		}
		session.cursorHidden = true
	}

	registerTerminalSession(session)
	if session.onInput != nil || session.onResize != nil {
		session.startAsyncLoops()
	}
	return terminalSessionObject(session)
}

func terminalOnInput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.onInput requires listener")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "terminal.onInput: listener must be a function")
	}
	opts := terminalDefaultOptions()
	opts.onInput = fn
	return terminalStartSession(env, pos, opts, "input")
}

func terminalOffInput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.offInput requires listener")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "terminal.offInput: listener must be a function")
	}
	return &object.Number{Value: float64(stopModuleTerminalSessions(env.VM(), "input", fn))}
}

func terminalOnResize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.onResize requires listener")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "terminal.onResize: listener must be a function")
	}
	opts := terminalDefaultOptions()
	opts.onResize = fn
	return terminalStartSession(env, pos, opts, "resize")
}

func terminalOffResize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.offResize requires listener")
	}
	fn, ok := args[0].(*object.Function)
	if !ok {
		return object.NewError(pos, "terminal.offResize: listener must be a function")
	}
	return &object.Number{Value: float64(stopModuleTerminalSessions(env.VM(), "resize", fn))}
}

type terminalOptions struct {
	raw             bool
	bracketedPaste  bool
	mouse           bool
	alternateScreen bool
	hideCursor      bool
	onInput         *object.Function
	onResize        *object.Function
	onError         *object.Function
	restoreOnError  bool
	restoreOnExit   bool
}

func terminalStartOptions(pos ast.Position, args []object.Object) (terminalOptions, *object.Error) {
	opts := terminalDefaultOptions()
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return opts, nil
	}
	hash, ok := args[0].(*object.Hash)
	if !ok {
		return opts, object.NewError(pos, "terminal.start: options must be an object")
	}
	if rawObj, ok := hashValue(hash, "raw"); ok && rawObj != object.UNDEFINED && rawObj != object.NULL {
		raw, ok := rawObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: raw must be a boolean")
		}
		opts.raw = raw.Value
	}
	if pasteObj, ok := hashValue(hash, "bracketedPaste"); ok && pasteObj != object.UNDEFINED && pasteObj != object.NULL {
		paste, ok := pasteObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: bracketedPaste must be a boolean")
		}
		opts.bracketedPaste = paste.Value
	}
	if mouseObj, ok := hashValue(hash, "mouse"); ok && mouseObj != object.UNDEFINED && mouseObj != object.NULL {
		mouse, ok := mouseObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: mouse must be a boolean")
		}
		opts.mouse = mouse.Value
	}
	if screenObj, ok := hashValue(hash, "alternateScreen"); ok && screenObj != object.UNDEFINED && screenObj != object.NULL {
		screen, ok := screenObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: alternateScreen must be a boolean")
		}
		opts.alternateScreen = screen.Value
	}
	if cursorObj, ok := hashValue(hash, "hideCursor"); ok && cursorObj != object.UNDEFINED && cursorObj != object.NULL {
		cursor, ok := cursorObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: hideCursor must be a boolean")
		}
		opts.hideCursor = cursor.Value
	}
	if fnObj, ok := hashValue(hash, "onInput"); ok && fnObj != object.UNDEFINED && fnObj != object.NULL {
		fn, ok := fnObj.(*object.Function)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: onInput must be a function")
		}
		opts.onInput = fn
	}
	if fnObj, ok := hashValue(hash, "onResize"); ok && fnObj != object.UNDEFINED && fnObj != object.NULL {
		fn, ok := fnObj.(*object.Function)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: onResize must be a function")
		}
		opts.onResize = fn
	}
	if fnObj, ok := hashValue(hash, "onError"); ok && fnObj != object.UNDEFINED && fnObj != object.NULL {
		fn, ok := fnObj.(*object.Function)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: onError must be a function")
		}
		opts.onError = fn
	}
	if restoreObj, ok := hashValue(hash, "restoreOnError"); ok && restoreObj != object.UNDEFINED && restoreObj != object.NULL {
		restore, ok := restoreObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: restoreOnError must be a boolean")
		}
		opts.restoreOnError = restore.Value
	}
	if restoreObj, ok := hashValue(hash, "restoreOnExit"); ok && restoreObj != object.UNDEFINED && restoreObj != object.NULL {
		restore, ok := restoreObj.(*object.Boolean)
		if !ok {
			return opts, object.NewError(pos, "terminal.start: restoreOnExit must be a boolean")
		}
		opts.restoreOnExit = restore.Value
	}
	return opts, nil
}

func terminalDefaultOptions() terminalOptions {
	return terminalOptions{restoreOnError: true, restoreOnExit: true}
}

func terminalSessionObject(session *terminalSession) *object.Hash {
	extra := &object.GoObject{Value: session}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__terminalSession", extra)
	setHashMember(obj, "write", &object.Builtin{Name: "terminal.session.write", Fn: terminalSessionWrite, Extra: extra})
	setHashMember(obj, "writeln", &object.Builtin{Name: "terminal.session.writeln", Fn: terminalSessionWriteln, Extra: extra})
	setHashMember(obj, "size", &object.Builtin{Name: "terminal.session.size", Fn: terminalSessionSize, Extra: extra})
	setHashMember(obj, "setRawMode", &object.Builtin{Name: "terminal.session.setRawMode", Fn: terminalSessionSetRawMode, Extra: extra})
	setHashMember(obj, "restore", &object.Builtin{Name: "terminal.session.restore", Fn: terminalSessionRestore, Extra: extra})
	setHashMember(obj, "stop", &object.Builtin{Name: "terminal.session.stop", Fn: terminalSessionStop, Extra: extra})
	setHashMember(obj, "drainInput", &object.Builtin{Name: "terminal.session.drainInput", Fn: terminalSessionDrainInput, Extra: extra})
	setHashMember(obj, "hideCursor", &object.Builtin{Name: "terminal.session.hideCursor", Fn: terminalSessionHideCursor, Extra: extra})
	setHashMember(obj, "showCursor", &object.Builtin{Name: "terminal.session.showCursor", Fn: terminalSessionShowCursor, Extra: extra})
	setHashMember(obj, "clearScreen", &object.Builtin{Name: "terminal.session.clearScreen", Fn: terminalClearScreen})
	setHashMember(obj, "clearLine", &object.Builtin{Name: "terminal.session.clearLine", Fn: terminalClearLine})
	setHashMember(obj, "clearFromCursor", &object.Builtin{Name: "terminal.session.clearFromCursor", Fn: terminalClearFromCursor})
	setHashMember(obj, "moveTo", &object.Builtin{Name: "terminal.session.moveTo", Fn: terminalMoveTo})
	setHashMember(obj, "moveBy", &object.Builtin{Name: "terminal.session.moveBy", Fn: terminalMoveBy})
	setHashMember(obj, "setTitle", &object.Builtin{Name: "terminal.session.setTitle", Fn: terminalSetTitle})
	setHashMember(obj, "enterAlternateScreen", &object.Builtin{Name: "terminal.session.enterAlternateScreen", Fn: terminalSessionEnterAlternateScreen, Extra: extra})
	setHashMember(obj, "leaveAlternateScreen", &object.Builtin{Name: "terminal.session.leaveAlternateScreen", Fn: terminalSessionLeaveAlternateScreen, Extra: extra})
	setHashMember(obj, "enableMouse", &object.Builtin{Name: "terminal.session.enableMouse", Fn: terminalSessionEnableMouse, Extra: extra})
	setHashMember(obj, "disableMouse", &object.Builtin{Name: "terminal.session.disableMouse", Fn: terminalSessionDisableMouse, Extra: extra})
	setHashMember(obj, "enableBracketedPaste", &object.Builtin{Name: "terminal.session.enableBracketedPaste", Fn: terminalSessionEnableBracketedPaste, Extra: extra})
	setHashMember(obj, "disableBracketedPaste", &object.Builtin{Name: "terminal.session.disableBracketedPaste", Fn: terminalSessionDisableBracketedPaste, Extra: extra})
	return obj
}

func terminalRestoreRawMode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return object.NewError(pos, "terminal.restoreRawMode: missing raw mode receiver")
	}
	mode, ok := goObj.Value.(*terminalRawMode)
	if !ok {
		return object.NewError(pos, "terminal.restoreRawMode: invalid raw mode receiver")
	}
	if mode.state == nil {
		return object.UNDEFINED
	}
	if err := term.Restore(mode.fd, mode.state); err != nil {
		return object.NewError(pos, "terminal.restoreRawMode: %v", err)
	}
	mode.state = nil
	return object.UNDEFINED
}

func terminalSessionWrite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWrite(env, pos, args...)
}

func terminalSessionWriteln(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteln(env, pos, args...)
}

func terminalSessionSize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalSize(env, pos, args...)
}

func terminalSessionSetRawMode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.setRawMode")
	if errObj != nil {
		return errObj
	}
	enabled := true
	if len(args) >= 1 {
		b, ok := args[0].(*object.Boolean)
		if !ok {
			return object.NewError(pos, "terminal.session.setRawMode: enabled must be a boolean")
		}
		enabled = b.Value
	}
	if err := session.setRawMode(enabled); err != nil {
		return object.NewError(pos, "terminal.session.setRawMode: %v", err)
	}
	return object.UNDEFINED
}

func terminalSessionRestore(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.restore")
	if errObj != nil {
		return errObj
	}
	if err := session.restore(); err != nil {
		return object.NewError(pos, "terminal.session.restore: %v", err)
	}
	return object.UNDEFINED
}

func terminalSessionStop(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.stop")
	if errObj != nil {
		return errObj
	}
	if err := session.stopSession(); err != nil {
		return object.NewError(pos, "terminal.session.stop: %v", err)
	}
	return object.UNDEFINED
}

func terminalSessionDrainInput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.drainInput")
	if errObj != nil {
		return errObj
	}
	maxMs := 50
	idleMs := 10
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "terminal.session.drainInput: maxMs must be a number")
		}
		maxMs = int(n.Value)
	}
	if len(args) >= 2 {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "terminal.session.drainInput: idleMs must be a number")
		}
		idleMs = int(n.Value)
	}
	if maxMs < 0 {
		maxMs = 0
	}
	if idleMs < 0 {
		idleMs = 0
	}
	deadline := time.Now().Add(time.Duration(maxMs) * time.Millisecond)
	idle := time.NewTimer(time.Duration(idleMs) * time.Millisecond)
	defer idle.Stop()
	drained := 0
	for {
		select {
		case <-idle.C:
			return &object.Number{Value: float64(drained)}
		case event := <-session.events:
			if event.kind == "input" {
				drained++
			}
			if time.Now().After(deadline) {
				return &object.Number{Value: float64(drained)}
			}
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(time.Duration(idleMs) * time.Millisecond)
		default:
			if time.Now().After(deadline) {
				return &object.Number{Value: float64(drained)}
			}
			time.Sleep(time.Millisecond)
		}
	}
}

func terminalSessionHideCursor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.hideCursor")
	if errObj != nil {
		return errObj
	}
	if result := terminalHideCursor(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.cursorHidden = true
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionShowCursor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.showCursor")
	if errObj != nil {
		return errObj
	}
	if result := terminalShowCursor(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.cursorHidden = false
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionEnableBracketedPaste(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.enableBracketedPaste")
	if errObj != nil {
		return errObj
	}
	if result := terminalEnableBracketedPaste(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.bracketedPaste = true
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionDisableBracketedPaste(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.disableBracketedPaste")
	if errObj != nil {
		return errObj
	}
	if result := terminalDisableBracketedPaste(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.bracketedPaste = false
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionEnterAlternateScreen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.enterAlternateScreen")
	if errObj != nil {
		return errObj
	}
	if result := terminalEnterAlternateScreen(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.alternateScreen = true
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionLeaveAlternateScreen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.leaveAlternateScreen")
	if errObj != nil {
		return errObj
	}
	if result := terminalLeaveAlternateScreen(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.alternateScreen = false
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionEnableMouse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.enableMouse")
	if errObj != nil {
		return errObj
	}
	if result := terminalEnableMouse(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.mouse = true
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalSessionDisableMouse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	session, errObj := boundTerminalSession(pos, env, "terminal.session.disableMouse")
	if errObj != nil {
		return errObj
	}
	if result := terminalDisableMouse(env, pos, args...); object.IsRuntimeError(result) {
		return result
	}
	session.mu.Lock()
	session.mouse = false
	session.mu.Unlock()
	return object.UNDEFINED
}

func terminalHideCursor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?25l")
}

func terminalShowCursor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?25h")
}

func terminalClearScreen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[2J\x1b[H")
}

func terminalClearLine(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[2K")
}

func terminalClearFromCursor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[J")
}

func terminalMoveTo(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "terminal.moveTo requires row and col")
	}
	row, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "terminal.moveTo: row must be a number")
	}
	col, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "terminal.moveTo: col must be a number")
	}
	r := int(row.Value)
	c := int(col.Value)
	if r < 1 {
		r = 1
	}
	if c < 1 {
		c = 1
	}
	return terminalWriteANSI(pos, fmt.Sprintf("\x1b[%d;%dH", r, c))
}

func terminalMoveBy(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "terminal.moveBy requires rows and cols")
	}
	rows, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "terminal.moveBy: rows must be a number")
	}
	cols, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "terminal.moveBy: cols must be a number")
	}
	seq := strings.Builder{}
	if rows.Value < 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dA", int(-rows.Value)))
	} else if rows.Value > 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dB", int(rows.Value)))
	}
	if cols.Value < 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dD", int(-cols.Value)))
	} else if cols.Value > 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dC", int(cols.Value)))
	}
	return terminalWriteANSI(pos, seq.String())
}

func terminalSetTitle(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.setTitle requires title")
	}
	return terminalWriteANSI(pos, "\x1b]0;"+objectToText(args[0])+"\x07")
}

type terminalStyleOptions struct {
	bold      bool
	dim       bool
	underline bool
	inverse   bool
	fg        string
	bg        string
	color     bool
}

func terminalStyle(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "terminal.style requires text")
	}
	opts := terminalStyleOptions{color: true}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "terminal.style: options must be an object")
		}
		var errObj *object.Error
		if opts.bold, errObj = terminalStyleBool(pos, hash, "bold", opts.bold); errObj != nil {
			return errObj
		}
		if opts.dim, errObj = terminalStyleBool(pos, hash, "dim", opts.dim); errObj != nil {
			return errObj
		}
		if opts.underline, errObj = terminalStyleBool(pos, hash, "underline", opts.underline); errObj != nil {
			return errObj
		}
		if opts.inverse, errObj = terminalStyleBool(pos, hash, "inverse", opts.inverse); errObj != nil {
			return errObj
		}
		if opts.color, errObj = terminalStyleBool(pos, hash, "color", opts.color); errObj != nil {
			return errObj
		}
		if opts.fg, errObj = terminalStyleStringOption(pos, hash, "fg"); errObj != nil {
			return errObj
		}
		if opts.bg, errObj = terminalStyleStringOption(pos, hash, "bg"); errObj != nil {
			return errObj
		}
	}
	return &object.String{Value: terminalStyleString(objectToText(args[0]), opts)}
}

func terminalHyperlink(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	label, errObj := requiredString(pos, "terminal.hyperlink", args, 0, "label")
	if errObj != nil {
		return errObj
	}
	url, errObj := requiredString(pos, "terminal.hyperlink", args, 1, "url")
	if errObj != nil {
		return errObj
	}
	enabled := true
	if len(args) >= 3 && args[2] != object.UNDEFINED && args[2] != object.NULL {
		opts, ok := args[2].(*object.Hash)
		if !ok {
			return object.NewError(pos, "terminal.hyperlink: options must be an object")
		}
		if value, ok := hashValue(opts, "enabled"); ok && value != object.UNDEFINED && value != object.NULL {
			b, ok := value.(*object.Boolean)
			if !ok {
				return object.NewError(pos, "terminal.hyperlink: enabled must be a boolean")
			}
			enabled = b.Value
		}
	}
	if !enabled {
		return &object.String{Value: label + " <" + url + ">"}
	}
	return &object.String{Value: "\x1b]8;;" + url + "\x1b\\" + label + "\x1b]8;;\x1b\\"}
}

func terminalStyleBool(pos ast.Position, hash *object.Hash, key string, fallback bool) (bool, *object.Error) {
	value, ok := hashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return fallback, nil
	}
	b, ok := value.(*object.Boolean)
	if !ok {
		return false, object.NewError(pos, "terminal.style: %s must be a boolean", key)
	}
	return b.Value, nil
}

func terminalStyleStringOption(pos ast.Position, hash *object.Hash, key string) (string, *object.Error) {
	value, ok := hashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return "", nil
	}
	s, ok := value.(*object.String)
	if !ok {
		return "", object.NewError(pos, "terminal.style: %s must be a string", key)
	}
	return s.Value, nil
}

func terminalStyleString(text string, opts terminalStyleOptions) string {
	if !opts.color {
		return text
	}
	var codes []string
	if opts.bold {
		codes = append(codes, "1")
	}
	if opts.dim {
		codes = append(codes, "2")
	}
	if opts.underline {
		codes = append(codes, "4")
	}
	if opts.inverse {
		codes = append(codes, "7")
	}
	if code := terminalColorCode(opts.fg, false); code != "" {
		codes = append(codes, code)
	}
	if code := terminalColorCode(opts.bg, true); code != "" {
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func terminalColorCode(name string, background bool) string {
	if name == "" {
		return ""
	}
	colors := map[string]int{
		"black":   30,
		"red":     31,
		"green":   32,
		"yellow":  33,
		"blue":    34,
		"magenta": 35,
		"cyan":    36,
		"white":   37,
		"gray":    90,
		"grey":    90,
		"muted":   90,
		"accent":  36,
		"error":   31,
		"success": 32,
		"warning": 33,
	}
	code, ok := colors[strings.ToLower(name)]
	if !ok {
		return ""
	}
	if background {
		if code >= 90 {
			return fmt.Sprintf("%d", code+10)
		}
		return fmt.Sprintf("%d", code+10)
	}
	return fmt.Sprintf("%d", code)
}

func terminalEnterAlternateScreen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?1049h")
}

func terminalLeaveAlternateScreen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?1049l")
}

func terminalEnableMouse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?1000h\x1b[?1002h\x1b[?1006h")
}

func terminalDisableMouse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?1006l\x1b[?1002l\x1b[?1000l")
}

func terminalEnableBracketedPaste(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?2004h")
}

func terminalDisableBracketedPaste(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return terminalWriteANSI(pos, "\x1b[?2004l")
}

func terminalWriteANSI(pos ast.Position, text string) object.Object {
	n, err := os.Stdout.Write([]byte(text))
	if err != nil {
		return object.NewError(pos, "terminal.write: %v", err)
	}
	return &object.Number{Value: float64(n)}
}

func boundTerminalSession(pos ast.Position, env *object.Environment, name string) (*terminalSession, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing terminal session receiver", name)
	}
	session, ok := goObj.Value.(*terminalSession)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid terminal session receiver", name)
	}
	return session, nil
}

func (s *terminalSession) startAsyncLoops() {
	s.mu.Lock()
	if s.asyncRegistered {
		s.mu.Unlock()
		return
	}
	s.asyncRegistered = true
	s.mu.Unlock()

	s.vm.AsyncAdd(1)
	s.vm.Go(func() {
		defer s.vm.AsyncDone()
		s.eventLoop()
	})
	if s.onInput != nil {
		go s.readInputLoop()
	}
	if s.onResize != nil {
		go s.resizeLoop()
	}
}

func (s *terminalSession) eventLoop() {
	for {
		select {
		case <-s.stop:
			return
		case event := <-s.events:
			switch event.kind {
			case "input":
				if s.onInput != nil {
					result := callTerminalFunction(s.onInput, nil, []object.Object{&object.String{Value: event.data}})
					if object.IsRuntimeError(result) {
						s.handleCallbackError(result)
						return
					}
				}
			case "resize":
				if s.onResize != nil {
					size := terminalSizeObject(event.cols, event.rows)
					result := callTerminalFunction(s.onResize, nil, []object.Object{size})
					if object.IsRuntimeError(result) {
						s.handleCallbackError(result)
						return
					}
				}
			}
		}
	}
}

func (s *terminalSession) readInputLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if !s.sendEvent(terminalEvent{kind: "input", data: string(buf[:n])}) {
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "terminal.start: input read: %v\n", err)
			}
			return
		}
		select {
		case <-s.stop:
			return
		default:
		}
	}
}

func (s *terminalSession) resizeLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			cols, rows := terminalGetSize()
			s.mu.Lock()
			changed := cols != s.lastCols || rows != s.lastRows
			if changed {
				s.lastCols = cols
				s.lastRows = rows
			}
			s.mu.Unlock()
			if changed && !s.sendEvent(terminalEvent{kind: "resize", cols: cols, rows: rows}) {
				return
			}
		}
	}
}

func (s *terminalSession) sendEvent(event terminalEvent) bool {
	select {
	case <-s.stop:
		return false
	case s.events <- event:
		return true
	default:
		select {
		case <-s.events:
		default:
		}
		select {
		case <-s.stop:
			return false
		case s.events <- event:
			return true
		default:
			return true
		}
	}
}

func (s *terminalSession) setRawMode(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		if s.raw != nil && s.raw.state != nil {
			return nil
		}
		raw, err := terminalMakeRaw()
		if err != nil {
			return err
		}
		s.raw = raw
		return nil
	}
	return s.restoreRawLocked()
}

func (s *terminalSession) stopSession() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	close(s.stop)
	var err error
	if s.restoreOnExit {
		err = s.restoreLocked()
	}
	s.mu.Unlock()
	unregisterTerminalSession(s)
	return err
}

func (s *terminalSession) restore() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restoreLocked()
}

func (s *terminalSession) restoreLocked() error {
	var errs []string
	if s.cursorHidden {
		if _, err := os.Stdout.Write([]byte("\x1b[?25h")); err != nil {
			errs = append(errs, err.Error())
		}
		s.cursorHidden = false
	}
	if s.mouse {
		if _, err := os.Stdout.Write([]byte("\x1b[?1006l\x1b[?1002l\x1b[?1000l")); err != nil {
			errs = append(errs, err.Error())
		}
		s.mouse = false
	}
	if s.bracketedPaste {
		if _, err := os.Stdout.Write([]byte("\x1b[?2004l")); err != nil {
			errs = append(errs, err.Error())
		}
		s.bracketedPaste = false
	}
	if s.alternateScreen {
		if _, err := os.Stdout.Write([]byte("\x1b[?1049l")); err != nil {
			errs = append(errs, err.Error())
		}
		s.alternateScreen = false
	}
	if err := s.restoreRawLocked(); err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *terminalSession) handleCallbackError(result object.Object) {
	if s.restoreOnError {
		_ = s.restore()
		_ = s.stopWithoutRestore()
	} else {
		_ = s.stopWithoutRestore()
	}
	if s.onError != nil {
		errorObj := result
		callbackResult := callTerminalFunction(s.onError, nil, []object.Object{errorObj, terminalSessionObject(s)})
		if object.IsRuntimeError(callbackResult) {
			fmt.Fprintln(os.Stderr, callbackResult.Inspect())
		}
		return
	}
	fmt.Fprintln(os.Stderr, result.Inspect())
}

func (s *terminalSession) stopWithoutRestore() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	close(s.stop)
	s.mu.Unlock()
	unregisterTerminalSession(s)
	return nil
}

func (s *terminalSession) restoreRawLocked() error {
	if s.raw == nil || s.raw.state == nil {
		return nil
	}
	if err := term.Restore(s.raw.fd, s.raw.state); err != nil {
		return err
	}
	s.raw.state = nil
	return nil
}

func terminalMakeRaw() (*terminalRawMode, error) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	if err := terminalEnableVirtualTerminalInput(fd); err != nil {
		_ = term.Restore(fd, state)
		return nil, err
	}
	return &terminalRawMode{fd: fd, state: state}, nil
}

func registerTerminalSession(session *terminalSession) {
	activeTerminalSessions.Lock()
	activeTerminalSessions.sessions[session] = struct{}{}
	activeTerminalSessions.Unlock()
}

func unregisterTerminalSession(session *terminalSession) {
	activeTerminalSessions.Lock()
	delete(activeTerminalSessions.sessions, session)
	activeTerminalSessions.Unlock()
}

func StopTerminalSessionsForVM(vm *object.VirtualMachine) {
	if vm == nil {
		return
	}
	activeTerminalSessions.Lock()
	sessions := make([]*terminalSession, 0, len(activeTerminalSessions.sessions))
	for session := range activeTerminalSessions.sessions {
		if session.vm == vm {
			sessions = append(sessions, session)
		}
	}
	activeTerminalSessions.Unlock()
	for _, session := range sessions {
		_ = session.stopSession()
	}
}

func StopAllTerminalSessions() {
	activeTerminalSessions.Lock()
	sessions := make([]*terminalSession, 0, len(activeTerminalSessions.sessions))
	for session := range activeTerminalSessions.sessions {
		sessions = append(sessions, session)
	}
	activeTerminalSessions.Unlock()
	for _, session := range sessions {
		_ = session.stopSession()
	}
}

func stopModuleTerminalSessions(vm *object.VirtualMachine, moduleEvent string, fn *object.Function) int {
	activeTerminalSessions.Lock()
	sessions := make([]*terminalSession, 0, len(activeTerminalSessions.sessions))
	for session := range activeTerminalSessions.sessions {
		if session.vm != vm || session.moduleEvent != moduleEvent {
			continue
		}
		if moduleEvent == "input" && session.onInput == fn {
			sessions = append(sessions, session)
		}
		if moduleEvent == "resize" && session.onResize == fn {
			sessions = append(sessions, session)
		}
	}
	activeTerminalSessions.Unlock()
	for _, session := range sessions {
		_ = session.stopSession()
	}
	return len(sessions)
}

func terminalGetSize() (int, int) {
	cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		cols, rows = terminalSizeFromEnv()
	}
	return cols, rows
}

func terminalSizeObject(cols, rows int) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "cols", &object.Number{Value: float64(cols)})
	setHashMember(out, "rows", &object.Number{Value: float64(rows)})
	return out
}

func callTerminalFunction(fn *object.Function, this object.Object, args []object.Object) object.Object {
	scope := fn.Env.NewScope()
	if this != nil {
		scope.Set("this", this)
	}
	for i, p := range fn.Parameters {
		if i < len(args) {
			if p.Spread {
				rest := make([]object.Object, len(args)-i)
				copy(rest, args[i:])
				scope.Set(p.Name, fn.Env.ObjectManager().NewArray(rest))
				break
			}
			scope.Set(p.Name, args[i])
		} else if p.Default != nil {
			scope.Set(p.Name, fn.Env.VM().EvalNode(p.Default, fn.Env))
		} else {
			scope.Set(p.Name, object.UNDEFINED)
		}
	}
	result := fn.Env.VM().EvalNode(fn.Body, scope)
	if rv, ok := result.(*object.ReturnValue); ok {
		return rv.Value
	}
	return result
}

func terminalFD(name string) int {
	switch name {
	case "stdin":
		return int(os.Stdin.Fd())
	case "stderr":
		return int(os.Stderr.Fd())
	default:
		return int(os.Stdout.Fd())
	}
}

func terminalSizeFromEnv() (int, int) {
	cols := envInt("COLUMNS", 80)
	rows := envInt("LINES", 24)
	return cols, rows
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	n := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return fallback
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return fallback
	}
	return n
}
