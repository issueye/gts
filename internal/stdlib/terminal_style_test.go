package stdlib

import (
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestTerminalStyle(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "bold", object.TRUE)
	setHashMember(opts, "fg", &object.String{Value: "accent"})
	result := terminalStyle(object.NewEnvironment(), ast.Position{}, &object.String{Value: "Title"}, opts)
	assertString(t, result, "\x1b[1;36mTitle\x1b[0m")
	assertNumber(t, textWidth(object.NewEnvironment(), ast.Position{}, result), 5)
}

func TestTerminalStyleColorFalse(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "bold", object.TRUE)
	setHashMember(opts, "color", object.FALSE)
	result := terminalStyle(object.NewEnvironment(), ast.Position{}, &object.String{Value: "plain"}, opts)
	assertString(t, result, "plain")
}

func TestTerminalHyperlink(t *testing.T) {
	result := terminalHyperlink(
		object.NewEnvironment(),
		ast.Position{},
		&object.String{Value: "Open"},
		&object.String{Value: "https://example.com"},
	)
	assertString(t, result, "\x1b]8;;https://example.com\x1b\\Open\x1b]8;;\x1b\\")
	assertNumber(t, textWidth(object.NewEnvironment(), ast.Position{}, result), 4)

	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "enabled", object.FALSE)
	fallback := terminalHyperlink(
		object.NewEnvironment(),
		ast.Position{},
		&object.String{Value: "Open"},
		&object.String{Value: "https://example.com"},
		opts,
	)
	assertString(t, fallback, "Open <https://example.com>")
}
