package stdlib

import (
	"math/rand"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/collections", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initCollectionsModule(exports)
		return exports, nil
	})
}

func initCollectionsModule(exports *object.Hash) {
	setHashMember(exports, "unique", &object.Builtin{Name: "collections.unique", Fn: collectionUnique})
	setHashMember(exports, "chunk", &object.Builtin{Name: "collections.chunk", Fn: collectionChunk})
	setHashMember(exports, "flatten", &object.Builtin{Name: "collections.flatten", Fn: collectionFlatten})
	setHashMember(exports, "sample", &object.Builtin{Name: "collections.sample", Fn: collectionSample})
	setHashMember(exports, "shuffle", &object.Builtin{Name: "collections.shuffle", Fn: collectionShuffle})
	setHashMember(exports, "range", &object.Builtin{Name: "collections.range", Fn: collectionRange})
}

func collectionUnique(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "unique requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "unique expects array")
	}

	seen := make(map[object.HashKey]bool)
	result := &object.Array{Elements: []object.Object{}}
	for _, item := range arr.Elements {
		hk := object.HashKeyFor(item)
		if !seen[hk] {
			seen[hk] = true
			result.Elements = append(result.Elements, item)
		}
	}
	return result
}

func collectionChunk(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "chunk requires array and size")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "chunk expects array")
	}
	size, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "chunk expects number for size")
	}

	chunkSize := int(size.Value)
	if chunkSize <= 0 {
		return object.NewError(pos, "chunk size must be positive")
	}

	result := &object.Array{Elements: []object.Object{}}
	for i := 0; i < len(arr.Elements); i += chunkSize {
		end := i + chunkSize
		if end > len(arr.Elements) {
			end = len(arr.Elements)
		}
		chunk := &object.Array{Elements: arr.Elements[i:end]}
		result.Elements = append(result.Elements, chunk)
	}
	return result
}

func collectionFlatten(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "flatten requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "flatten expects array")
	}

	result := &object.Array{Elements: []object.Object{}}
	for _, item := range arr.Elements {
		if innerArr, ok := item.(*object.Array); ok {
			result.Elements = append(result.Elements, innerArr.Elements...)
		} else {
			result.Elements = append(result.Elements, item)
		}
	}
	return result
}

func collectionSample(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "sample requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "sample expects array")
	}
	if len(arr.Elements) == 0 {
		return object.UNDEFINED
	}
	return arr.Elements[rand.Intn(len(arr.Elements))]
}

func collectionShuffle(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "shuffle requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "shuffle expects array")
	}

	result := make([]object.Object, len(arr.Elements))
	copy(result, arr.Elements)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return &object.Array{Elements: result}
}

func collectionRange(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "range requires at least end value")
	}

	var start, end, step float64
	step = 1

	if len(args) == 1 {
		start = 0
		if n, ok := args[0].(*object.Number); ok {
			end = n.Value
		} else {
			return object.NewError(pos, "range expects number")
		}
	} else {
		if n, ok := args[0].(*object.Number); ok {
			start = n.Value
		} else {
			return object.NewError(pos, "range expects number")
		}
		if n, ok := args[1].(*object.Number); ok {
			end = n.Value
		} else {
			return object.NewError(pos, "range expects number")
		}
		if len(args) > 2 {
			if n, ok := args[2].(*object.Number); ok {
				step = n.Value
			}
		}
	}

	if step == 0 {
		return object.NewError(pos, "range step cannot be zero")
	}

	result := &object.Array{Elements: []object.Object{}}
	if step > 0 {
		for i := start; i < end; i += step {
			result.Elements = append(result.Elements, &object.Number{Value: i})
		}
	} else {
		for i := start; i > end; i += step {
			result.Elements = append(result.Elements, &object.Number{Value: i})
		}
	}
	return result
}
