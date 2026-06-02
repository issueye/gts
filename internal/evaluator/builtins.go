package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func RegisterBuiltins(env *object.Environment) {
	RegisterBuiltinsWithCache(env, nil)
}

// RequireFn is a callback to load a module by path.
type RequireFn func(path string) (object.Object, error)

func RegisterBuiltinsWithCache(env *object.Environment, require RequireFn) {
	env.Set("console", &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hashKey(&object.String{Value: "log"}): {
				Key:   &object.String{Value: "log"},
				Value: &object.Builtin{Name: "console.log", Fn: builtinConsoleLog},
			},
		},
	})
	env.Set("println", &object.Builtin{Name: "println", Fn: builtinPrintln})
	env.Set("print", &object.Builtin{Name: "print", Fn: builtinPrint})
	env.Set("String", &object.Builtin{Name: "String", Fn: builtinString})
	env.Set("Number", &object.Builtin{Name: "Number", Fn: builtinNumber})
	env.Set("Boolean", &object.Builtin{Name: "Boolean", Fn: builtinBoolean})
	env.Set("Error", &object.Builtin{Name: "Error", Fn: builtinError})
	env.Set("TypeError", &object.Builtin{Name: "TypeError", Fn: builtinNamedError("TypeError")})
	env.Set("RangeError", &object.Builtin{Name: "RangeError", Fn: builtinNamedError("RangeError")})
	env.Set("ReferenceError", &object.Builtin{Name: "ReferenceError", Fn: builtinNamedError("ReferenceError")})
	env.Set("SyntaxError", &object.Builtin{Name: "SyntaxError", Fn: builtinNamedError("SyntaxError")})
	env.Set("parseInt", &object.Builtin{Name: "parseInt", Fn: builtinParseInt})
	env.Set("parseFloat", &object.Builtin{Name: "parseFloat", Fn: builtinParseFloat})
	env.Set("isNaN", &object.Builtin{Name: "isNaN", Fn: builtinIsNaN})
	env.Set("isFinite", &object.Builtin{Name: "isFinite", Fn: builtinIsFinite})
	env.Set("Math", &object.Hash{
		Pairs: map[object.HashKey]object.HashPair{
			hashKey(&object.String{Value: "abs"}):    {Key: &object.String{Value: "abs"}, Value: &object.Builtin{Name: "Math.abs", Fn: builtinMathAbs}},
			hashKey(&object.String{Value: "floor"}):  {Key: &object.String{Value: "floor"}, Value: &object.Builtin{Name: "Math.floor", Fn: builtinMathFloor}},
			hashKey(&object.String{Value: "ceil"}):   {Key: &object.String{Value: "ceil"}, Value: &object.Builtin{Name: "Math.ceil", Fn: builtinMathCeil}},
			hashKey(&object.String{Value: "round"}):  {Key: &object.String{Value: "round"}, Value: &object.Builtin{Name: "Math.round", Fn: builtinMathRound}},
			hashKey(&object.String{Value: "max"}):    {Key: &object.String{Value: "max"}, Value: &object.Builtin{Name: "Math.max", Fn: builtinMathMax}},
			hashKey(&object.String{Value: "min"}):    {Key: &object.String{Value: "min"}, Value: &object.Builtin{Name: "Math.min", Fn: builtinMathMin}},
			hashKey(&object.String{Value: "pow"}):    {Key: &object.String{Value: "pow"}, Value: &object.Builtin{Name: "Math.pow", Fn: builtinMathPow}},
			hashKey(&object.String{Value: "sqrt"}):   {Key: &object.String{Value: "sqrt"}, Value: &object.Builtin{Name: "Math.sqrt", Fn: builtinMathSqrt}},
			hashKey(&object.String{Value: "random"}): {Key: &object.String{Value: "random"}, Value: &object.Builtin{Name: "Math.random", Fn: builtinMathRandom}},
			hashKey(&object.String{Value: "PI"}):     {Key: &object.String{Value: "PI"}, Value: &object.Number{Value: 3.141592653589793}},
		},
	})
	registerJSON(env)
	registerObject(env)
	registerAsync(env)

	// Wire up the goroutine pool
	if Pool != nil {
		object.Spawn = Pool.Go
	} else {
		object.Spawn = Go
	}

	// Wire up the async evaluator bridge
	object.EvalPromiseFn = func(node interface{}, env *object.Environment, promise *object.Promise) {
		result := Eval(node.(ast.Node), env)
		if rv, ok := result.(*object.ReturnValue); ok {
			promise.Resolve(rv.Value)
			return
		}
		promise.Resolve(result)
	}

	// Wire up the general eval bridge for stdlib modules
	object.EvalFn = func(node interface{}, env *object.Environment) object.Object {
		return Eval(node.(ast.Node), env)
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

func builtinConsoleLog(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	for _, a := range args {
		fmt.Print(a.Inspect(), " ")
	}
	fmt.Println()
	return object.UNDEFINED
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

func builtinString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return &object.String{Value: ""}
	}
	return &object.String{Value: args[0].Inspect()}
}

func builtinNumber(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NULL
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
	}
	return &object.Number{Value: 0}
}

func builtinBoolean(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.FALSE
	}
	return object.NativeBool(object.IsTruthy(args[0]))
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
		v, _ := strconv.ParseInt(a.Value, 10, 64)
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
func builtinMathRandom(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return &object.Number{Value: math.Float64frombits(math.Float64bits(math.Pi) % 1)}
}
