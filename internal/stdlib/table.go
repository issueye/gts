package stdlib

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/table", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTableModule(exports)
		return exports, nil
	})
}

func initTableModule(exports *object.Hash) {
	setHashMember(exports, "layout", &object.Builtin{Name: "table.layout", Fn: tableLayout})
}

func tableLayout(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "table.layout requires rows")
	}
	rows, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "table.layout: rows must be an array")
	}
	opts := tableLayoutOptions{width: 80, minColWidth: 6, wrap: true}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "table.layout: options must be an object")
		}
		if value, ok := hashValue(hash, "width"); ok && value != object.UNDEFINED && value != object.NULL {
			n, ok := value.(*object.Number)
			if !ok {
				return object.NewError(pos, "table.layout: width must be a number")
			}
			opts.width = int(n.Value)
		}
		if value, ok := hashValue(hash, "minColWidth"); ok && value != object.UNDEFINED && value != object.NULL {
			n, ok := value.(*object.Number)
			if !ok {
				return object.NewError(pos, "table.layout: minColWidth must be a number")
			}
			opts.minColWidth = int(n.Value)
		}
		if value, ok := hashValue(hash, "wrap"); ok && value != object.UNDEFINED && value != object.NULL {
			b, ok := value.(*object.Boolean)
			if !ok {
				return object.NewError(pos, "table.layout: wrap must be a boolean")
			}
			opts.wrap = b.Value
		}
	}
	matrix, errObj := tableRows(rows, pos)
	if errObj != nil {
		return errObj
	}
	widths := tableColumnWidths(matrix, opts)
	lines := tableRenderLines(matrix, widths, opts)
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "lines", strSliceToArray(lines))
	setHashMember(out, "widths", tableWidthsArray(widths))
	return out
}

type tableLayoutOptions struct {
	width       int
	minColWidth int
	wrap        bool
}

func tableRows(rows *object.Array, pos ast.Position) ([][]string, *object.Error) {
	matrix := make([][]string, len(rows.Elements))
	for i, rowObj := range rows.Elements {
		row, ok := rowObj.(*object.Array)
		if !ok {
			return nil, object.NewError(pos, "table.layout: row %d must be an array", i)
		}
		matrix[i] = make([]string, len(row.Elements))
		for j, cell := range row.Elements {
			matrix[i][j] = objectToText(cell)
		}
	}
	return matrix, nil
}

func tableColumnWidths(rows [][]string, opts tableLayoutOptions) []int {
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	if cols == 0 {
		return nil
	}
	available := opts.width - (cols-1)*3
	if available < cols {
		available = cols
	}
	each := available / cols
	if each < opts.minColWidth {
		each = opts.minColWidth
	}
	widths := make([]int, cols)
	for i := range widths {
		widths[i] = each
	}
	return widths
}

func tableRenderLines(rows [][]string, widths []int, opts tableLayoutOptions) []string {
	var out []string
	for _, row := range rows {
		wrappedCells := make([][]string, len(widths))
		height := 1
		for i := range widths {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if opts.wrap {
				wrappedCells[i] = textWrapToWidth(cell, widths[i])
			} else {
				wrappedCells[i] = []string{textTruncateToWidth(cell, widths[i])}
			}
			if len(wrappedCells[i]) > height {
				height = len(wrappedCells[i])
			}
		}
		for lineIdx := 0; lineIdx < height; lineIdx++ {
			parts := make([]string, len(widths))
			for col := range widths {
				text := ""
				if lineIdx < len(wrappedCells[col]) {
					text = wrappedCells[col][lineIdx]
				}
				parts[col] = textPadRightToWidth(text, widths[col])
			}
			out = append(out, strings.Join(parts, " | "))
		}
	}
	return out
}

func textPadRightToWidth(value string, width int) string {
	out := value
	for textVisibleWidth(out) < width {
		out += " "
	}
	return out
}

func tableWidthsArray(widths []int) *object.Array {
	elements := make([]object.Object, len(widths))
	for i, width := range widths {
		elements[i] = &object.Number{Value: float64(width)}
	}
	return &object.Array{Elements: elements}
}
