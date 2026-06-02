package evaluator

import (
	"strings"
	"unicode"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var stringMethods map[string]object.BuiltinFunc

func init() {
	stringMethods = map[string]object.BuiltinFunc{
		"charAt":      builtinStrCharAt,
		"charCodeAt":  builtinStrCharCodeAt,
		"concat":      builtinStrConcat,
		"includes":    builtinStrIncludes,
		"indexOf":     builtinStrIndexOf,
		"lastIndexOf": builtinStrLastIndexOf,
		"startsWith":  builtinStrStartsWith,
		"endsWith":    builtinStrEndsWith,
		"slice":       builtinStrSlice,
		"substring":   builtinStrSubstring,
		"split":       builtinStrSplit,
		"trim":        builtinStrTrim,
		"trimStart":   builtinStrTrimStart,
		"trimEnd":     builtinStrTrimEnd,
		"toUpperCase": builtinStrToUpper,
		"toLowerCase": builtinStrToLower,
		"replace":     builtinStrReplace,
		"repeat":      builtinStrRepeat,
		"padStart":    builtinStrPadStart,
		"padEnd":      builtinStrPadEnd,
	}
}

func getStr(env *object.Environment) string {
	if s, ok := env.Extra.(*object.String); ok {
		return s.Value
	}
	return ""
}

func builtinStrCharAt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	idx := 0
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			idx = int(n.Value)
		}
	}
	if idx < 0 || idx >= len(s) {
		return &object.String{Value: ""}
	}
	return &object.String{Value: string(s[idx])}
}

func builtinStrCharCodeAt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	idx := 0
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			idx = int(n.Value)
		}
	}
	if idx < 0 || idx >= len(s) {
		return &object.Number{Value: float64(0)}
	}
	return &object.Number{Value: float64(s[idx])}
}

func builtinStrConcat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	var b strings.Builder
	b.WriteString(s)
	for _, a := range args {
		b.WriteString(a.Inspect())
	}
	return &object.String{Value: b.String()}
}

func builtinStrIncludes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	s := getStr(env)
	search := args[0].Inspect()
	from := 0
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	if from < 0 {
		from = 0
	}
	if from > len(s) {
		return object.FALSE
	}
	return object.NativeBool(strings.Contains(s[from:], search))
}

func builtinStrIndexOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Number{Value: -1}
	}
	s := getStr(env)
	search := args[0].Inspect()
	from := 0
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	if from < 0 || from >= len(s) {
		return &object.Number{Value: -1}
	}
	idx := strings.Index(s[from:], search)
	if idx < 0 {
		return &object.Number{Value: -1}
	}
	return &object.Number{Value: float64(from + idx)}
}

func builtinStrLastIndexOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Number{Value: -1}
	}
	s := getStr(env)
	search := args[0].Inspect()
	from := len(s)
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	if from < 0 {
		from = 0
	}
	if from > len(s) {
		from = len(s)
	}
	idx := strings.LastIndex(s[:from], search)
	return &object.Number{Value: float64(idx)}
}

func builtinStrStartsWith(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	s := getStr(env)
	search := args[0].Inspect()
	from := 0
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	if from < 0 || from > len(s) {
		return object.FALSE
	}
	return object.NativeBool(strings.HasPrefix(s[from:], search))
}

func builtinStrEndsWith(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	s := getStr(env)
	search := args[0].Inspect()
	end := len(s)
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			end = int(n.Value)
		}
	}
	if end < 0 || end > len(s) {
		return object.FALSE
	}
	return object.NativeBool(strings.HasSuffix(s[:end], search))
}

func builtinStrSlice(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	length := len(s)
	start := 0
	end := length
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			start = int(n.Value)
			if start < 0 {
				start = length + start
			}
			if start < 0 {
				start = 0
			}
			if start > length {
				start = length
			}
		}
	}
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			end = int(n.Value)
			if end < 0 {
				end = length + end
			}
			if end < 0 {
				end = 0
			}
			if end > length {
				end = length
			}
		}
	}
	if start >= end {
		return &object.String{Value: ""}
	}
	return &object.String{Value: s[start:end]}
}

func builtinStrSubstring(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	length := len(s)
	start := 0
	end := length
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			start = int(n.Value)
			if start < 0 {
				start = 0
			}
			if start > length {
				start = length
			}
		}
	}
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			end = int(n.Value)
			if end < 0 {
				end = 0
			}
			if end > length {
				end = length
			}
		}
	}
	if start > end {
		start, end = end, start
	}
	return &object.String{Value: s[start:end]}
}

func builtinStrSplit(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	if len(args) < 1 {
		return &object.Array{Elements: []object.Object{&object.String{Value: s}}}
	}
	sep := args[0].Inspect()
	parts := strings.Split(s, sep)
	limit := len(parts)
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			l := int(n.Value)
			if l < limit {
				limit = l
			}
		}
	}
	elements := make([]object.Object, limit)
	for i := 0; i < limit; i++ {
		elements[i] = &object.String{Value: parts[i]}
	}
	return &object.Array{Elements: elements, Pos: pos}
}

func builtinStrTrim(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: strings.TrimFunc(getStr(env), unicode.IsSpace)}
}

func builtinStrTrimStart(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: strings.TrimLeftFunc(getStr(env), unicode.IsSpace)}
}

func builtinStrTrimEnd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: strings.TrimRightFunc(getStr(env), unicode.IsSpace)}
}

func builtinStrToUpper(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: strings.ToUpper(getStr(env))}
}

func builtinStrToLower(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.String{Value: strings.ToLower(getStr(env))}
}

func builtinStrReplace(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return &object.String{Value: getStr(env)}
	}
	s := getStr(env)
	old := args[0].Inspect()
	newStr := args[1].Inspect()
	return &object.String{Value: strings.Replace(s, old, newStr, 1)}
}

func builtinStrRepeat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	count := 0
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			count = int(n.Value)
		}
	}
	if count <= 0 {
		return &object.String{Value: ""}
	}
	return &object.String{Value: strings.Repeat(s, count)}
}

func builtinStrPadStart(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	targetLen := len(s)
	padChar := " "
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			targetLen = int(n.Value)
		}
	}
	if len(args) > 1 {
		padChar = args[1].Inspect()
	}
	if len(s) >= targetLen {
		return &object.String{Value: s}
	}
	padding := strings.Repeat(padChar, (targetLen-len(s)+len(padChar)-1)/len(padChar))
	return &object.String{Value: padding[:targetLen-len(s)] + s}
}

func builtinStrPadEnd(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	s := getStr(env)
	targetLen := len(s)
	padChar := " "
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			targetLen = int(n.Value)
		}
	}
	if len(args) > 1 {
		padChar = args[1].Inspect()
	}
	if len(s) >= targetLen {
		return &object.String{Value: s}
	}
	padding := strings.Repeat(padChar, (targetLen-len(s)+len(padChar)-1)/len(padChar))
	return &object.String{Value: s + padding[:targetLen-len(s)]}
}
