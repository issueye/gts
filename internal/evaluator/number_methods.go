package evaluator

import (
	"fmt"
	"math"
	"strconv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var numberMethods map[string]object.BuiltinFunc

func init() {
	numberMethods = map[string]object.BuiltinFunc{
		"toString":      builtinNumberToString,
		"toFixed":       builtinNumberToFixed,
		"toPrecision":   builtinNumberToPrecision,
		"toExponential": builtinNumberToExponential,
	}
}

func boundNumber(env *object.Environment) (*object.Number, bool) {
	n, ok := env.Extra.(*object.Number)
	return n, ok
}

func builtinNumberToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	num, ok := boundNumber(env)
	if !ok {
		return object.NewError(pos, "Number.toString: missing receiver")
	}
	radix := 10
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			radix = int(n.Value)
		}
	}
	if radix < 2 || radix > 36 {
		return object.NewError(pos, "RangeError: radix must be between 2 and 36")
	}
	if math.IsNaN(num.Value) || math.IsInf(num.Value, 0) || math.Trunc(num.Value) != num.Value || radix == 10 {
		return &object.String{Value: num.Inspect()}
	}
	return &object.String{Value: strconv.FormatInt(int64(num.Value), radix)}
}

func builtinNumberToFixed(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	num, ok := boundNumber(env)
	if !ok {
		return object.NewError(pos, "Number.toFixed: missing receiver")
	}
	digits := 0
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			digits = int(n.Value)
		}
	}
	if digits < 0 || digits > 100 {
		return object.NewError(pos, "RangeError: fraction digits must be between 0 and 100")
	}
	return &object.String{Value: strconv.FormatFloat(num.Value, 'f', digits, 64)}
}

func builtinNumberToPrecision(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	num, ok := boundNumber(env)
	if !ok {
		return object.NewError(pos, "Number.toPrecision: missing receiver")
	}
	if len(args) < 1 || args[0] == object.UNDEFINED {
		return &object.String{Value: num.Inspect()}
	}
	precision := 0
	if n, ok := args[0].(*object.Number); ok {
		precision = int(n.Value)
	}
	if precision < 1 || precision > 100 {
		return object.NewError(pos, "RangeError: precision must be between 1 and 100")
	}
	return &object.String{Value: strconv.FormatFloat(num.Value, 'g', precision, 64)}
}

func builtinNumberToExponential(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	num, ok := boundNumber(env)
	if !ok {
		return object.NewError(pos, "Number.toExponential: missing receiver")
	}
	digits := -1
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			digits = int(n.Value)
		}
	}
	if digits < -1 || digits > 100 {
		return object.NewError(pos, "RangeError: fraction digits must be between 0 and 100")
	}
	if digits == -1 {
		return &object.String{Value: strconv.FormatFloat(num.Value, 'e', -1, 64)}
	}
	return &object.String{Value: fmt.Sprintf("%.*e", digits, num.Value)}
}
