package evaluator

import (
	"sort"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

var arrayMethods map[string]object.BuiltinFunc

func init() {
	arrayMethods = map[string]object.BuiltinFunc{
		"toString":    builtinNativeToString,
		"push":        builtinArrayPush,
		"pop":         builtinArrayPop,
		"shift":       builtinArrayShift,
		"unshift":     builtinArrayUnshift,
		"map":         builtinArrayMap,
		"filter":      builtinArrayFilter,
		"reduce":      builtinArrayReduce,
		"reduceRight": builtinArrayReduceRight,
		"forEach":     builtinArrayForEach,
		"find":        builtinArrayFind,
		"findIndex":   builtinArrayFindIndex,
		"some":        builtinArraySome,
		"every":       builtinArrayEvery,
		"slice":       builtinArraySlice,
		"splice":      builtinArraySplice,
		"concat":      builtinArrayConcat,
		"join":        builtinArrayJoin,
		"indexOf":     builtinArrayIndexOf,
		"lastIndexOf": builtinArrayLastIndexOf,
		"includes":    builtinArrayIncludes,
		"sort":        builtinArraySort,
		"reverse":     builtinArrayReverse,
		"fill":        builtinArrayFill,
		"flat":        builtinArrayFlat,
		"flatMap":     builtinArrayFlatMap,
		"copyWithin":  builtinArrayCopyWithin,
	}
}

// ============================================================================
// Mutating methods
// ============================================================================

func builtinArrayPush(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	arr.Elements = append(arr.Elements, args...)
	return &object.Number{Value: float64(len(arr.Elements))}
}

func builtinArrayPop(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(arr.Elements) == 0 {
		return object.UNDEFINED
	}
	last := arr.Elements[len(arr.Elements)-1]
	arr.Elements = arr.Elements[:len(arr.Elements)-1]
	return last
}

func builtinArrayShift(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(arr.Elements) == 0 {
		return object.UNDEFINED
	}
	first := arr.Elements[0]
	arr.Elements = arr.Elements[1:]
	return first
}

func builtinArrayUnshift(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	arr.Elements = append(args, arr.Elements...)
	return &object.Number{Value: float64(len(arr.Elements))}
}

func builtinArrayReverse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	for i, j := 0, len(arr.Elements)-1; i < j; i, j = i+1, j-1 {
		arr.Elements[i], arr.Elements[j] = arr.Elements[j], arr.Elements[i]
	}
	return arr
}

func builtinArraySort(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: sort requires a compare function")
	}
	cmpFn := args[0]
	sort.SliceStable(arr.Elements, func(i, j int) bool {
		result := applyFunction(cmpFn, env, []object.Object{arr.Elements[i], arr.Elements[j]}, pos)
		if num, ok := result.(*object.Number); ok {
			return num.Value < 0
		}
		return false
	})
	return arr
}

func builtinArraySplice(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	elements := arr.Elements
	length := len(elements)

	start := 0
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

	deleteCount := length - start
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			deleteCount = int(n.Value)
			if deleteCount < 0 {
				deleteCount = 0
			}
			if deleteCount > length-start {
				deleteCount = length - start
			}
		}
	}

	removed := make([]object.Object, deleteCount)
	copy(removed, elements[start:start+deleteCount])

	insertItems := args[2:]

	newElements := make([]object.Object, 0, length-deleteCount+len(insertItems))
	newElements = append(newElements, elements[:start]...)
	newElements = append(newElements, insertItems...)
	newElements = append(newElements, elements[start+deleteCount:]...)

	arr.Elements = newElements
	return &object.Array{Elements: removed, Pos: pos}
}

func builtinArrayFill(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return arr
	}
	value := args[0]
	length := len(arr.Elements)
	start := 0
	end := length
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			start = int(n.Value)
		}
	}
	if len(args) > 2 {
		if n, ok := args[2].(*object.Number); ok {
			end = int(n.Value)
		}
	}
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	for i := start; i < end; i++ {
		arr.Elements[i] = value
	}
	return arr
}

func builtinArrayCopyWithin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	length := len(arr.Elements)
	if length == 0 || len(args) < 2 {
		return arr
	}
	target := 0
	start := 0
	end := length
	if n, ok := args[0].(*object.Number); ok {
		target = int(n.Value)
	}
	if n, ok := args[1].(*object.Number); ok {
		start = int(n.Value)
	}
	if len(args) > 2 {
		if n, ok := args[2].(*object.Number); ok {
			end = int(n.Value)
		}
	}
	if target < 0 {
		target = length + target
	}
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}
	if end > length {
		end = length
	}
	copyCount := end - start
	if copyCount <= 0 {
		return arr
	}
	copy(arr.Elements[target:], arr.Elements[start:end])
	return arr
}

// ============================================================================
// Access / iterate methods
// ============================================================================

func builtinArraySlice(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	length := len(arr.Elements)
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
	if start > end {
		start = end
	}
	elements := make([]object.Object, end-start)
	copy(elements, arr.Elements[start:end])
	return &object.Array{Elements: elements, Pos: pos}
}

func builtinArrayConcat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	elements := make([]object.Object, len(arr.Elements))
	copy(elements, arr.Elements)
	for _, a := range args {
		if sub, ok := a.(*object.Array); ok {
			elements = append(elements, sub.Elements...)
		} else {
			elements = append(elements, a)
		}
	}
	return &object.Array{Elements: elements, Pos: pos}
}

func builtinArrayJoin(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	sep := ","
	if len(args) > 0 {
		if s, ok := args[0].(*object.String); ok {
			sep = s.Value
		}
	}
	return &object.String{Value: joinArrayElements(arr, sep)}
}

func builtinArrayIndexOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return &object.Number{Value: -1}
	}
	search := args[0]
	from := 0
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	elements := arr.Elements
	for i := from; i < len(elements); i++ {
		if strictEqual(elements[i], search) {
			return &object.Number{Value: float64(i)}
		}
	}
	return &object.Number{Value: -1}
}

func builtinArrayLastIndexOf(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return &object.Number{Value: -1}
	}
	search := args[0]
	elements := arr.Elements
	from := len(elements) - 1
	if len(args) > 1 {
		if n, ok := args[1].(*object.Number); ok {
			from = int(n.Value)
		}
	}
	for i := from; i >= 0; i-- {
		if strictEqual(elements[i], search) {
			return &object.Number{Value: float64(i)}
		}
	}
	return &object.Number{Value: -1}
}

func builtinArrayIncludes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.FALSE
	}
	search := args[0]
	for _, e := range arr.Elements {
		if strictEqual(e, search) {
			return object.TRUE
		}
	}
	return object.FALSE
}

// ============================================================================
// Higher-order methods
// ============================================================================

func builtinArrayMap(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: map requires a function")
	}
	callback := args[0]
	results := make([]object.Object, len(arr.Elements))
	for i, e := range arr.Elements {
		results[i] = applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
	}
	return &object.Array{Elements: results, Pos: pos}
}

func builtinArrayFilter(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: filter requires a function")
	}
	callback := args[0]
	results := make([]object.Object, 0)
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if object.IsTruthy(r) {
			results = append(results, e)
		}
	}
	return &object.Array{Elements: results, Pos: pos}
}

func builtinArrayReduce(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: reduce requires a function")
	}
	callback := args[0]
	elements := arr.Elements
	if len(elements) == 0 {
		if len(args) > 1 {
			return args[1]
		}
		return object.NewError(pos, "TypeError: reduce of empty array with no initial value")
	}
	start := 0
	var acc object.Object
	if len(args) > 1 {
		acc = args[1]
	} else {
		acc = elements[0]
		start = 1
	}
	for i := start; i < len(elements); i++ {
		acc = applyFunction(callback, env, []object.Object{acc, elements[i], &object.Number{Value: float64(i)}, arr}, pos)
	}
	return acc
}

func builtinArrayReduceRight(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: reduceRight requires a function")
	}
	callback := args[0]
	elements := arr.Elements
	if len(elements) == 0 {
		if len(args) > 1 {
			return args[1]
		}
		return object.NewError(pos, "TypeError: reduceRight of empty array with no initial value")
	}
	start := len(elements) - 1
	var acc object.Object
	if len(args) > 1 {
		acc = args[1]
	} else {
		acc = elements[start]
		start--
	}
	for i := start; i >= 0; i-- {
		acc = applyFunction(callback, env, []object.Object{acc, elements[i], &object.Number{Value: float64(i)}, arr}, pos)
	}
	return acc
}

func builtinArrayForEach(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: forEach requires a function")
	}
	callback := args[0]
	for i, e := range arr.Elements {
		applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
	}
	return object.UNDEFINED
}

func builtinArrayFind(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: find requires a function")
	}
	callback := args[0]
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if object.IsTruthy(r) {
			return e
		}
	}
	return object.UNDEFINED
}

func builtinArrayFindIndex(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: findIndex requires a function")
	}
	callback := args[0]
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if object.IsTruthy(r) {
			return &object.Number{Value: float64(i)}
		}
	}
	return &object.Number{Value: -1}
}

func builtinArraySome(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: some requires a function")
	}
	callback := args[0]
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if object.IsTruthy(r) {
			return object.TRUE
		}
	}
	return object.FALSE
}

func builtinArrayEvery(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: every requires a function")
	}
	callback := args[0]
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if !object.IsTruthy(r) {
			return object.FALSE
		}
	}
	return object.TRUE
}

func builtinArrayFlat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	depth := 1
	if len(args) > 0 {
		if n, ok := args[0].(*object.Number); ok {
			depth = int(n.Value)
		}
	}
	return &object.Array{Elements: flattenArray(arr.Elements, depth), Pos: pos}
}

func flattenArray(elements []object.Object, depth int) []object.Object {
	if depth < 1 {
		result := make([]object.Object, len(elements))
		copy(result, elements)
		return result
	}
	result := make([]object.Object, 0)
	for _, e := range elements {
		if sub, ok := e.(*object.Array); ok {
			result = append(result, flattenArray(sub.Elements, depth-1)...)
		} else {
			result = append(result, e)
		}
	}
	return result
}

func builtinArrayFlatMap(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	arr := env.Extra.(*object.Array)
	if len(args) < 1 {
		return object.NewError(pos, "TypeError: flatMap requires a function")
	}
	callback := args[0]
	results := make([]object.Object, 0)
	for i, e := range arr.Elements {
		r := applyFunction(callback, env, []object.Object{e, &object.Number{Value: float64(i)}, arr}, pos)
		if sub, ok := r.(*object.Array); ok {
			results = append(results, sub.Elements...)
		} else {
			results = append(results, r)
		}
	}
	return &object.Array{Elements: results, Pos: pos}
}
