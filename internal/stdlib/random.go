package stdlib

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/random", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initRandomModule(exports)
		return exports, nil
	})
}

func initRandomModule(exports *object.Hash) {
	setHashMember(exports, "int", &object.Builtin{Name: "random.int", Fn: randomInt})
	setHashMember(exports, "float", &object.Builtin{Name: "random.float", Fn: randomFloat})
	setHashMember(exports, "bool", &object.Builtin{Name: "random.bool", Fn: randomBool})
	setHashMember(exports, "pick", &object.Builtin{Name: "random.pick", Fn: randomPick})
	setHashMember(exports, "sample", &object.Builtin{Name: "random.sample", Fn: randomSample})
	setHashMember(exports, "shuffle", &object.Builtin{Name: "random.shuffle", Fn: randomShuffle})
	setHashMember(exports, "hex", &object.Builtin{Name: "random.hex", Fn: randomHex})
	setHashMember(exports, "base64", &object.Builtin{Name: "random.base64", Fn: randomBase64})
	setHashMember(exports, "alphanumeric", &object.Builtin{Name: "random.alphanumeric", Fn: randomAlphanumeric})
	setHashMember(exports, "alpha", &object.Builtin{Name: "random.alpha", Fn: randomAlpha})
	setHashMember(exports, "numeric", &object.Builtin{Name: "random.numeric", Fn: randomNumeric})
	setHashMember(exports, "uuid", &object.Builtin{Name: "random.uuid", Fn: randomUUID})
	setHashMember(exports, "uuidv4", &object.Builtin{Name: "random.uuidv4", Fn: randomUUID})
	setHashMember(exports, "bytes", &object.Builtin{Name: "random.bytes", Fn: randomBytes})
}

func randomInt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "random.int requires min and max")
	}
	minVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.int: min must be a number")
	}
	maxVal, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.int: max must be a number")
	}
	min := int64(minVal.Value)
	max := int64(maxVal.Value)
	if min >= max {
		return object.NewError(pos, "random.int: min must be less than max")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max-min))
	if err != nil {
		return object.NewError(pos, "random.int: %v", err)
	}
	return &object.Number{Value: float64(n.Int64() + min)}
}

func randomFloat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "random.float requires min and max")
	}
	minVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.float: min must be a number")
	}
	maxVal, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.float: max must be a number")
	}
	min := minVal.Value
	max := maxVal.Value
	if min >= max {
		return object.NewError(pos, "random.float: min must be less than max")
	}
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return object.NewError(pos, "random.float: %v", err)
	}
	val := float64(uint64(buf[0])<<56|uint64(buf[1])<<48|uint64(buf[2])<<40|uint64(buf[3])<<32|
		uint64(buf[4])<<24|uint64(buf[5])<<16|uint64(buf[6])<<8|uint64(buf[7])) / float64(^uint64(0))
	return &object.Number{Value: min + val*(max-min)}
}

func randomBool(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var buf [1]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return object.NewError(pos, "random.bool: %v", err)
	}
	return object.NativeBool(buf[0]&1 == 1)
}

func randomPick(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.pick requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "random.pick: argument must be an array")
	}
	if len(arr.Elements) == 0 {
		return object.NULL
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(arr.Elements))))
	if err != nil {
		return object.NewError(pos, "random.pick: %v", err)
	}
	return arr.Elements[n.Int64()]
}

func randomSample(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "random.sample requires array and count")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "random.sample: first argument must be an array")
	}
	countVal, ok := args[1].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.sample: count must be a number")
	}
	count := int(countVal.Value)
	if count < 0 {
		return object.NewError(pos, "random.sample: count must be non-negative")
	}
	if count > len(arr.Elements) {
		count = len(arr.Elements)
	}
	indices := make([]int, len(arr.Elements))
	for i := range indices {
		indices[i] = i
	}
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(indices)-i)))
		if err != nil {
			return object.NewError(pos, "random.sample: %v", err)
		}
		j := int(n.Int64()) + i
		indices[i], indices[j] = indices[j], indices[i]
	}
	result := make([]object.Object, count)
	for i := 0; i < count; i++ {
		result[i] = arr.Elements[indices[i]]
	}
	return &object.Array{Elements: result}
}

func randomShuffle(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.shuffle requires array")
	}
	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "random.shuffle: argument must be an array")
	}
	result := make([]object.Object, len(arr.Elements))
	copy(result, arr.Elements)
	for i := len(result) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return object.NewError(pos, "random.shuffle: %v", err)
		}
		j := int(n.Int64())
		result[i], result[j] = result[j], result[i]
	}
	return &object.Array{Elements: result}
}

func randomHex(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.hex requires byte count")
	}
	countVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.hex: count must be a number")
	}
	count := int(countVal.Value)
	if count < 0 || count > 1024 {
		return object.NewError(pos, "random.hex: count must be in range [0, 1024]")
	}
	buf := make([]byte, count)
	if _, err := rand.Read(buf); err != nil {
		return object.NewError(pos, "random.hex: %v", err)
	}
	return &object.String{Value: hex.EncodeToString(buf)}
}

func randomBase64(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.base64 requires byte count")
	}
	countVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.base64: count must be a number")
	}
	count := int(countVal.Value)
	if count < 0 || count > 1024 {
		return object.NewError(pos, "random.base64: count must be in range [0, 1024]")
	}
	buf := make([]byte, count)
	if _, err := rand.Read(buf); err != nil {
		return object.NewError(pos, "random.base64: %v", err)
	}
	return &object.String{Value: base64.StdEncoding.EncodeToString(buf)}
}

const (
	alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	alphaChars        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numericChars      = "0123456789"
)

func randomString(pos ast.Position, name string, charset string, length int) object.Object {
	if length < 0 || length > 1024 {
		return object.NewError(pos, "%s: length must be in range [0, 1024]", name)
	}
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return object.NewError(pos, "%s: %v", name, err)
		}
		result[i] = charset[n.Int64()]
	}
	return &object.String{Value: string(result)}
}

func randomAlphanumeric(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.alphanumeric requires length")
	}
	lengthVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.alphanumeric: length must be a number")
	}
	return randomString(pos, "random.alphanumeric", alphanumericChars, int(lengthVal.Value))
}

func randomAlpha(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.alpha requires length")
	}
	lengthVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.alpha: length must be a number")
	}
	return randomString(pos, "random.alpha", alphaChars, int(lengthVal.Value))
}

func randomNumeric(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.numeric requires length")
	}
	lengthVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.numeric: length must be a number")
	}
	return randomString(pos, "random.numeric", numericChars, int(lengthVal.Value))
}

func randomUUID(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return object.NewError(pos, "random.uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return &object.String{Value: fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])}
}

func randomBytes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "random.bytes requires size")
	}
	sizeVal, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "random.bytes: size must be a number")
	}
	size := int(sizeVal.Value)
	if size < 0 || size > 1024*1024 {
		return object.NewError(pos, "random.bytes: size must be in range [0, 1048576]")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return object.NewError(pos, "random.bytes: %v", err)
	}
	elements := make([]object.Object, len(buf))
	for i, b := range buf {
		elements[i] = &object.Number{Value: float64(b)}
	}
	return &object.Array{Elements: elements}
}
