package gtp

import (
	"encoding/json"
	"testing"
)

func TestValueRoundTrip(t *testing.T) {
	value := Object(map[string]Value{
		"name": String("demo"),
		"n":    Number(42),
		"list": Array([]Value{Bool(true), Null()}),
	})
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Value
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if got, _ := StringField(decoded, "name"); got != "demo" {
		t.Fatalf("name = %q", got)
	}
	if got, _ := NumberField(decoded, "n"); got != 42 {
		t.Fatalf("n = %v", got)
	}
}

func TestEncodeJSONLAppendsNewline(t *testing.T) {
	data, err := EncodeJSONL(Frame{ID: "x\nok", Type: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if data[len(data)-1] != '\n' {
		t.Fatalf("encoded frame missing newline: %q", data)
	}
}
