package stdlib

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

func TestTUIParseInputMessage(t *testing.T) {
	cases := []struct {
		raw  string
		typ  string
		key  string
		text string
	}{
		{"\x03", "key", "ctrl+c", ""},
		{"\x1b[A", "key", "up", ""},
		{"hello", "text", "", "hello"},
	}
	for _, tc := range cases {
		msg := tuiParseInputMessage(tc.raw)
		hash, ok := msg.(*object.Hash)
		if !ok {
			t.Fatalf("message must be object, got %T", msg)
		}
		if got, _ := tuiHashString(hash, "type"); got != tc.typ {
			t.Fatalf("type for %q = %q, want %q", tc.raw, got, tc.typ)
		}
		if tc.key != "" {
			if got, _ := tuiHashString(hash, "key"); got != tc.key {
				t.Fatalf("key for %q = %q, want %q", tc.raw, got, tc.key)
			}
		}
		if tc.text != "" {
			if got, _ := tuiHashString(hash, "text"); got != tc.text {
				t.Fatalf("text for %q = %q, want %q", tc.raw, got, tc.text)
			}
		}
	}
}

func TestTUIBoxAndRow(t *testing.T) {
	boxA := tuiBox(object.NewEnvironment(), ast.Position{}, &object.String{Value: "A"}, tuiOptions(map[string]object.Object{
		"width": &object.Number{Value: 7},
		"title": &object.String{Value: "one"},
	}))
	boxB := tuiBox(object.NewEnvironment(), ast.Position{}, &object.String{Value: "B"}, tuiOptions(map[string]object.Object{
		"width": &object.Number{Value: 5},
	}))
	row := tuiRow(object.NewEnvironment(), ast.Position{}, boxA, boxB)
	got, ok := row.(*object.String)
	if !ok {
		t.Fatalf("row must be string, got %T: %s", row, row.Inspect())
	}
	if !strings.Contains(got.Value, "one") || !strings.Contains(got.Value, "A") || !strings.Contains(got.Value, "B") {
		t.Fatalf("unexpected row render:\n%s", got.Value)
	}
}

func TestTUIInput(t *testing.T) {
	result := tuiInput(object.NewEnvironment(), ast.Position{}, tuiOptions(map[string]object.Object{
		"width": &object.Number{Value: 18},
		"title": &object.String{Value: "Prompt"},
		"value": &object.String{Value: "你好world"},
		"cursor": &object.Number{Value: 2},
		"meta": &object.String{Value: "meta"},
	}))
	got, ok := result.(*object.String)
	if !ok {
		t.Fatalf("input must be string, got %T: %s", result, result.Inspect())
	}
	if !strings.Contains(got.Value, "Prompt") || !strings.Contains(got.Value, "你好") || !strings.Contains(got.Value, "meta") {
		t.Fatalf("unexpected input render:\n%s", got.Value)
	}
	if !strings.Contains(got.Value, "\x1b[7m \x1b[0m") {
		t.Fatalf("input should render cursor:\n%s", got.Value)
	}
	for _, row := range strings.Split(got.Value, "\n") {
		if textVisibleWidth(row) > 18 {
			t.Fatalf("input row too wide (%d): %q", textVisibleWidth(row), row)
		}
	}
}

func TestTUIAppDispatchRenderScript(t *testing.T) {
	result := evalTUITestScript(t, `
let tui = require("@std/tui");
let app = tui.createApp({
  init: function(size) {
    return { count: 0, cols: size.cols };
  },
  update: function(state, msg) {
    if (msg.type === "text") {
      state.count = state.count + 1;
    }
    if (msg.type === "key") {
      if (msg.key === "ctrl+c") {
      return { state: state, quit: true };
      }
    }
    return state;
  },
  view: function(state, size) {
    return "count=" + String(state.count) + " cols=" + String(size.cols);
  },
});
app.dispatch(tui.text("a"));
app.dispatch(tui.text("b"));
app.render({ cols: 12, rows: 3 });
`)
	str, ok := result.(*object.String)
	if !ok {
		t.Fatalf("want string, got %T: %s", result, result.Inspect())
	}
	if str.Value != "count=2 cols=12" {
		t.Fatalf("unexpected render: %q", str.Value)
	}
}

func tuiOptions(values map[string]object.Object) *object.Hash {
	h := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for key, value := range values {
		setHashMember(h, key, value)
	}
	return h
}

func evalTUITestScript(t *testing.T, src string) object.Object {
	t.Helper()
	vm := object.NewVirtualMachine()
	env := vm.NewEnvironment()
	module.SetupExports(env)
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})
	l := lexer.New(src)
	p := parser.New(l, "tui_test.gs")
	program := p.ParseProgram()
	if len(l.Errors()) > 0 || len(program.Errors) > 0 {
		t.Fatalf("parse errors: %v %v", l.Errors(), program.Errors)
	}
	result := evaluator.Eval(program, env)
	if object.IsError(result) {
		t.Fatalf("runtime error: %s", result.Inspect())
	}
	return result
}
