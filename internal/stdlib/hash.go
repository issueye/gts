package stdlib

import (
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"hash/fnv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/hash", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initHashModule(exports)
		return exports, nil
	})
}

func initHashModule(exports *object.Hash) {
	setHashMember(exports, "adler32", &object.Builtin{Name: "hash.adler32", Fn: hashAdler32})
	setHashMember(exports, "crc32", &object.Builtin{Name: "hash.crc32", Fn: hashCRC32})
	setHashMember(exports, "crc64", &object.Builtin{Name: "hash.crc64", Fn: hashCRC64})
	setHashMember(exports, "fnv1a", &object.Builtin{Name: "hash.fnv1a", Fn: hashFNV1a})
	setHashMember(exports, "adler32Number", &object.Builtin{Name: "hash.adler32Number", Fn: hashAdler32Number})
	setHashMember(exports, "crc32Number", &object.Builtin{Name: "hash.crc32Number", Fn: hashCRC32Number})
}

func hashAdler32(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.adler32", args)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: fmt.Sprintf("%08x", adler32.Checksum(data))}
}

func hashCRC32(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.crc32", args)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: fmt.Sprintf("%08x", crc32.ChecksumIEEE(data))}
}

func hashCRC64(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.crc64", args)
	if errObj != nil {
		return errObj
	}
	table := crc64.MakeTable(crc64.ISO)
	return &object.String{Value: fmt.Sprintf("%016x", crc64.Checksum(data, table))}
}

func hashFNV1a(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.fnv1a", args)
	if errObj != nil {
		return errObj
	}
	h := fnv.New64a()
	if _, err := h.Write(data); err != nil {
		return object.NewError(pos, "hash.fnv1a: %v", err)
	}
	return &object.String{Value: fmt.Sprintf("%016x", h.Sum64())}
}

func hashAdler32Number(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.adler32Number", args)
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(adler32.Checksum(data))}
}

func hashCRC32Number(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	data, errObj := hashInput(pos, "hash.crc32Number", args)
	if errObj != nil {
		return errObj
	}
	return &object.Number{Value: float64(crc32.ChecksumIEEE(data))}
}

func hashInput(pos ast.Position, name string, args []object.Object) ([]byte, *object.Error) {
	if len(args) < 1 {
		return nil, object.NewError(pos, "%s requires value", name)
	}
	return bufferBytesFromObject(pos, name, args[0], "utf8")
}
