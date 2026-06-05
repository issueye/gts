package stdlib

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestHighlightTerminalDiff(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "lang", &object.String{Value: "diff"})
	result := highlightTerminal(object.NewEnvironment(), ast.Position{}, &object.String{Value: "+ added\n- removed"}, opts)
	hash, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want hash, got %T: %s", result, result.Inspect())
	}
	text, _ := hashValue(hash, "text")
	s, ok := text.(*object.String)
	if !ok {
		t.Fatalf("want text string, got %T: %s", text, text.Inspect())
	}
	if !strings.Contains(s.Value, "\x1b[32m+ added\x1b[0m") || !strings.Contains(s.Value, "\x1b[31m- removed\x1b[0m") {
		t.Fatalf("unexpected highlighted diff: %q", s.Value)
	}
}

func TestHighlightTerminalColorFalse(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "lang", &object.String{Value: "diff"})
	setHashMember(opts, "color", object.FALSE)
	result := highlightTerminal(object.NewEnvironment(), ast.Position{}, &object.String{Value: "+ added"}, opts)
	hash := result.(*object.Hash)
	text, _ := hashValue(hash, "text")
	assertString(t, text, "+ added")
}
