package stdlib

import (
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestTextWidthCJKAndANSI(t *testing.T) {
	env := object.NewEnvironment()
	pos := ast.Position{}

	assertNumber(t, textWidth(env, pos, &object.String{Value: "你"}), 2)
	assertNumber(t, textWidth(env, pos, &object.String{Value: "你好"}), 4)
	assertNumber(t, textWidth(env, pos, &object.String{Value: "\x1b[31m你好\x1b[0m"}), 4)
}

func TestTextTruncateAndPadWidth(t *testing.T) {
	env := object.NewEnvironment()
	pos := ast.Position{}

	assertString(t, textTruncateWidth(env, pos, &object.String{Value: "你好a"}, &object.Number{Value: 4}), "你好")
	assertString(t, textTruncateWidth(env, pos, &object.String{Value: "\x1b[31m你好a\x1b[0m"}, &object.Number{Value: 4}), "\x1b[31m你好\x1b[0m")
	assertString(t, textPadRightWidth(env, pos, &object.String{Value: "你"}, &object.Number{Value: 4}), "你  ")
}

func TestTextWrapWidth(t *testing.T) {
	result := textWrapWidth(object.NewEnvironment(), ast.Position{}, &object.String{Value: "你好世界"}, &object.Number{Value: 4})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("want array, got %T: %s", result, result.Inspect())
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("want 2 lines, got %d", len(arr.Elements))
	}
	assertString(t, arr.Elements[0], "你好")
	assertString(t, arr.Elements[1], "世界")
}

func TestTextCharsStripsANSI(t *testing.T) {
	result := textChars(object.NewEnvironment(), ast.Position{}, &object.String{Value: "\x1b[1m你a\x1b[0m"})
	arr, ok := result.(*object.Array)
	if !ok {
		t.Fatalf("want array, got %T: %s", result, result.Inspect())
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("want 2 chars, got %d", len(arr.Elements))
	}
	assertString(t, arr.Elements[0], "你")
	assertString(t, arr.Elements[1], "a")
}

func assertNumber(t *testing.T, obj object.Object, expected float64) {
	t.Helper()
	n, ok := obj.(*object.Number)
	if !ok {
		t.Fatalf("want number, got %T: %s", obj, obj.Inspect())
	}
	if n.Value != expected {
		t.Fatalf("want %v, got %v", expected, n.Value)
	}
}

func assertString(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	s, ok := obj.(*object.String)
	if !ok {
		t.Fatalf("want string, got %T: %s", obj, obj.Inspect())
	}
	if s.Value != expected {
		t.Fatalf("want %q, got %q", expected, s.Value)
	}
}
