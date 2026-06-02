package evaluator

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l, "<test>")
	prog := p.ParseProgram()
	if len(prog.Errors) > 0 {
		return &object.Error{Message: strings.Join(prog.Errors, "\n")}
	}
	env := object.NewEnvironment()
	RegisterBuiltins(env)
	return Eval(prog, env)
}

func TestEval_Integer(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"5;", "5"},
		{"-5;", "-5"},
		{"5 + 5;", "10"},
		{"5 - 3;", "2"},
		{"3 * 4;", "12"},
		{"12 / 3;", "4"},
		{"10 % 3;", "1"},
		{"2 ** 3;", "8"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testNumber(t, evaluated, tt.expected)
	}
}

func TestEval_String(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"hello";`, "hello"},
		{`"hello" + " world";`, "hello world"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testString(t, evaluated, tt.expected)
	}
}

func TestEval_Boolean(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true;", true},
		{"false;", false},
		{"!true;", false},
		{"!false;", true},
		{"5 === 5;", true},
		{"5 !== 3;", true},
		{"5 === 3;", false},
		{`"a" === "a";`, true},
		{`1 < 2;`, true},
		{`2 <= 2;`, true},
		{`3 > 1;`, true},
		{`3 >= 3;`, true},
		{`null === null;`, true},
		{`undefined === undefined;`, true},
		{`null === undefined;`, false},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBoolean(t, evaluated, tt.expected)
	}
}

func TestEval_LetStmt(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let a = 5; a;", "5"},
		{"let a = 5; a * 2;", "10"},
		{"let a = 5; let b = a; b;", "5"},
		{"let a = 5; let b = a; let c = a + b + 5; c;", "15"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testNumber(t, evaluated, tt.expected)
	}
}

func TestEval_Function(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"function add(a, b) { return a + b; } add(1, 2);", "3"},
		{"function double(x) { return x * 2; } double(5);", "10"},
		{"let f = function(x) { return x + 1; }; f(3);", "4"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testStringOrNumber(t, evaluated, tt.expected)
	}
}

func TestEval_Closure(t *testing.T) {
	input := `
function makeCounter() {
  let n = 0;
  return function() {
    n = n + 1;
    return n;
  };
}
let c = makeCounter();
c();
c();
c();
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "3")
}

func TestEval_IfElse(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"let f = function() { if (true) { return 10; } return 20; }; f();", "10"},
		{"let f = function() { if (false) { return 10; } else { return 20; } }; f();", "20"},
		{`let x = 5; if (x > 0) { "pos"; } else { "neg"; }`, "pos"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testStringOrNumber(t, evaluated, tt.expected)
	}
}

func TestEval_Array(t *testing.T) {
	input := `let a = [1, 2, 3]; a[0];`
	evaluated := testEval(input)
	testNumber(t, evaluated, "1")

	input2 := `let a = [1, 2, 3]; a.length;`
	evaluated2 := testEval(input2)
	testNumber(t, evaluated2, "3")
}

func TestEval_Object(t *testing.T) {
	input := `let x = { a: 1, b: 2 }; x.a;`
	evaluated := testEval(input)
	testNumber(t, evaluated, "1")
}

func TestEval_WhileLoop(t *testing.T) {
	input := `
let sum = 0;
let i = 0;
while (i < 5) {
  sum = sum + i;
  i = i + 1;
}
sum;
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "10")
}

func TestEval_ArrayPushPop(t *testing.T) {
	input := `let a = [1, 2]; a.push(3); a;`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("want length 3, got %d", len(arr.Elements))
	}
}

func TestEval_ArrayMap(t *testing.T) {
	input := `let a = [1, 2, 3]; a.map(x => x * 2);`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("want 3 elems, got %d", len(arr.Elements))
	}
}

func TestEval_ArrayFilter(t *testing.T) {
	input := `let a = [1, 2, 3, 4]; a.filter(x => x > 2);`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("want 2 elems, got %d", len(arr.Elements))
	}
}

func TestEval_ArrayReduce(t *testing.T) {
	input := `let a = [1, 2, 3]; let fn = function(acc, x) { return acc + x; }; a.reduce(fn, 0);`
	evaluated := testEval(input)
	testNumber(t, evaluated, "6")
}

func TestEval_ArrayForEach(t *testing.T) {
	input := `
let sum = 0;
let a = [1, 2, 3];
let fn = function(x) { sum = sum + x; };
a.forEach(fn);
sum;
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "6")
}

func TestEval_ArrayFind(t *testing.T) {
	input := `let a = [1, 2, 3, 4]; a.find(x => x > 2);`
	evaluated := testEval(input)
	testNumber(t, evaluated, "3")
}

func TestEval_ArrayFindIndex(t *testing.T) {
	input := `let a = [1, 2, 3, 4]; a.findIndex(x => x > 2);`
	evaluated := testEval(input)
	testNumber(t, evaluated, "2")
}

func TestEval_ArraySome(t *testing.T) {
	input := `let a = [1, 2, 3]; a.some(x => x > 2);`
	evaluated := testEval(input)
	testBoolean(t, evaluated, true)
}

func TestEval_ArrayEvery(t *testing.T) {
	input := `let a = [1, 2, 3]; a.every(x => x > 0);`
	evaluated := testEval(input)
	testBoolean(t, evaluated, true)
}

func TestEval_ArraySlice(t *testing.T) {
	input := `let a = [1, 2, 3, 4]; a.slice(1, 3);`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("want 2 elems, got %d", len(arr.Elements))
	}
}

func TestEval_ArrayJoin(t *testing.T) {
	input := `let a = [1, 2, 3]; a.join("-");`
	evaluated := testEval(input)
	testString(t, evaluated, "1-2-3")
}

func TestEval_ArrayIncludes(t *testing.T) {
	input := `let a = [1, 2, 3]; a.includes(2);`
	evaluated := testEval(input)
	testBoolean(t, evaluated, true)
}

func TestEval_ArrayIndexOf(t *testing.T) {
	input := `let a = [1, 2, 3]; a.indexOf(3);`
	evaluated := testEval(input)
	testNumber(t, evaluated, "2")
}

func TestEval_ArrayReverse(t *testing.T) {
	input := `let a = [1, 2, 3]; a.reverse(); a[0];`
	evaluated := testEval(input)
	testNumber(t, evaluated, "3")
}

func TestEval_ArrayConcat(t *testing.T) {
	input := `let a = [1, 2]; let b = [3, 4]; a.concat(b);`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 4 {
		t.Fatalf("want 4 elems, got %d", len(arr.Elements))
	}
}

func TestEval_ArrayFlat(t *testing.T) {
	input := `let a = [1, [2, 3]]; a.flat();`
	evaluated := testEval(input)
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("want Array, got %T", evaluated)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("want 3 elems, got %d", len(arr.Elements))
	}
}

func TestEval_Match(t *testing.T) {
	input := `match 1 { 1 => "one", _ => "other" };`
	evaluated := testEval(input)
	testString(t, evaluated, "one")
}

func TestEval_TypeError_PlusMixed(t *testing.T) {
	input := `1 + "1";`
	evaluated := testEval(input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("want TypeError, got %T", evaluated)
	}
}

func TestEval_TypeError_CompareMixed(t *testing.T) {
	input := `1 < "2";`
	evaluated := testEval(input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("want TypeError, got %T", evaluated)
	}
}

func TestEval_NaN(t *testing.T) {
	input := `let x = 0; 0 / 0;`
	evaluated := testEval(input)
	if n, ok := evaluated.(*object.Number); ok {
		_ = n
	} else if _, ok := evaluated.(*object.Error); ok {
		t.Logf("got error (expected): %s", evaluated.Inspect())
	}
}

func TestEval_ClassBasic(t *testing.T) {
	input := `
class Animal {
  constructor(name) { this.name = name; }
  speak() { println(this.name, "says hi"); }
}
let a = new Animal("Rex");
a.name;
`
	evaluated := testEval(input)
	testString(t, evaluated, "Rex")
}

func TestEval_ClassExtends(t *testing.T) {
	input := `
class Animal {
  constructor(name) { this.name = name; }
  greet() { return this.name; }
}
class Dog extends Animal {
  bark() { return this.name + " barks"; }
}
let d = new Dog("Rex");
d.greet() + " " + d.bark();
`
	evaluated := testEval(input)
	testString(t, evaluated, "Rex Rex barks")
}

func TestEval_ClassThisBinding(t *testing.T) {
	input := `
class Counter {
  constructor(start) { this.n = start; }
  inc() { this.n = this.n + 1; return this.n; }
  dec() { this.n = this.n - 1; return this.n; }
}
let c = new Counter(5);
c.inc(); c.inc(); c.dec();
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "6")
}

func TestEval_StringMethods(t *testing.T) {
	tests := []struct{ input, expected string }{
		{`"hello".length;`, "5"},
		{`"hello".charAt(0);`, "h"},
		{`"hello".toUpperCase();`, "HELLO"},
		{`"HELLO".toLowerCase();`, "hello"},
		{`"  a  ".trim();`, "a"},
		{`"  a  ".trimStart();`, "a  "},
		{`"  a  ".trimEnd();`, "  a"},
		{`"hello".concat(" world");`, "hello world"},
		{`"hello".includes("ell");`, "true"},
		{`"hello".includes("xyz");`, "false"},
		{`"hello".indexOf("l");`, "2"},
		{`"hello".indexOf("z");`, "-1"},
		{`"hello".lastIndexOf("l");`, "3"},
		{`"hello".startsWith("he");`, "true"},
		{`"hello".endsWith("lo");`, "true"},
		{`"hello".slice(1, 4);`, "ell"},
		{`"hello".slice(1);`, "ello"},
		{`"hello".substring(1, 4);`, "ell"},
		{`"hi".repeat(3);`, "hihihi"},
		{`"1-2".replace("-", ":");`, "1:2"},
		{`"ab".padStart(4, "x");`, "xxab"},
		{`"ab".padEnd(5, "y");`, "abyyy"},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testStringOrNumOrBool(t, evaluated, tt.expected)
	}
}

func testStringOrNumOrBool(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	if err, ok := obj.(*object.Error); ok {
		t.Fatalf("eval error: %s", err.Inspect())
	}
	switch v := obj.(type) {
	case *object.String:
		if v.Value != expected {
			t.Fatalf("want %q, got %q", expected, v.Value)
		}
	case *object.Number:
		if v.Inspect() != expected {
			t.Fatalf("want %q, got %q", expected, v.Inspect())
		}
	case *object.Boolean:
		if v.Inspect() != expected {
			t.Fatalf("want %q, got %q", expected, v.Inspect())
		}
	case *object.Array:
		if v.Inspect() != expected {
			t.Fatalf("want %q, got %q", expected, v.Inspect())
		}
	default:
		t.Fatalf("unexpected type %T: %s", obj, obj.Inspect())
	}
}

func testNumber(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	if err, ok := obj.(*object.Error); ok {
		t.Fatalf("eval error: %s", err.Inspect())
	}
	num, ok := obj.(*object.Number)
	if !ok {
		t.Fatalf("want Number, got %T (%s)", obj, obj.Inspect())
	}
	if num.Inspect() != expected {
		t.Fatalf("want %q, got %q", expected, num.Inspect())
	}
}

func testString(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	if err, ok := obj.(*object.Error); ok {
		t.Fatalf("eval error: %s", err.Inspect())
	}
	s, ok := obj.(*object.String)
	if !ok {
		t.Fatalf("want String, got %T (%s)", obj, obj.Inspect())
	}
	if s.Value != expected {
		t.Fatalf("want %q, got %q", expected, s.Value)
	}
}

func testBoolean(t *testing.T, obj object.Object, expected bool) {
	t.Helper()
	if err, ok := obj.(*object.Error); ok {
		t.Fatalf("eval error: %s", err.Inspect())
	}
	b, ok := obj.(*object.Boolean)
	if !ok {
		t.Fatalf("want Boolean, got %T (%s)", obj, obj.Inspect())
	}
	if b.Value != expected {
		t.Fatalf("want %v, got %v", expected, b.Value)
	}
}

func testStringOrNumber(t *testing.T, obj object.Object, expected string) {
	t.Helper()
	if err, ok := obj.(*object.Error); ok {
		t.Fatalf("eval error: %s", err.Inspect())
	}
	switch v := obj.(type) {
	case *object.String:
		if v.Value != expected {
			t.Fatalf("want %q, got %q", expected, v.Value)
		}
	case *object.Number:
		if v.Inspect() != expected {
			t.Fatalf("want %q, got %q", expected, v.Inspect())
		}
	default:
		t.Fatalf("want String or Number, got %T", obj)
	}
}
