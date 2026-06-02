package object

import "testing"

func TestTypeHelpersHandleNil(t *testing.T) {
	if IsNumber(nil) {
		t.Fatal("nil should not be a number")
	}
	if IsString(nil) {
		t.Fatal("nil should not be a string")
	}
	if IsError(nil) {
		t.Fatal("nil should not be an error")
	}
}

func TestTypeHelpers(t *testing.T) {
	if !IsNumber(&Number{Value: 1}) {
		t.Fatal("number helper returned false")
	}
	if !IsString(&String{Value: "x"}) {
		t.Fatal("string helper returned false")
	}
	if !IsError(&Error{Message: "boom"}) {
		t.Fatal("error helper returned false")
	}
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		name string
		obj  Object
		want bool
	}{
		{"null", NULL, false},
		{"undefined", UNDEFINED, false},
		{"false", FALSE, false},
		{"true", TRUE, true},
		{"zero", &Number{Value: 0}, false},
		{"number", &Number{Value: 1}, true},
		{"empty string", &String{Value: ""}, false},
		{"string", &String{Value: "x"}, true},
		{"object", &Hash{Pairs: map[HashKey]HashPair{}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTruthy(tt.obj); got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
