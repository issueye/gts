package stdlib

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/compress/gzip", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initCompressGzipModule(exports)
		return exports, nil
	})
}

func initCompressGzipModule(exports *object.Hash) {
	setHashMember(exports, "compress", &object.Builtin{Name: "gzip.compress", Fn: gzipCompress})
	setHashMember(exports, "decompress", &object.Builtin{Name: "gzip.decompress", Fn: gzipDecompress})
	setHashMember(exports, "compressFileSync", &object.Builtin{Name: "gzip.compressFileSync", Fn: gzipCompressFileSync})
	setHashMember(exports, "decompressFileSync", &object.Builtin{Name: "gzip.decompressFileSync", Fn: gzipDecompressFileSync})
}

func gzipCompress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "gzip.compress requires value")
	}
	data, errObj := bufferBytesFromObject(pos, "gzip.compress", args[0], "utf8")
	if errObj != nil {
		return errObj
	}
	compressed, err := gzipCompressBytes(data)
	if err != nil {
		return object.NewError(pos, "gzip.compress: %v", err)
	}
	return newBufferObject(compressed)
}

func gzipDecompress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "gzip.decompress requires value")
	}
	data, errObj := bufferBytesFromObject(pos, "gzip.decompress", args[0], "utf8")
	if errObj != nil {
		return errObj
	}
	decompressed, err := gzipDecompressBytes(data)
	if err != nil {
		return object.NewError(pos, "gzip.decompress: %v", err)
	}
	if wantsBuffer(args, 1) {
		return newBufferObject(decompressed)
	}
	return &object.String{Value: string(decompressed)}
}

func gzipCompressFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	src, errObj := requiredString(pos, "gzip.compressFileSync", args, 0, "source path")
	if errObj != nil {
		return errObj
	}
	dst, errObj := requiredString(pos, "gzip.compressFileSync", args, 1, "destination path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return object.NewError(pos, "gzip.compressFileSync: %v", err)
	}
	compressed, err := gzipCompressBytes(data)
	if err != nil {
		return object.NewError(pos, "gzip.compressFileSync: %v", err)
	}
	if err := os.WriteFile(dst, compressed, 0644); err != nil {
		return object.NewError(pos, "gzip.compressFileSync: %v", err)
	}
	return object.UNDEFINED
}

func gzipDecompressFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	src, errObj := requiredString(pos, "gzip.decompressFileSync", args, 0, "source path")
	if errObj != nil {
		return errObj
	}
	dst, errObj := requiredString(pos, "gzip.decompressFileSync", args, 1, "destination path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return object.NewError(pos, "gzip.decompressFileSync: %v", err)
	}
	decompressed, err := gzipDecompressBytes(data)
	if err != nil {
		return object.NewError(pos, "gzip.decompressFileSync: %v", err)
	}
	if err := os.WriteFile(dst, decompressed, 0644); err != nil {
		return object.NewError(pos, "gzip.decompressFileSync: %v", err)
	}
	return object.UNDEFINED
}

func gzipCompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gzipDecompressBytes(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
