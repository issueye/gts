package stdlib

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

var colorEnabled = true
var colorLevel = 3

var ansiStripRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func init() {
	module.RegisterNative("@std/color", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initColorModule(exports)
		return exports, nil
	})
}

func initColorModule(exports *object.Hash) {
	setHashMember(exports, "red", createColorFunction(31))
	setHashMember(exports, "green", createColorFunction(32))
	setHashMember(exports, "yellow", createColorFunction(33))
	setHashMember(exports, "blue", createColorFunction(34))
	setHashMember(exports, "magenta", createColorFunction(35))
	setHashMember(exports, "cyan", createColorFunction(36))
	setHashMember(exports, "white", createColorFunction(37))
	setHashMember(exports, "gray", createColorFunction(90))
	setHashMember(exports, "black", createColorFunction(30))

	setHashMember(exports, "bgRed", createColorFunction(41))
	setHashMember(exports, "bgGreen", createColorFunction(42))
	setHashMember(exports, "bgYellow", createColorFunction(43))
	setHashMember(exports, "bgBlue", createColorFunction(44))
	setHashMember(exports, "bgMagenta", createColorFunction(45))
	setHashMember(exports, "bgCyan", createColorFunction(46))
	setHashMember(exports, "bgWhite", createColorFunction(47))

	setHashMember(exports, "bold", createColorFunction(1))
	setHashMember(exports, "dim", createColorFunction(2))
	setHashMember(exports, "italic", createColorFunction(3))
	setHashMember(exports, "underline", createColorFunction(4))
	setHashMember(exports, "strikethrough", createColorFunction(9))

	setHashMember(exports, "strip", &object.Builtin{Name: "color.strip", Fn: colorStrip})
	setHashMember(exports, "rgb", &object.Builtin{Name: "color.rgb", Fn: colorRgb})
	setHashMember(exports, "hex", &object.Builtin{Name: "color.hex", Fn: colorHex})

	setHashMember(exports, "enabled", &object.Boolean{Value: colorEnabled})
	setHashMember(exports, "level", &object.Number{Value: float64(colorLevel)})
}

func createColorFunction(code int) *object.Hash {
	fn := &object.Builtin{
		Name: fmt.Sprintf("color.%d", code),
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.String{Value: ""}
			}
			text := args[0].Inspect()
			if !colorEnabled {
				return &object.String{Value: text}
			}
			return &object.String{Value: fmt.Sprintf("\x1b[%dm%s\x1b[0m", code, text)}
		},
	}

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.Pairs[object.HashKeyFor(&object.String{Value: "_call"})] = object.HashPair{
		Key:   &object.String{Value: "_call"},
		Value: fn,
	}

	colors := []struct {
		name string
		code int
	}{
		{"red", 31}, {"green", 32}, {"yellow", 33}, {"blue", 34},
		{"magenta", 35}, {"cyan", 36}, {"white", 37}, {"gray", 90}, {"black", 30},
		{"bgRed", 41}, {"bgGreen", 42}, {"bgYellow", 43}, {"bgBlue", 44},
		{"bgMagenta", 45}, {"bgCyan", 46}, {"bgWhite", 47},
		{"bold", 1}, {"dim", 2}, {"italic", 3}, {"underline", 4}, {"strikethrough", 9},
	}

	for _, c := range colors {
		chainFn := createChainFunction(code, c.code)
		result.Pairs[object.HashKeyFor(&object.String{Value: c.name})] = object.HashPair{
			Key:   &object.String{Value: c.name},
			Value: chainFn,
		}
	}

	return result
}

func createChainFunction(baseCode, chainCode int) *object.Hash {
	fn := &object.Builtin{
		Name: fmt.Sprintf("color.%d.%d", baseCode, chainCode),
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.String{Value: ""}
			}
			text := args[0].Inspect()
			if !colorEnabled {
				return &object.String{Value: text}
			}
			return &object.String{Value: fmt.Sprintf("\x1b[%dm\x1b[%dm%s\x1b[0m", baseCode, chainCode, text)}
		},
	}

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.Pairs[object.HashKeyFor(&object.String{Value: "_call"})] = object.HashPair{
		Key:   &object.String{Value: "_call"},
		Value: fn,
	}
	return result
}

func colorStrip(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "color.strip", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: ansiStripRegex.ReplaceAllString(text, "")}
}

func colorRgb(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 3 {
		return object.NewError(pos, "color.rgb requires r, g, b")
	}

	r, ok1 := args[0].(*object.Number)
	g, ok2 := args[1].(*object.Number)
	b, ok3 := args[2].(*object.Number)
	if !ok1 || !ok2 || !ok3 {
		return object.NewError(pos, "color.rgb requires integer values")
	}

	fn := &object.Builtin{
		Name: "color.rgb",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.String{Value: ""}
			}
			text := args[0].Inspect()
			if !colorEnabled || colorLevel < 3 {
				return &object.String{Value: text}
			}
			return &object.String{Value: fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r.Value, g.Value, b.Value, text)}
		},
	}

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.Pairs[object.HashKeyFor(&object.String{Value: "_call"})] = object.HashPair{
		Key:   &object.String{Value: "_call"},
		Value: fn,
	}
	return result
}

func colorHex(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	hexColor, errObj := requiredString(pos, "color.hex", args, 0, "hex")
	if errObj != nil {
		return errObj
	}

	hexColor = strings.TrimPrefix(hexColor, "#")
	if len(hexColor) != 6 {
		return object.NewError(pos, "color.hex requires 6-digit hex color")
	}

	r, err1 := strconv.ParseInt(hexColor[0:2], 16, 64)
	g, err2 := strconv.ParseInt(hexColor[2:4], 16, 64)
	b, err3 := strconv.ParseInt(hexColor[4:6], 16, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return object.NewError(pos, "color.hex: invalid hex color")
	}

	fn := &object.Builtin{
		Name: "color.hex",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.String{Value: ""}
			}
			text := args[0].Inspect()
			if !colorEnabled || colorLevel < 3 {
				return &object.String{Value: text}
			}
			return &object.String{Value: fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)}
		},
	}

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	result.Pairs[object.HashKeyFor(&object.String{Value: "_call"})] = object.HashPair{
		Key:   &object.String{Value: "_call"},
		Value: fn,
	}
	return result
}
