package stdlib

import (
	"os"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"golang.org/x/term"
)

type terminalRawMode struct {
	fd    int
	state *term.State
}

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
