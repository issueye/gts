package stdlib

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/crypto", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initCryptoModule(exports)
		return exports, nil
	})
}

func initCryptoModule(exports *object.Hash) {
	setHashMember(exports, "randomUUID", &object.Builtin{Name: "crypto.randomUUID", Fn: cryptoRandomUUID})
	setHashMember(exports, "sha256", &object.Builtin{Name: "crypto.sha256", Fn: cryptoSHA256})
	setHashMember(exports, "randomBytes", &object.Builtin{Name: "crypto.randomBytes", Fn: cryptoRandomBytes})
}

func cryptoRandomUUID(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return object.NewError(pos, "crypto.randomUUID: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return &object.String{Value: fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])}
}

func cryptoSHA256(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "crypto.sha256", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	sum := sha256.Sum256([]byte(value))
	return &object.String{Value: hex.EncodeToString(sum[:])}
}

func cryptoRandomBytes(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "crypto.randomBytes requires size")
	}
	sizeArg, ok := args[0].(*object.Number)
	if !ok {
		return object.NewError(pos, "crypto.randomBytes: size must be a number")
	}
	size := int(sizeArg.Value)
	if size < 0 {
		return object.NewError(pos, "crypto.randomBytes: size must be non-negative")
	}
	if size > 1024*1024 {
		return object.NewError(pos, "crypto.randomBytes: size must be <= 1048576")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return object.NewError(pos, "crypto.randomBytes: %v", err)
	}
	elements := make([]object.Object, len(buf))
	for i, b := range buf {
		elements[i] = &object.Number{Value: float64(b)}
	}
	return &object.Array{Elements: elements}
}
