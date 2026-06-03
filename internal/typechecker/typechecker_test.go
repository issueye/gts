package typechecker

import (
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestMatchesPrimitiveTypes(t *testing.T) {
	tests := []struct {
		name string
		anno *ast.TypeAnnotation
		obj  object.Object
		want bool
	}{
		{"number", primitive("number"), &object.Number{Value: 1.5}, true},
		{"int ok", primitive("int"), &object.Number{Value: 2}, true},
		{"int rejects float", primitive("int"), &object.Number{Value: 2.5}, false},
		{"string", primitive("string"), &object.String{Value: "ok"}, true},
		{"boolean", primitive("boolean"), object.TRUE, true},
		{"null", primitive("null"), object.NULL, true},
		{"undefined", primitive("undefined"), object.UNDEFINED, true},
		{"any", primitive("any"), object.UNDEFINED, true},
		{"mismatch", primitive("string"), &object.Number{Value: 1}, false},
	}
	for _, tt := range tests {
		if got := Matches(tt.anno, tt.obj); got != tt.want {
			t.Fatalf("%s: want %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestMatchesOptionalUnionAndArray(t *testing.T) {
	union := &ast.TypeAnnotation{Kind: ast.TK_UNION, Union: []*ast.TypeAnnotation{primitive("number"), primitive("string")}}

	tests := []struct {
		name string
		anno *ast.TypeAnnotation
		obj  object.Object
		want bool
	}{
		{"optional undefined", &ast.TypeAnnotation{Kind: ast.TK_PRIMITIVE, Name: "string", Optional: true}, object.UNDEFINED, true},
		{"union number", union, &object.Number{Value: 1}, true},
		{"union string", union, &object.String{Value: "ok"}, true},
		{"union rejects bool", union, object.TRUE, false},
		{"array ok", &ast.TypeAnnotation{Kind: ast.TK_ARRAY, ArrayOf: primitive("number")}, &object.Array{Elements: []object.Object{&object.Number{Value: 1}}}, true},
		{"array element mismatch", &ast.TypeAnnotation{Kind: ast.TK_ARRAY, ArrayOf: primitive("number")}, &object.Array{Elements: []object.Object{&object.String{Value: "x"}}}, false},
	}
	for _, tt := range tests {
		if got := Matches(tt.anno, tt.obj); got != tt.want {
			t.Fatalf("%s: want %v, got %v", tt.name, tt.want, got)
		}
	}
}

func primitive(name string) *ast.TypeAnnotation {
	return &ast.TypeAnnotation{Kind: ast.TK_PRIMITIVE, Name: name}
}
