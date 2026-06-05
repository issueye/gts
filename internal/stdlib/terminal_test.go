package stdlib

import (
	"io"
	"os"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestTerminalStartOptionsTuiLifecycle(t *testing.T) {
	optsHash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(optsHash, "raw", object.TRUE)
	setHashMember(optsHash, "bracketedPaste", object.TRUE)
	setHashMember(optsHash, "mouse", object.TRUE)
	setHashMember(optsHash, "alternateScreen", object.TRUE)
	setHashMember(optsHash, "hideCursor", object.TRUE)
	setHashMember(optsHash, "restoreOnError", object.FALSE)
	setHashMember(optsHash, "restoreOnExit", object.FALSE)

	opts, errObj := terminalStartOptions(ast.Position{}, []object.Object{optsHash})
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Inspect())
	}
	if !opts.raw || !opts.bracketedPaste || !opts.mouse || !opts.alternateScreen || !opts.hideCursor {
		t.Fatalf("expected all TUI lifecycle options to be true: %#v", opts)
	}
	if opts.restoreOnError || opts.restoreOnExit {
		t.Fatalf("expected explicit restore flags to be false: %#v", opts)
	}
}

func TestTerminalStartOptionsRestoreDefaults(t *testing.T) {
	opts, errObj := terminalStartOptions(ast.Position{}, nil)
	if errObj != nil {
		t.Fatalf("unexpected error: %s", errObj.Inspect())
	}
	if !opts.restoreOnError || !opts.restoreOnExit {
		t.Fatalf("expected restore defaults to be true: %#v", opts)
	}
}

func TestTerminalSessionRestoreIsIdempotent(t *testing.T) {
	oldStdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = read.Close()
		_ = write.Close()
	})

	session := &terminalSession{
		bracketedPaste:  true,
		mouse:           true,
		alternateScreen: true,
		cursorHidden:    true,
	}
	if err := session.restore(); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if err := session.restore(); err != nil {
		t.Fatalf("second restore failed: %v", err)
	}

	_ = write.Close()
	data, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	want := "\x1b[?25h\x1b[?1006l\x1b[?1002l\x1b[?1000l\x1b[?2004l\x1b[?1049l"
	if got != want {
		t.Fatalf("unexpected restore sequence:\nwant %q\ngot  %q", want, got)
	}
}
