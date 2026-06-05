package stdlib

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestTableLayoutWrapsRows(t *testing.T) {
	rows := &object.Array{Elements: []object.Object{
		&object.Array{Elements: []object.Object{
			&object.String{Value: "Name"},
			&object.String{Value: "Desc"},
		}},
		&object.Array{Elements: []object.Object{
			&object.String{Value: "你好"},
			&object.String{Value: "long text"},
		}},
	}}
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "width", &object.Number{Value: 16})
	setHashMember(opts, "minColWidth", &object.Number{Value: 4})

	result := tableLayout(object.NewEnvironment(), ast.Position{}, rows, opts)
	out, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want table layout hash, got %T: %s", result, result.Inspect())
	}
	lines := mustArray(t, out, "lines")
	widths := mustArray(t, out, "widths")
	if len(widths.Elements) != 2 {
		t.Fatalf("want 2 widths, got %d", len(widths.Elements))
	}
	if len(lines.Elements) < 3 {
		t.Fatalf("want wrapped table lines, got %d: %s", len(lines.Elements), lines.Inspect())
	}
	firstLine, ok := lines.Elements[0].(*object.String)
	if !ok {
		t.Fatalf("want first line string, got %T", lines.Elements[0])
	}
	if !strings.Contains(firstLine.Value, "Name") || !strings.Contains(firstLine.Value, "Desc") {
		t.Fatalf("want first line to contain headers, got %q", firstLine.Value)
	}
	if gotWidth := textWidth(object.NewEnvironment(), ast.Position{}, firstLine).(*object.Number).Value; gotWidth > 16 {
		t.Fatalf("want first line width <= 16, got %v", gotWidth)
	}
}
