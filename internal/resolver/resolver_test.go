package resolver

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/parser"
)

func resolveSource(t *testing.T, input string, opts Options) *Result {
	t.Helper()
	l := lexer.New(input)
	p := parser.New(l, "<test>")
	prog := p.ParseProgram()
	if len(l.Errors()) > 0 {
		t.Fatalf("lexer errors: %s", strings.Join(l.Errors(), "\n"))
	}
	if len(prog.Errors) > 0 {
		t.Fatalf("parse errors: %s", strings.Join(prog.Errors, "\n"))
	}
	return Resolve(prog, opts)
}

func TestResolveReportsUndefinedIdentifier(t *testing.T) {
	result := resolveSource(t, `x + 1;`, Options{})
	if len(result.Errors) != 1 {
		t.Fatalf("want 1 error, got %d: %#v", len(result.Errors), result.Errors)
	}
	if !strings.Contains(result.Errors[0].Message, `undefined identifier "x"`) {
		t.Fatalf("unexpected error: %s", result.Errors[0].Message)
	}
}

func TestResolveUsesPredeclaredBindings(t *testing.T) {
	result := resolveSource(t, `Math.abs(1);`, Options{Predeclared: []string{"Math"}})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	if len(result.References) != 1 || result.References[0].Binding == nil || result.References[0].Binding.Name != "Math" {
		t.Fatalf("Math reference was not resolved: %#v", result.References)
	}
}

func TestResolveReportsDuplicateDeclarationInSameScope(t *testing.T) {
	result := resolveSource(t, `let x = 1; const x = 2;`, Options{})
	if len(result.Errors) != 1 {
		t.Fatalf("want duplicate declaration error, got %#v", result.Errors)
	}
	if !strings.Contains(result.Errors[0].Message, `duplicate declaration of "x"`) {
		t.Fatalf("unexpected error: %s", result.Errors[0].Message)
	}
}

func TestResolveAllowsShadowingInNestedScope(t *testing.T) {
	result := resolveSource(t, `let x = 1; { let x = 2; x; } x;`, Options{})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	if len(result.References) != 2 {
		t.Fatalf("want 2 references, got %d", len(result.References))
	}
	if result.References[0].Binding == result.References[1].Binding {
		t.Fatal("nested x should resolve to shadow binding")
	}
}

func TestResolveFunctionScopeAndClosureReference(t *testing.T) {
	result := resolveSource(t, `
let outer = 1;
function f(arg) {
  let local = arg;
  return outer + local;
}
`, Options{})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	var outerRef *Reference
	for _, ref := range result.References {
		if ref.Name == "outer" {
			outerRef = ref
			break
		}
	}
	if outerRef == nil {
		t.Fatal("outer reference not found")
	}
	if outerRef.Depth < 1 {
		t.Fatalf("outer should resolve through an outer scope, got depth %d", outerRef.Depth)
	}
}

func TestResolveImportsAndExports(t *testing.T) {
	result := resolveSource(t, `
import { value as local } from "./lib.gs";
export { local as value };
`, Options{})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	if len(result.Bindings) != 1 || result.Bindings[0].Name != "local" || result.Bindings[0].Kind != BindingImport {
		t.Fatalf("import binding not recorded: %#v", result.Bindings)
	}
	if len(result.References) != 1 || result.References[0].Name != "local" {
		t.Fatalf("export reference not recorded: %#v", result.References)
	}
}

func TestResolveMemberPropertyIsNotIdentifierReference(t *testing.T) {
	result := resolveSource(t, `let obj = { value: 1 }; obj.value;`, Options{})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	for _, ref := range result.References {
		if ref.Name == "value" {
			t.Fatal("non-computed member property should not be a lexical reference")
		}
	}
}

func TestResolveMatchArmBinding(t *testing.T) {
	result := resolveSource(t, `let label = match status { 200 (val) if val > 0 => val, _ => 0 };`, Options{
		Predeclared: []string{"status"},
	})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
	var found bool
	for _, binding := range result.Bindings {
		if binding.Name == "val" && binding.Kind == BindingPattern {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("match arm binding was not recorded: %#v", result.Bindings)
	}
}
