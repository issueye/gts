package stdlib

import (
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestColorRGB(t *testing.T) {
	result := colorRgb(
		object.NewEnvironment(),
		ast.Position{},
		&object.Number{Value: 1},
		&object.Number{Value: 2},
		&object.Number{Value: 3},
	)
	hash, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want hash, got %T", result)
	}
	call, ok := getHashValue(hash, "_call").(*object.Builtin)
	if !ok {
		t.Fatalf("want _call builtin, got %T", getHashValue(hash, "_call"))
	}
	assertString(t, call.Fn(object.NewEnvironment(), ast.Position{}, &object.String{Value: "x"}), "\x1b[38;2;1;2;3mx\x1b[0m")
}

func TestColorRGBRejectsInvalidChannels(t *testing.T) {
	tests := []object.Object{
		&object.Number{Value: 1.5},
		&object.Number{Value: -1},
		&object.Number{Value: 256},
	}

	for _, value := range tests {
		result := colorRgb(
			object.NewEnvironment(),
			ast.Position{},
			value,
			&object.Number{Value: 2},
			&object.Number{Value: 3},
		)
		if _, ok := result.(*object.Error); !ok {
			t.Fatalf("want error for %s, got %T", value.Inspect(), result)
		}
	}
}

