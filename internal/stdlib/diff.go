package stdlib

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/diff", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "lines", &object.Builtin{Name: "diff.lines", Fn: diffLines})
		setHashMember(exports, "chars", &object.Builtin{Name: "diff.chars", Fn: diffChars})
		return exports, nil
	})
}

func diffLines(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "diff.lines requires two strings")
	}
	str1, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "diff.lines expects string")
	}
	str2, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "diff.lines expects string")
	}

	lines1 := strings.Split(str1.Value, "\n")
	lines2 := strings.Split(str2.Value, "\n")

	diffs := computeDiff(lines1, lines2)
	result := &object.Array{Elements: make([]object.Object, len(diffs))}
	for i, d := range diffs {
		diff := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(diff, "type", &object.String{Value: d.Type})
		setHashMember(diff, "value", &object.String{Value: d.Value})
		result.Elements[i] = diff
	}
	return result
}

func diffChars(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "diff.chars requires two strings")
	}
	str1, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "diff.chars expects string")
	}
	str2, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "diff.chars expects string")
	}

	chars1 := strings.Split(str1.Value, "")
	chars2 := strings.Split(str2.Value, "")

	diffs := computeDiff(chars1, chars2)
	result := &object.Array{Elements: make([]object.Object, len(diffs))}
	for i, d := range diffs {
		diff := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(diff, "type", &object.String{Value: d.Type})
		setHashMember(diff, "value", &object.String{Value: d.Value})
		result.Elements[i] = diff
	}
	return result
}

type diffItem struct {
	Type  string
	Value string
}

func computeDiff(a, b []string) []diffItem {
	result := []diffItem{}
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, diffItem{Type: "equal", Value: a[i]})
			i++
			j++
		} else {
			if i+1 < len(a) && a[i+1] == b[j] {
				result = append(result, diffItem{Type: "removed", Value: a[i]})
				i++
			} else if j+1 < len(b) && a[i] == b[j+1] {
				result = append(result, diffItem{Type: "added", Value: b[j]})
				j++
			} else {
				result = append(result, diffItem{Type: "removed", Value: a[i]})
				result = append(result, diffItem{Type: "added", Value: b[j]})
				i++
				j++
			}
		}
	}

	for i < len(a) {
		result = append(result, diffItem{Type: "removed", Value: a[i]})
		i++
	}
	for j < len(b) {
		result = append(result, diffItem{Type: "added", Value: b[j]})
		j++
	}

	return result
}
