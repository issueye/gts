package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func RegisterBuiltins(env *object.Environment) {
	RegisterBuiltinsWithCache(env, nil)
}

// RequireFn is a callback to load a module by path.
type RequireFn func(path string) (object.Object, error)

func RegisterBuiltinsWithCache(env *object.Environment, require RequireFn) {
	registerStandardGlobalConstants(env)

	if !env.VM().HasEvaluator() {
		env.VM().SetEvaluator(func(node interface{}, env *object.Environment) object.Object {
			return Eval(node.(ast.Node), env)
		})
	}

	if require != nil {
		env.Set("require", &object.Builtin{Name: "require", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "require requires a path string")
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return object.NewError(pos, "require requires a string path")
			}
			result, err := require(path.Value)
			if err != nil {
				return object.NewError(pos, "require: %v", err)
			}
			return result
		}})
	}
}

func registerStandardGlobalConstants(env *object.Environment) {
	if env.VM().HasGlobalConst("Math") {
		return
	}

	env.VM().SetGlobalConsts(map[string]object.Object{
		"console":            consoleObject(),
		"println":            &object.Builtin{Name: "println", Fn: builtinPrintln},
		"print":              &object.Builtin{Name: "print", Fn: builtinPrint},
		"String":             stringConstructorObject(),
		"Number":             numberConstructorObject(),
		"Boolean":            callableBuiltinObject("Boolean", builtinBoolean, map[string]object.Object{"__constructBoolean": object.TRUE}),
		"Date":               dateConstructorObject(),
		"RegExp":             callableBuiltinObject("RegExp", builtinRegExp, nil),
		"Error":              &object.Builtin{Name: "Error", Fn: builtinError},
		"TypeError":          &object.Builtin{Name: "TypeError", Fn: builtinNamedError("TypeError")},
		"RangeError":         &object.Builtin{Name: "RangeError", Fn: builtinNamedError("RangeError")},
		"ReferenceError":     &object.Builtin{Name: "ReferenceError", Fn: builtinNamedError("ReferenceError")},
		"SyntaxError":        &object.Builtin{Name: "SyntaxError", Fn: builtinNamedError("SyntaxError")},
		"parseInt":           &object.Builtin{Name: "parseInt", Fn: builtinParseInt},
		"parseFloat":         &object.Builtin{Name: "parseFloat", Fn: builtinParseFloat},
		"isNaN":              &object.Builtin{Name: "isNaN", Fn: builtinIsNaN},
		"isFinite":           &object.Builtin{Name: "isFinite", Fn: builtinIsFinite},
		"encodeURI":          &object.Builtin{Name: "encodeURI", Fn: builtinEncodeURI},
		"decodeURI":          &object.Builtin{Name: "decodeURI", Fn: builtinDecodeURI},
		"encodeURIComponent": &object.Builtin{Name: "encodeURIComponent", Fn: builtinEncodeURIComponent},
		"decodeURIComponent": &object.Builtin{Name: "decodeURIComponent", Fn: builtinDecodeURIComponent},
		"Math":               mathObject(),
		"JSON":               jsonObject(),
		"Object":             objectConstructorObject(),
		"Array":              arrayConstructorObject(),
		"Map":                callableBuiltinObject("Map", builtinMapConstructor, nil),
		"Set":                callableBuiltinObject("Set", builtinSetConstructor, nil),
		"Promise":            promiseConstructorObject(),
		"setTimeout":         &object.Builtin{Name: "setTimeout", Fn: builtinSetTimeout},
		"clearTimeout":       &object.Builtin{Name: "clearTimeout", Fn: builtinClearTimeout},
		"setInterval":        &object.Builtin{Name: "setInterval", Fn: builtinSetInterval},
		"clearInterval":      &object.Builtin{Name: "clearInterval", Fn: builtinClearInterval},
		"queueMicrotask":     &object.Builtin{Name: "queueMicrotask", Fn: builtinQueueMicrotask},
		"sleep":              &object.Builtin{Name: "sleep", Fn: builtinSleep},
	})
}

func stringConstructorObject() object.Object {
	return callableBuiltinObject("String", builtinString, map[string]object.Object{
		"fromCharCode":  &object.Builtin{Name: "String.fromCharCode", Fn: builtinStringFromCharCode},
		"fromCodePoint": &object.Builtin{Name: "String.fromCodePoint", Fn: builtinStringFromCodePoint},
		"raw":           &object.Builtin{Name: "String.raw", Fn: builtinStringRaw},
	})
}

func numberConstructorObject() object.Object {
	return callableBuiltinObject("Number", builtinNumber, map[string]object.Object{
		"MAX_SAFE_INTEGER":  &object.Number{Value: 9007199254740991},
		"MIN_SAFE_INTEGER":  &object.Number{Value: -9007199254740991},
		"MAX_VALUE":         &object.Number{Value: math.MaxFloat64},
		"MIN_VALUE":         &object.Number{Value: math.SmallestNonzeroFloat64},
		"EPSILON":           &object.Number{Value: 2.220446049250313e-16},
		"POSITIVE_INFINITY": &object.Number{Value: math.Inf(1)},
		"NEGATIVE_INFINITY": &object.Number{Value: math.Inf(-1)},
		"NaN":               &object.Number{Value: math.NaN()},
		"isInteger":         &object.Builtin{Name: "Number.isInteger", Fn: builtinNumberIsInteger},
		"isFinite":          &object.Builtin{Name: "Number.isFinite", Fn: builtinNumberIsFinite},
		"isNaN":             &object.Builtin{Name: "Number.isNaN", Fn: builtinNumberIsNaN},
		"isSafeInteger":     &object.Builtin{Name: "Number.isSafeInteger", Fn: builtinNumberIsSafeInteger},
		"parseFloat":        &object.Builtin{Name: "Number.parseFloat", Fn: builtinParseFloat},
		"parseInt":          &object.Builtin{Name: "Number.parseInt", Fn: builtinParseInt},
	})
}

func dateConstructorObject() object.Object {
	return callableBuiltinObject("Date", builtinDate, map[string]object.Object{
		"now":   &object.Builtin{Name: "Date.now", Fn: builtinDateNow},
		"parse": &object.Builtin{Name: "Date.parse", Fn: builtinDateParse},
		"UTC":   &object.Builtin{Name: "Date.UTC", Fn: builtinDateUTC},
	})
}

func mathObject() object.Object {
	return orderedHash(
		hashEntry("E", &object.Number{Value: math.E}),
		hashEntry("LN2", &object.Number{Value: math.Ln2}),
		hashEntry("LN10", &object.Number{Value: math.Ln10}),
		hashEntry("LOG2E", &object.Number{Value: 1 / math.Ln2}),
		hashEntry("LOG10E", &object.Number{Value: 1 / math.Ln10}),
		hashEntry("PI", &object.Number{Value: math.Pi}),
		hashEntry("SQRT2", &object.Number{Value: math.Sqrt2}),
		hashEntry("SQRT1_2", &object.Number{Value: math.Sqrt(0.5)}),
		hashEntry("abs", &object.Builtin{Name: "Math.abs", Fn: builtinMathAbs}),
		hashEntry("sign", &object.Builtin{Name: "Math.sign", Fn: builtinMathSign}),
		hashEntry("floor", &object.Builtin{Name: "Math.floor", Fn: builtinMathFloor}),
		hashEntry("ceil", &object.Builtin{Name: "Math.ceil", Fn: builtinMathCeil}),
		hashEntry("round", &object.Builtin{Name: "Math.round", Fn: builtinMathRound}),
		hashEntry("trunc", &object.Builtin{Name: "Math.trunc", Fn: builtinMathTrunc}),
		hashEntry("max", &object.Builtin{Name: "Math.max", Fn: builtinMathMax}),
		hashEntry("min", &object.Builtin{Name: "Math.min", Fn: builtinMathMin}),
		hashEntry("pow", &object.Builtin{Name: "Math.pow", Fn: builtinMathPow}),
		hashEntry("sqrt", &object.Builtin{Name: "Math.sqrt", Fn: builtinMathSqrt}),
		hashEntry("cbrt", &object.Builtin{Name: "Math.cbrt", Fn: builtinMathCbrt}),
		hashEntry("exp", &object.Builtin{Name: "Math.exp", Fn: builtinMathExp}),
		hashEntry("log", &object.Builtin{Name: "Math.log", Fn: builtinMathLog}),
		hashEntry("log2", &object.Builtin{Name: "Math.log2", Fn: builtinMathLog2}),
		hashEntry("log10", &object.Builtin{Name: "Math.log10", Fn: builtinMathLog10}),
		hashEntry("sin", &object.Builtin{Name: "Math.sin", Fn: builtinMathSin}),
		hashEntry("cos", &object.Builtin{Name: "Math.cos", Fn: builtinMathCos}),
		hashEntry("tan", &object.Builtin{Name: "Math.tan", Fn: builtinMathTan}),
		hashEntry("asin", &object.Builtin{Name: "Math.asin", Fn: builtinMathAsin}),
		hashEntry("acos", &object.Builtin{Name: "Math.acos", Fn: builtinMathAcos}),
		hashEntry("atan", &object.Builtin{Name: "Math.atan", Fn: builtinMathAtan}),
		hashEntry("atan2", &object.Builtin{Name: "Math.atan2", Fn: builtinMathAtan2}),
		hashEntry("random", &object.Builtin{Name: "Math.random", Fn: builtinMathRandom}),
		hashEntry("hypot", &object.Builtin{Name: "Math.hypot", Fn: builtinMathHypot}),
		hashEntry("clamp", &object.Builtin{Name: "Math.clamp", Fn: builtinMathClamp}),
		hashEntry("lerp", &object.Builtin{Name: "Math.lerp", Fn: builtinMathLerp}),
	)
}

func builtinPrintln(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	for _, a := range args {
		fmt.Print(a.Inspect())
	}
	fmt.Println()
	return object.UNDEFINED
}

func builtinPrint(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	for _, a := range args {
		fmt.Print(a.Inspect())
	}
	return object.UNDEFINED
}

func callableBuiltinObject(name string, fn object.BuiltinFunc, members map[string]object.Object) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	obj.SetMember(&object.String{Value: "__call"}, &object.Builtin{Name: name, Fn: fn})
	for key, value := range members {
		obj.SetMember(&object.String{Value: key}, value)
	}
	return obj
}

func builtinString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.String{Value: ""}
	}
	return &object.String{Value: args[0].Inspect()}
}

func builtinNumber(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Number{Value: 0}
	}
	switch a := args[0].(type) {
	case *object.Number:
		return a
	case *object.String:
		val := 0.0
		fmt.Sscanf(a.Value, "%f", &val)
		return &object.Number{Value: val}
	case *object.Boolean:
		if a.Value {
			return &object.Number{Value: 1}
		}
		return &object.Number{Value: 0}
	case *object.Null:
		return &object.Number{Value: 0}
	case *object.Undefined:
		return &object.Number{Value: math.NaN()}
	}
	return &object.Number{Value: 0}
}

func builtinBoolean(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	return object.NativeBool(object.IsTruthy(args[0]))
}

func builtinStringFromCharCode(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var b strings.Builder
	for _, arg := range args {
		num, ok := arg.(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: String.fromCharCode requires number arguments")
		}
		b.WriteRune(rune(uint16(int(num.Value))))
	}
	return &object.String{Value: b.String()}
}

func builtinStringFromCodePoint(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var b strings.Builder
	for _, arg := range args {
		num, ok := arg.(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: String.fromCodePoint requires number arguments")
		}
		cp := int(num.Value)
		if cp < 0 || cp > utf8.MaxRune {
			return object.NewError(pos, "RangeError: invalid code point %d", cp)
		}
		b.WriteRune(rune(cp))
	}
	return &object.String{Value: b.String()}
}

func builtinStringRaw(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: String.raw requires a template object")
	}
	rawObj := getHashKey(args[0], &object.String{Value: "raw"})
	raw, ok := rawObj.(*object.Array)
	if !ok {
		return object.NewError(pos, "TypeError: String.raw template object must have a raw array")
	}
	if len(raw.Elements) == 0 {
		return &object.String{Value: ""}
	}
	var b strings.Builder
	for i, part := range raw.Elements {
		b.WriteString(part.Inspect())
		if i+1 < len(raw.Elements) && i+1 < len(args) {
			b.WriteString(args[i+1].Inspect())
		}
	}
	return &object.String{Value: b.String()}
}

func builtinNumberIsInteger(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	num, ok := args[0].(*object.Number)
	return object.NativeBool(ok && !math.IsNaN(num.Value) && !math.IsInf(num.Value, 0) && math.Trunc(num.Value) == num.Value)
}

func builtinNumberIsFinite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return builtinIsFinite(env, pos, args...)
}

func builtinNumberIsNaN(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return builtinIsNaN(env, pos, args...)
}

func builtinNumberIsSafeInteger(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	num, ok := args[0].(*object.Number)
	return object.NativeBool(ok && !math.IsNaN(num.Value) && !math.IsInf(num.Value, 0) && math.Trunc(num.Value) == num.Value && math.Abs(num.Value) <= 9007199254740991)
}

func builtinEncodeURI(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return encodeURIWithMode(pos, args, false)
}

func builtinDecodeURI(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return decodeURIWithMode(pos, args, false)
}

func builtinEncodeURIComponent(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return encodeURIWithMode(pos, args, true)
}

func builtinDecodeURIComponent(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return decodeURIWithMode(pos, args, true)
}

func encodeURIWithMode(pos ast.Position, args []object.Object, component bool) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "URI encode requires a string argument")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "TypeError: URI encode requires a string argument")
	}
	return &object.String{Value: percentEncode(s.Value, component)}
}

func decodeURIWithMode(pos ast.Position, args []object.Object, component bool) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "URI decode requires a string argument")
	}
	s, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "TypeError: URI decode requires a string argument")
	}
	decoded, err := percentDecode(s.Value, component)
	if err != nil {
		return object.NewError(pos, "URI decode: %v", err)
	}
	return &object.String{Value: decoded}
}

func percentEncode(s string, component bool) string {
	var b strings.Builder
	hex := "0123456789ABCDEF"
	for _, c := range []byte(s) {
		if isURIUnescaped(c) || (!component && isURIReserved(c)) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

func percentDecode(s string, component bool) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			b.WriteByte(s[i])
			continue
		}
		if i+2 >= len(s) {
			return "", fmt.Errorf("incomplete percent escape")
		}
		code, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
		if err != nil {
			return "", err
		}
		ch := byte(code)
		if !component && isURIReserved(ch) {
			b.WriteString(s[i : i+3])
		} else {
			b.WriteByte(ch)
		}
		i += 2
	}
	return b.String(), nil
}

func isURIUnescaped(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || strings.ContainsRune("-_.!~*'()", rune(c))
}

func isURIReserved(c byte) bool {
	return strings.ContainsRune(";/?:@&=+$,#", rune(c))
}

func builtinError(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return newScriptError(pos, "Error", args...)
}

func builtinNamedError(name string) object.BuiltinFunc {
	return func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return newScriptError(pos, name, args...)
	}
}

func newScriptError(pos ast.Position, name string, args ...object.Object) object.Object {
	message := ""
	if len(args) > 0 {
		message = args[0].Inspect()
	}
	return object.NewNamedError(pos, name, message)
}

func builtinParseInt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "parseInt requires a string argument, and an explicit radix")
	}
	switch a := args[0].(type) {
	case *object.Number:
		return &object.Number{Value: float64(int64(a.Value))}
	case *object.String:
		radix := 10
		if len(args) > 1 {
			if n, ok := args[1].(*object.Number); ok {
				radix = int(n.Value)
			}
		}
		if radix < 2 || radix > 36 {
			return object.NewError(pos, "RangeError: parseInt radix must be between 2 and 36")
		}
		v, _ := strconv.ParseInt(a.Value, radix, 64)
		return &object.Number{Value: float64(v)}
	case *object.Boolean:
		if a.Value {
			return &object.Number{Value: 1}
		}
		return &object.Number{Value: 0}
	}
	return &object.Number{Value: 0}
}

func builtinParseFloat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.Number{Value: 0}
	}
	switch a := args[0].(type) {
	case *object.Number:
		return a
	case *object.String:
		v, _ := strconv.ParseFloat(a.Value, 64)
		return &object.Number{Value: v}
	}
	return &object.Number{Value: 0}
}

func builtinIsNaN(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	num, ok := args[0].(*object.Number)
	return object.NativeBool(ok && math.IsNaN(num.Value))
}

func builtinIsFinite(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	num, ok := args[0].(*object.Number)
	return object.NativeBool(ok && !math.IsInf(num.Value, 0) && !math.IsNaN(num.Value))
}

// Math functions
func requireNumber(args []object.Object, pos ast.Position, name string) (*object.Number, *object.Error) {
	if len(args) < 1 {
		return nil, object.NewError(pos, "%s requires at least 1 number argument", name)
	}
	num, ok := args[0].(*object.Number)
	if !ok {
		return nil, object.NewError(pos, "TypeError: %s requires a number argument, got %s", name, args[0].Type())
	}
	return num, nil
}

func builtinMathAbs(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.abs")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Abs(n.Value)}
}
func builtinMathSign(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.sign")
	if e != nil {
		return e
	}
	if n.Value > 0 {
		return &object.Number{Value: 1}
	}
	if n.Value < 0 {
		return &object.Number{Value: -1}
	}
	return &object.Number{Value: 0}
}
func builtinMathFloor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.floor")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Floor(n.Value)}
}
func builtinMathCeil(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.ceil")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Ceil(n.Value)}
}
func builtinMathRound(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.round")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Round(n.Value)}
}
func builtinMathTrunc(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.trunc")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Trunc(n.Value)}
}
func builtinMathMax(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "Math.max requires at least 2 arguments")
	}
	max := -math.MaxFloat64
	for _, a := range args {
		n, ok := a.(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: Math.max requires number arguments")
		}
		if n.Value > max {
			max = n.Value
		}
	}
	return &object.Number{Value: max}
}
func builtinMathMin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "Math.min requires at least 2 arguments")
	}
	min := math.MaxFloat64
	for _, a := range args {
		n, ok := a.(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: Math.min requires number arguments")
		}
		if n.Value < min {
			min = n.Value
		}
	}
	return &object.Number{Value: min}
}
func builtinMathPow(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "Math.pow requires 2 arguments")
	}
	a, ok1 := args[0].(*object.Number)
	b, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return object.NewError(pos, "TypeError: Math.pow requires number arguments")
	}
	return &object.Number{Value: math.Pow(a.Value, b.Value)}
}
func builtinMathSqrt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.sqrt")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Sqrt(n.Value)}
}
func builtinMathCbrt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.cbrt")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Cbrt(n.Value)}
}
func builtinMathExp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.exp")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Exp(n.Value)}
}
func builtinMathLog(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.log")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Log(n.Value)}
}
func builtinMathLog2(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.log2")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Log2(n.Value)}
}
func builtinMathLog10(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.log10")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Log10(n.Value)}
}
func builtinMathSin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.sin")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Sin(n.Value)}
}
func builtinMathCos(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.cos")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Cos(n.Value)}
}
func builtinMathTan(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.tan")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Tan(n.Value)}
}
func builtinMathAsin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.asin")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Asin(n.Value)}
}
func builtinMathAcos(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.acos")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Acos(n.Value)}
}
func builtinMathAtan(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	n, e := requireNumber(args, pos, "Math.atan")
	if e != nil {
		return e
	}
	return &object.Number{Value: math.Atan(n.Value)}
}
func builtinMathAtan2(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "Math.atan2 requires 2 arguments")
	}
	y, ok1 := args[0].(*object.Number)
	x, ok2 := args[1].(*object.Number)
	if !ok1 || !ok2 {
		return object.NewError(pos, "TypeError: Math.atan2 requires number arguments")
	}
	return &object.Number{Value: math.Atan2(y.Value, x.Value)}
}
func builtinMathRandom(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: math.Float64frombits(math.Float64bits(math.Pi) % 1)}
}
func builtinMathHypot(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	sum := 0.0
	for _, arg := range args {
		n, ok := arg.(*object.Number)
		if !ok {
			return object.NewError(pos, "TypeError: Math.hypot requires number arguments")
		}
		sum += n.Value * n.Value
	}
	return &object.Number{Value: math.Sqrt(sum)}
}
func builtinMathClamp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 3 {
		return object.NewError(pos, "Math.clamp requires 3 arguments")
	}
	x, ok1 := args[0].(*object.Number)
	lo, ok2 := args[1].(*object.Number)
	hi, ok3 := args[2].(*object.Number)
	if !ok1 || !ok2 || !ok3 {
		return object.NewError(pos, "TypeError: Math.clamp requires number arguments")
	}
	return &object.Number{Value: math.Min(math.Max(x.Value, lo.Value), hi.Value)}
}
func builtinMathLerp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 3 {
		return object.NewError(pos, "Math.lerp requires 3 arguments")
	}
	a, ok1 := args[0].(*object.Number)
	b, ok2 := args[1].(*object.Number)
	t, ok3 := args[2].(*object.Number)
	if !ok1 || !ok2 || !ok3 {
		return object.NewError(pos, "TypeError: Math.lerp requires number arguments")
	}
	return &object.Number{Value: a.Value + (b.Value-a.Value)*t.Value}
}
