package gtp

import (
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	ok := true
	frame := Frame{
		Version:    Version,
		ID:         "42",
		Type:       "result",
		OK:         &ok,
		Result:     ptr(Object(map[string]Value{"answer": Number(42)})),
		DeadlineMS: 1000,
	}
	data, err := EncodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "42" || got.Type != "result" || got.Result == nil {
		t.Fatalf("decoded frame = %#v", got)
	}
	answer := got.Result.Fields["answer"]
	if answer.Type != "number" || answer.Value.(float64) != 42 {
		t.Fatalf("answer = %#v", answer)
	}
}

func TestJSONLRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	dec := NewDecoder(&buf)
	if err := enc.Encode(callFrame()); err != nil {
		t.Fatal(err)
	}
	got, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "call-1" || len(got.Args) != 2 {
		t.Fatalf("decoded frame = %#v", got)
	}
}

func ptr[T any](v T) *T { return &v }
