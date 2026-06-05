package stdlib

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestMarkdownParseBlocksAndDiagnostics(t *testing.T) {
	source := "# 标题\n\n- **重点**\n\n```js\nconsole.log(1)\n"
	result := markdownParse(object.NewEnvironment(), ast.Position{}, &object.String{Value: source})
	doc, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want document hash, got %T: %s", result, result.Inspect())
	}
	children := mustArray(t, doc, "children")
	if len(children.Elements) != 3 {
		t.Fatalf("want 3 blocks, got %d: %s", len(children.Elements), result.Inspect())
	}
	assertNodeType(t, children.Elements[0], "heading")
	assertNodeType(t, children.Elements[1], "list")
	assertNodeType(t, children.Elements[2], "code")
	diagnostics := mustArray(t, doc, "diagnostics")
	if len(diagnostics.Elements) < 1 {
		t.Fatalf("want unclosed fence diagnostic")
	}
}

func TestMarkdownParseInlineNodes(t *testing.T) {
	source := "A **bold** *em* `code` [link](https://example.com)"
	result := markdownParse(object.NewEnvironment(), ast.Position{}, &object.String{Value: source})
	doc := result.(*object.Hash)
	children := mustArray(t, doc, "children")
	paragraph := children.Elements[0].(*object.Hash)
	inline := mustArray(t, paragraph, "children")
	types := map[string]bool{}
	for _, item := range inline.Elements {
		hash := item.(*object.Hash)
		types[markdownNodeString(hash, "type")] = true
	}
	for _, want := range []string{"text", "strong", "em", "code", "link"} {
		if !types[want] {
			t.Fatalf("missing inline type %s in %s", want, inline.Inspect())
		}
	}
}

func TestMarkdownRenderTerminalWrapsCJK(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "width", &object.Number{Value: 4})
	result := markdownRenderTerminal(object.NewEnvironment(), ast.Position{}, &object.String{Value: "你好世界"}, opts)
	out, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want render hash, got %T: %s", result, result.Inspect())
	}
	lines := mustArray(t, out, "lines")
	if len(lines.Elements) != 2 {
		t.Fatalf("want 2 wrapped lines, got %d: %s", len(lines.Elements), lines.Inspect())
	}
	assertString(t, lines.Elements[0], "你好")
	assertString(t, lines.Elements[1], "世界")
	assertNumber(t, textWidth(object.NewEnvironment(), ast.Position{}, lines.Elements[0]), 4)
}

func TestMarkdownParseBlockquoteAndHR(t *testing.T) {
	source := "> quoted\n\n---\n"
	result := markdownParse(object.NewEnvironment(), ast.Position{}, &object.String{Value: source})
	doc := result.(*object.Hash)
	children := mustArray(t, doc, "children")
	if len(children.Elements) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(children.Elements))
	}
	assertNodeType(t, children.Elements[0], "blockquote")
	assertNodeType(t, children.Elements[1], "hr")
}

func TestMarkdownStreamSnapshotAndFinalize(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "width", &object.Number{Value: 4})
	stream := markdownCreateStream(object.NewEnvironment(), ast.Position{}, opts)
	hash, ok := stream.(*object.Hash)
	if !ok {
		t.Fatalf("want stream hash, got %T: %s", stream, stream.Inspect())
	}
	appendFn, _ := hashValue(hash, "append")
	snapshotFn, _ := hashValue(hash, "snapshot")
	finalizeFn, _ := hashValue(hash, "finalize")
	env := object.NewEnvironment()
	env.Extra, _ = hashValue(hash, "__markdownStream")
	mustNumberObject(t, appendFn.(*object.Builtin).Fn(env, ast.Position{}, &object.String{Value: "你好"}), 6)
	mustNumberObject(t, appendFn.(*object.Builtin).Fn(env, ast.Position{}, &object.String{Value: "世界"}), 12)
	snapshot := snapshotFn.(*object.Builtin).Fn(env, ast.Position{})
	lines := mustArray(t, snapshot.(*object.Hash), "lines")
	if len(lines.Elements) != 2 {
		t.Fatalf("want streaming wrap to 2 lines, got %d", len(lines.Elements))
	}
	final := finalizeFn.(*object.Builtin).Fn(env, ast.Position{})
	if _, ok := hashValue(final.(*object.Hash), "document"); !ok {
		t.Fatalf("finalize should include document")
	}
}

func TestMarkdownFromHTML(t *testing.T) {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "baseUrl", &object.String{Value: "https://example.com/docs"})
	source := `<nav>skip</nav><h1>Title</h1><p>Hello <a href="/x">link</a></p><pre><code>let x = 1</code></pre>`

	result := markdownFromHTML(object.NewEnvironment(), ast.Position{}, &object.String{Value: source}, opts)
	got, ok := result.(*object.String)
	if !ok {
		t.Fatalf("want string, got %T: %s", result, result.Inspect())
	}
	for _, want := range []string{"# Title", "[link](https://example.com/docs/x)", "```\nlet x = 1\n```"} {
		if !strings.Contains(got.Value, want) {
			t.Fatalf("want converted markdown to contain %q, got:\n%s", want, got.Value)
		}
	}
	if strings.Contains(got.Value, "skip") {
		t.Fatalf("navigation content should be dropped, got:\n%s", got.Value)
	}
}

func mustArray(t *testing.T, hash *object.Hash, key string) *object.Array {
	t.Helper()
	value, ok := hashValue(hash, key)
	if !ok {
		t.Fatalf("missing key %s in %s", key, hash.Inspect())
	}
	arr, ok := value.(*object.Array)
	if !ok {
		t.Fatalf("want array for %s, got %T: %s", key, value, value.Inspect())
	}
	return arr
}

func mustNumberObject(t *testing.T, obj object.Object, expected float64) {
	t.Helper()
	n, ok := obj.(*object.Number)
	if !ok {
		t.Fatalf("want number, got %T: %s", obj, obj.Inspect())
	}
	if n.Value != expected {
		t.Fatalf("want %v, got %v", expected, n.Value)
	}
}

func assertNodeType(t *testing.T, obj object.Object, want string) {
	t.Helper()
	hash, ok := obj.(*object.Hash)
	if !ok {
		t.Fatalf("want hash node, got %T: %s", obj, obj.Inspect())
	}
	if got := markdownNodeString(hash, "type"); got != want {
		t.Fatalf("want node type %s, got %s (%s)", want, got, markdownDebugNode(hash))
	}
}
