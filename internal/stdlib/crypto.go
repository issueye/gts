package stdlib

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"

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
	setHashMember(exports, "sha1", &object.Builtin{Name: "crypto.sha1", Fn: cryptoSHA1})
	setHashMember(exports, "sha256", &object.Builtin{Name: "crypto.sha256", Fn: cryptoSHA256})
	setHashMember(exports, "sha512", &object.Builtin{Name: "crypto.sha512", Fn: cryptoSHA512})
	setHashMember(exports, "hmac", &object.Builtin{Name: "crypto.hmac", Fn: cryptoHMAC})
	setHashMember(exports, "pbkdf2", &object.Builtin{Name: "crypto.pbkdf2", Fn: cryptoPBKDF2})
	setHashMember(exports, "randomBytes", &object.Builtin{Name: "crypto.randomBytes", Fn: cryptoRandomBytes})
	setHashMember(exports, "timingSafeEqual", &object.Builtin{Name: "crypto.timingSafeEqual", Fn: cryptoTimingSafeEqual})
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
	return cryptoHash(pos, "crypto.sha256", sha256.New, args...)
}

func cryptoSHA1(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cryptoHash(pos, "crypto.sha1", sha1.New, args...)
}

func cryptoSHA512(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return cryptoHash(pos, "crypto.sha512", sha512.New, args...)
}

func cryptoHMAC(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	algorithm, errObj := requiredString(pos, "crypto.hmac", args, 0, "algorithm")
	if errObj != nil {
		return errObj
	}
	hashFn, errObj := cryptoHashFunc(pos, "crypto.hmac", algorithm)
	if errObj != nil {
		return errObj
	}
	key, errObj := cryptoBytesArg(pos, "crypto.hmac", args, 1, "key")
	if errObj != nil {
		return errObj
	}
	data, errObj := cryptoBytesArg(pos, "crypto.hmac", args, 2, "value")
	if errObj != nil {
		return errObj
	}
	mac := hmac.New(hashFn, key)
	if _, err := mac.Write(data); err != nil {
		return object.NewError(pos, "crypto.hmac: %v", err)
	}
	return &object.String{Value: hex.EncodeToString(mac.Sum(nil))}
}

func cryptoPBKDF2(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	password, errObj := cryptoBytesArg(pos, "crypto.pbkdf2", args, 0, "password")
	if errObj != nil {
		return errObj
	}
	salt, errObj := cryptoBytesArg(pos, "crypto.pbkdf2", args, 1, "salt")
	if errObj != nil {
		return errObj
	}
	iterations, errObj := requiredPositiveInt(pos, "crypto.pbkdf2", args, 2, "iterations")
	if errObj != nil {
		return errObj
	}
	keyLength, errObj := requiredPositiveInt(pos, "crypto.pbkdf2", args, 3, "keyLength")
	if errObj != nil {
		return errObj
	}
	algorithm := "sha256"
	if len(args) >= 5 {
		value, ok := args[4].(*object.String)
		if !ok {
			return object.NewError(pos, "crypto.pbkdf2: algorithm must be a string")
		}
		algorithm = value.Value
	}
	hashFn, errObj := cryptoHashFunc(pos, "crypto.pbkdf2", algorithm)
	if errObj != nil {
		return errObj
	}
	key, err := pbkdf2.Key(hashFn, string(password), salt, iterations, keyLength)
	if err != nil {
		return object.NewError(pos, "crypto.pbkdf2: %v", err)
	}
	if wantsBuffer(args, 5) {
		return newBufferObject(key)
	}
	return &object.String{Value: hex.EncodeToString(key)}
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

func cryptoTimingSafeEqual(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	left, errObj := cryptoBytesArg(pos, "crypto.timingSafeEqual", args, 0, "left")
	if errObj != nil {
		return errObj
	}
	right, errObj := cryptoBytesArg(pos, "crypto.timingSafeEqual", args, 1, "right")
	if errObj != nil {
		return errObj
	}
	if len(left) != len(right) {
		return object.FALSE
	}
	return object.NativeBool(subtle.ConstantTimeCompare(left, right) == 1)
}

func cryptoHash(pos ast.Position, name string, hashFn func() hash.Hash, args ...object.Object) object.Object {
	data, errObj := cryptoBytesArg(pos, name, args, 0, "value")
	if errObj != nil {
		return errObj
	}
	h := hashFn()
	if _, err := h.Write(data); err != nil {
		return object.NewError(pos, "%s: %v", name, err)
	}
	return &object.String{Value: hex.EncodeToString(h.Sum(nil))}
}

func cryptoBytesArg(pos ast.Position, name string, args []object.Object, index int, label string) ([]byte, *object.Error) {
	if len(args) <= index {
		return nil, object.NewError(pos, "%s requires %s", name, label)
	}
	return bufferBytesFromObject(pos, name, args[index], "utf8")
}

func requiredPositiveInt(pos ast.Position, name string, args []object.Object, index int, label string) (int, *object.Error) {
	if len(args) <= index {
		return 0, object.NewError(pos, "%s requires %s", name, label)
	}
	n, ok := args[index].(*object.Number)
	if !ok {
		return 0, object.NewError(pos, "%s: %s must be a number", name, label)
	}
	value := int(n.Value)
	if value <= 0 {
		return 0, object.NewError(pos, "%s: %s must be positive", name, label)
	}
	return value, nil
}

func cryptoHashFunc(pos ast.Position, name, algorithm string) (func() hash.Hash, *object.Error) {
	switch algorithm {
	case "sha1", "SHA1":
		return sha1.New, nil
	case "sha256", "SHA256":
		return sha256.New, nil
	case "sha512", "SHA512":
		return sha512.New, nil
	default:
		return nil, object.NewError(pos, "%s: unsupported hash algorithm %q", name, algorithm)
	}
}
