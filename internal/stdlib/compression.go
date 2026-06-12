package stdlib

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/compression", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "gzipCompress", &object.Builtin{Name: "compression.gzipCompress", Fn: compressionGzipCompress})
		setHashMember(exports, "gzipDecompress", &object.Builtin{Name: "compression.gzipDecompress", Fn: compressionGzipDecompress})
		return exports, nil
	})
}

func compressionGzipCompress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "gzipCompress requires data")
	}
	str, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "gzipCompress expects string")
	}

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write([]byte(str.Value)); err != nil {
		return object.NewError(pos, "gzipCompress: %v", err)
	}
	writer.Close()

	return &object.String{Value: buf.String()}
}

func compressionGzipDecompress(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "gzipDecompress requires data")
	}
	str, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "gzipDecompress expects string")
	}

	reader, err := gzip.NewReader(bytes.NewReader([]byte(str.Value)))
	if err != nil {
		return object.NewError(pos, "gzipDecompress: %v", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return object.NewError(pos, "gzipDecompress: %v", err)
	}

	return &object.String{Value: buf.String()}
}
