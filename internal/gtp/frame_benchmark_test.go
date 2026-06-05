package gtp

import (
	"bytes"
	"io"
	"testing"
)

func callFrame() Frame {
	return Frame{
		Version:    Version,
		ID:         "call-1",
		Type:       "call",
		Module:     "@go/bench",
		Method:     "sum",
		DeadlineMS: 5000,
		Args: []Value{
			Object(map[string]Value{
				"limit": Number(1000),
				"name":  String("benchmark"),
				"flags": Array([]Value{String("a"), String("b"), String("c")}),
			}),
			Array([]Value{Number(1), Number(2), Number(3), Number(4), Number(5)}),
		},
	}
}

func resultFrame() Frame {
	ok := true
	return Frame{
		Version: Version,
		ID:      "call-1",
		Type:    "result",
		OK:      &ok,
		Result: ptr(Object(map[string]Value{
			"sum":     Number(15),
			"elapsed": Number(0.25),
			"items":   Array([]Value{String("one"), String("two"), String("three")}),
		})),
	}
}

func BenchmarkEncodeCallFrame(b *testing.B) {
	frame := callFrame()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, err := EncodeFrame(frame)
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty frame")
		}
	}
}

func BenchmarkDecodeCallFrame(b *testing.B) {
	data, err := EncodeFrame(callFrame())
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		frame, err := DecodeFrame(data)
		if err != nil {
			b.Fatal(err)
		}
		if frame.ID == "" {
			b.Fatal("empty id")
		}
	}
}

func BenchmarkEncodeResultFrame(b *testing.B) {
	frame := resultFrame()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, err := EncodeFrame(frame)
		if err != nil {
			b.Fatal(err)
		}
		if len(data) == 0 {
			b.Fatal("empty frame")
		}
	}
}

func BenchmarkJSONLLoopback(b *testing.B) {
	frame := callFrame()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		dec := NewDecoder(&buf)
		if err := enc.Encode(frame); err != nil {
			b.Fatal(err)
		}
		got, err := dec.Decode()
		if err != nil && err != io.EOF {
			b.Fatal(err)
		}
		if got.ID == "" {
			b.Fatal("empty id")
		}
	}
}

func BenchmarkJSONLLoopbackReuseBuffer(b *testing.B) {
	frame := callFrame()
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := enc.Encode(frame); err != nil {
			b.Fatal(err)
		}
		line := bytes.TrimSuffix(buf.Bytes(), []byte{'\n'})
		got, err := DecodeFrame(line)
		if err != nil {
			b.Fatal(err)
		}
		if got.ID == "" {
			b.Fatal("empty id")
		}
	}
}
