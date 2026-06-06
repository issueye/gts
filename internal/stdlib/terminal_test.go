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
	setHashMember(optsHash, "resizeDebounceMs", &object.Number{Value: 75})

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
	if opts.resizeDebounceMs != 75 {
		t.Fatalf("expected resize debounce 75ms, got %#v", opts)
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
	if opts.resizeDebounceMs != 50 {
		t.Fatalf("expected resize debounce default to be 50ms: %#v", opts)
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

func TestTerminalClearSequenceRequiresExplicitScrollback(t *testing.T) {
	if got := terminalClearSequence(terminalClearOptions{screen: true}); got != "\x1b[2J\x1b[H" {
		t.Fatalf("clear screen sequence = %q", got)
	}
	if got := terminalClearSequence(terminalClearOptions{screen: true, scrollback: true}); got != "\x1b[3J\x1b[2J\x1b[H" {
		t.Fatalf("clear scrollback sequence = %q", got)
	}
	if got := terminalClearSequence(terminalClearOptions{scrollback: true}); got != "\x1b[3J" {
		t.Fatalf("clear scrollback-only sequence = %q", got)
	}
}

func TestTerminalRenderFrameClipsRowsAndColumns(t *testing.T) {
	opts := terminalFrameOptions{rows: 2, cols: 4, clip: true, full: true}
	seq, lines := terminalBuildFrameSequence("hello\n世界ab\nextra", opts, nil)
	if len(lines) != 2 {
		t.Fatalf("want 2 clipped lines, got %#v", lines)
	}
	if lines[0] != "hell" || lines[1] != "世界" {
		t.Fatalf("unexpected clipped lines: %#v", lines)
	}
	want := "\x1b[2J\x1b[Hhell\x1b[2;1H世界"
	if seq != want {
		t.Fatalf("unexpected frame sequence:\nwant %q\ngot  %q", want, seq)
	}
}

func TestTerminalRenderFrameDiffOnlyWritesChangedRows(t *testing.T) {
	opts := terminalFrameOptions{rows: 3, cols: 20, clip: true, diff: true}
	previous := []string{"one", "two", "three"}
	seq, lines := terminalBuildFrameSequence("one\nTWO", opts, previous)
	if len(lines) != 2 {
		t.Fatalf("want normalized next frame, got %#v", lines)
	}
	want := "\x1b[2;1H\x1b[2KTWO\x1b[3;1H\x1b[2K"
	if seq != want {
		t.Fatalf("unexpected diff sequence:\nwant %q\ngot  %q", want, seq)
	}
}
