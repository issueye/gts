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

func TestEval_ForLetInitializer(t *testing.T) {
	input := `let total = 0; for (let i = 0; i < 3; i = i + 1) { total = total + i; } total;`
	evaluated := testEval(input)
	testNumber(t, evaluated, "3")
}

func TestEval_ForInUsesIterators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "array keys",
			input:    `let out = ""; for (let k in [10, 20, 30]) { out = out + k; } out;`,
			expected: "012",
		},
		{
			name:     "string indexes",
			input:    `let total = 0; for (let k in "abc") { total = total + k; } total;`,
			expected: "3",
		},
		{
			name: "object keys",
			input: `
let seen = 0;
for (let k in { a: 1, b: 2 }) {
  if (k === "a") { seen = seen + 1; }
  if (k === "b") { seen = seen + 2; }
}
seen;
`,
			expected: "3",
		},
		{
			name: "map keys",
			input: `
let seen = 0;
let m = new Map([["a", 1], ["b", 2]]);
for (let k in m) {
  if (k === "a") { seen = seen + 1; }
  if (k === "b") { seen = seen + 2; }
}
seen;
`,
			expected: "3",
		},
		{
			name: "set keys",
			input: `
let total = 0;
let s = new Set([1, 2, 2]);
for (let k in s) { total = total + k; }
total;
`,
			expected: "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testStringOrNumber(t, evaluated, tt.expected)
		})
	}
}

func TestEval_ForOfUsesIterators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "array values",
			input:    `let total = 0; for (let v of [1, 2, 3]) { total = total + v; } total;`,
			expected: "6",
		},
		{
			name:     "string characters",
			input:    `let out = ""; for (let ch of "abc") { out = out + ch; } out;`,
			expected: "abc",
		},
		{
			name:     "object values",
			input:    `let total = 0; for (let v of { a: 1, b: 2 }) { total = total + v; } total;`,
			expected: "3",
		},
		{
			name:     "map values",
			input:    `let total = 0; let m = new Map([["a", 1], ["b", 2]]); for (let v of m) { total = total + v; } total;`,
			expected: "3",
		},
		{
			name:     "set values",
			input:    `let total = 0; let s = new Set([1, 2, 2]); for (let v of s) { total = total + v; } total;`,
			expected: "3",
		},
		{
			name: "object protocol values",
			input: `
let total = 0;
let obj = { __iterator: function() { return [4, 5, 6]; } };
for (let v of obj) { total = total + v; }
total;
`,
			expected: "15",
		},
		{
			name: "instance protocol values",
			input: `
class Bag {
  __iterator() { return [7, 8]; }
}
let total = 0;
for (let v of new Bag()) { total = total + v; }
total;
`,
			expected: "15",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluated := testEval(tt.input)
			testStringOrNumber(t, evaluated, tt.expected)
		})
	}
}

func TestEval_CustomKeyIteratorProtocol(t *testing.T) {
	input := `
let out = "";
let obj = {
  __keyIterator: function() { return ["x", "y"]; }
};
for (let k in obj) { out = out + k; }
out;
`
	evaluated := testEval(input)
	testString(t, evaluated, "xy")
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
		{`"ABC".charCodeAt(1);`, "66"},
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
		{`"hello".slice(-2);`, "lo"},
		{`"hello".substring(1, 4);`, "ell"},
		{`"a,b,c".split(",", 2).join("|");`, "a|b"},
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

func TestEval_BuiltinsDocs_GlobalConsoleMathJSON(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}

assert(print("") === undefined, "print return");
assert(println("") === undefined, "println return");
assert(console.log("docs") === undefined, "console.log return");
assert(console.info("docs") === undefined, "console.info return");
assert(console.warn("docs") === undefined, "console.warn return");
assert(console.error("docs") === undefined, "console.error return");
assert(console.debug("docs") === undefined, "console.debug return");
assert(console.assert(true, "docs") === undefined, "console.assert pass return");
assert(console.time("docs") === undefined, "console.time return");
assert(console.timeEnd("docs") === undefined, "console.timeEnd return");
assert(console.trace("docs") === undefined, "console.trace return");
assert(console.count("docs") === undefined, "console.count return");
assert(console.countReset("docs") === undefined, "console.countReset return");
assert(console.group("docs") === undefined, "console.group return");
assert(console.groupEnd() === undefined, "console.groupEnd return");
assert(console.table([{a: 1}, {a: 2, b: true}]) === undefined, "console.table return");

assert(String(12) === "12", "String number");
assert(String(true) === "true", "String boolean");
assert(Number("12.5") === 12.5, "Number string");
assert(Number(true) === 1, "Number true");
assert(Number(false) === 0, "Number false");
assert(Boolean(1) === true, "Boolean true");
assert(Boolean(0) === false, "Boolean false");
assert(parseInt("42") === 42, "parseInt string");
assert(parseInt(42.9) === 42, "parseInt number");
assert(parseFloat("3.5") === 3.5, "parseFloat string");
assert(isNaN(0 / 0) === true, "isNaN");
assert(isFinite(1) === true, "isFinite finite");
assert(isFinite(0 / 0) === false, "isFinite nan");

assert(Math.abs(-3) === 3, "Math.abs");
assert(Math.floor(3.8) === 3, "Math.floor");
assert(Math.ceil(3.2) === 4, "Math.ceil");
assert(Math.round(3.5) === 4, "Math.round");
assert(Math.max(1, 5, 2) === 5, "Math.max");
assert(Math.min(1, -5, 2) === -5, "Math.min");
assert(Math.pow(2, 3) === 8, "Math.pow");
assert(Math.sqrt(9) === 3, "Math.sqrt");
assert(Math.PI > 3, "Math.PI");
let randomValue = Math.random();
assert(randomValue >= 0, "Math.random lower");
assert(randomValue < 1, "Math.random upper");

assert(JSON.stringify([1, "a", true, null]) === "[1,\"a\",true,null]", "JSON.stringify array");
let parsed = JSON.parse("{\"a\":1,\"b\":[true,null]}");
assert(parsed.a === 1, "JSON.parse object");
assert(parsed.b[0] === true, "JSON.parse array boolean");
assert(parsed.b[1] === null, "JSON.parse array null");

"ok";
`
	evaluated := testEval(input)
	testString(t, evaluated, "ok")
}

func TestEval_BuiltinsDocs_ObjectAndArray(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}

let obj = { a: 1, b: 2 };
assert(Object.keys(obj).length === 2, "Object.keys");
assert(Object.values(obj).length === 2, "Object.values");
assert(Object.entries({ only: 7 })[0][0] === "only", "Object.entries key");
assert(Object.entries({ only: 7 })[0][1] === 7, "Object.entries value");
let assigned = Object.assign({ a: 1 }, { b: 2 }, { a: 3 });
assert(assigned.a === 3, "Object.assign override");
assert(assigned.b === 2, "Object.assign add");
assert(Object.hasOwn(assigned, "a") === true, "Object.hasOwn true");
assert(Object.hasOwn(assigned, "missing") === false, "Object.hasOwn false");

let a = [1, 2];
assert(a.length === 2, "Array.length");
assert(a.push(3) === 3, "Array.push return");
assert(a.pop() === 3, "Array.pop");
assert(a.unshift(0) === 3, "Array.unshift return");
assert(a.shift() === 0, "Array.shift");
assert(a.concat([3, 4]).join("-") === "1-2-3-4", "Array.concat/join");
assert([1, 2, 3, 4].slice(1, 3).join(",") === "2,3", "Array.slice");
let spliceTarget = [1, 2, 3, 4];
let removed = spliceTarget.splice(1, 2, 9, 10);
assert(removed.join(",") === "2,3", "Array.splice removed");
assert(spliceTarget.join(",") === "1,9,10,4", "Array.splice target");
assert([1, 2, 3, 2].indexOf(2, 2) === 3, "Array.indexOf from");
assert([1, 2, 3, 2].lastIndexOf(2) === 3, "Array.lastIndexOf");
assert([1, 2, 3].includes(2) === true, "Array.includes");
assert([1, 2, 3].find(function(x) { return x > 1; }) === 2, "Array.find");
assert([1, 2, 3].findIndex(function(x) { return x > 1; }) === 1, "Array.findIndex");
assert([1, 2, 3].filter(function(x) { return x > 1; }).join(",") === "2,3", "Array.filter");
assert([1, 2, 3].map(function(x) { return x * 2; }).join(",") === "2,4,6", "Array.map");
assert([1, 2, 3].reduce(function(acc, x) { return acc + x; }, 0) === 6, "Array.reduce");
assert([1, 2, 3].reduceRight(function(acc, x) { return acc + x; }, 0) === 6, "Array.reduceRight");
let sum = 0;
[1, 2, 3].forEach(function(x) { sum = sum + x; });
assert(sum === 6, "Array.forEach");
assert([1, 2, 3].some(function(x) { return x === 2; }) === true, "Array.some");
assert([1, 2, 3].every(function(x) { return x > 0; }) === true, "Array.every");
assert([3, 1, 2].sort(function(a, b) { return a - b; }).join(",") === "1,2,3", "Array.sort");
assert([1, 2, 3].reverse().join(",") === "3,2,1", "Array.reverse");
assert([1, [2, [3]]].flat(2).join(",") === "1,2,3", "Array.flat");
assert([1, 2].flatMap(function(x) { return [x, x + 10]; }).join(",") === "1,11,2,12", "Array.flatMap");
assert([1, 2, 3].fill(9, 1, 3).join(",") === "1,9,9", "Array.fill");
assert([1, 2, 3, 4].copyWithin(1, 2, 4).join(",") === "1,3,4,4", "Array.copyWithin");

"ok";
`
	evaluated := testEval(input)
	testString(t, evaluated, "ok")
}

func TestEval_BuiltinsDocs_PromiseStaticAll(t *testing.T) {
	input := `
Promise.all([Promise.resolve(1), 2, Promise.resolve(3)])
  .then(function(values) {
    return values.join(",");
  });
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "1,2,3")
}

func TestEval_BuiltinsDocs_MapSet(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}

let m = new Map();
assert(m.size === 0, "Map initial size");
assert(m.set("k", 1) === m, "Map.set chain");
assert(m.get("k") === 1, "Map.get");
assert(m.has("k") === true, "Map.has true");
assert(m.size === 1, "Map size after set");
m.set("k", 2);
assert(m.get("k") === 2, "Map.set overwrite");
assert(m.size === 1, "Map overwrite size");
assert(m.delete("missing") === false, "Map.delete missing");
assert(m.delete("k") === true, "Map.delete present");
assert(m.has("k") === false, "Map.has false");
assert(m.get("k") === undefined, "Map.get missing");
let fromEntries = new Map([["a", 1], ["b", 2]]);
assert(fromEntries.size === 2, "Map iterable size");
assert(fromEntries.get("b") === 2, "Map iterable get");
fromEntries.clear();
assert(fromEntries.size === 0, "Map.clear");

let s = new Set([1, 2, 2]);
assert(s.size === 2, "Set iterable dedupe");
assert(s.has(1) === true, "Set.has true");
assert(s.add(3) === s, "Set.add chain");
assert(s.size === 3, "Set.add size");
assert(s.delete(2) === true, "Set.delete present");
assert(s.delete(2) === false, "Set.delete missing");
assert(s.has(2) === false, "Set.has false");
s.clear();
assert(s.size === 0, "Set.clear");

"ok";
`
	evaluated := testEval(input)
	testString(t, evaluated, "ok")
}

func TestEval_BuiltinsDocs_PromiseRaceAndAllSettled(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`
Promise.race([Promise.resolve(4), Promise.resolve(9)])
  .then(function(value) { return value; });
`, "4"},
		{`
Promise.race([Promise.reject(new Error("fast")), Promise.resolve(9)])
  .catch(function(e) { return e.message; });
`, "fast"},
		{`
Promise.allSettled([Promise.resolve(1), Promise.reject(new TypeError("bad")), 3])
  .then(function(results) {
    return results[0].status + ":" + results[0].value.toString() + "|" +
      results[1].status + ":" + results[1].reason.name + ":" + results[1].reason.message + "|" +
      results[2].status + ":" + results[2].value.toString();
  });
`, "fulfilled:1|rejected:TypeError:bad|fulfilled:3"},
	}

	for _, tt := range tests {
		evaluated := waitIfPromise(testEval(tt.input))
		testStringOrNumber(t, evaluated, tt.expected)
	}
}

func TestEval_BuiltinsDocs_TimersAndMicrotasks(t *testing.T) {
	input := `
let events = [];
queueMicrotask(function() { events.push("micro"); });
let cancelled = setTimeout(function() { events.push("cancelled"); }, 1);
clearTimeout(cancelled);
setTimeout(function(value) { events.push(value); }, 5, "timeout");
let id = setInterval(function(label) {
  events.push(label);
  clearInterval(id);
}, 1, "interval");
sleep(20);
let score = 0;
if (events.includes("micro")) { score = score + 1; }
if (events.includes("timeout")) { score = score + 10; }
if (events.includes("interval")) { score = score + 100; }
if (events.includes("cancelled")) { score = score + 1000; }
score;
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "111")
}

func TestEval_BuiltinsDocs_ExtendedGlobalMathObjectArrayStringNumber(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}

assert(encodeURI("https://a.test/x y?q=1&v=中").includes("%20"), "encodeURI space");
assert(encodeURI("https://a.test/x y?q=1&v=中").includes("https://"), "encodeURI reserved");
assert(decodeURIComponent(encodeURIComponent("a b 中")) === "a b 中", "component roundtrip");
assert(parseInt("ff", 16) === 255, "parseInt radix");

assert(Math.E > 2, "Math.E");
assert(Math.LN2 > 0, "Math.LN2");
assert(Math.LOG2E > 1, "Math.LOG2E");
assert(Math.SQRT2 > 1, "Math.SQRT2");
assert(Math.SQRT1_2 < 1, "Math.SQRT1_2");
assert(Math.sign(-3) === -1, "Math.sign");
assert(Math.trunc(3.8) === 3, "Math.trunc");
assert(Math.cbrt(27) === 3, "Math.cbrt");
assert(Math.exp(0) === 1, "Math.exp");
assert(Math.log(Math.E) === 1, "Math.log");
assert(Math.log2(8) === 3, "Math.log2");
assert(Math.log10(100) === 2, "Math.log10");
assert(Math.sin(0) === 0, "Math.sin");
assert(Math.cos(0) === 1, "Math.cos");
assert(Math.tan(0) === 0, "Math.tan");
assert(Math.asin(0) === 0, "Math.asin");
assert(Math.acos(1) === 0, "Math.acos");
assert(Math.atan(0) === 0, "Math.atan");
assert(Math.atan2(0, 1) === 0, "Math.atan2");
assert(Math.hypot(3, 4) === 5, "Math.hypot");
assert(Math.clamp(9, 1, 5) === 5, "Math.clamp");
assert(Math.lerp(10, 20, 0.25) === 12.5, "Math.lerp");

let proto = { inherited: 7 };
let made = Object.create(proto, { own: { value: 3 } });
assert(made.inherited === 7, "Object.create proto");
assert(made.own === 3, "Object.create props");
assert(Object.getPrototypeOf(made) === proto, "Object.getPrototypeOf");
Object.setPrototypeOf(made, { inherited: 9 });
assert(made.inherited === 9, "Object.setPrototypeOf");
assert(Object.fromEntries([["a", 1], ["b", 2]]).b === 2, "Object.fromEntries");
assert(Object.is(0 / 0, 0 / 0) === true, "Object.is NaN");
let descTarget = {};
Object.defineProperty(descTarget, "x", { value: 42 });
assert(descTarget.x === 42, "Object.defineProperty");
assert(Object.getOwnPropertyDescriptor(descTarget, "x").value === 42, "Object.getOwnPropertyDescriptor");
assert(Object.getOwnPropertyNames(descTarget)[0] === "x", "Object.getOwnPropertyNames");
Object.seal(descTarget);
assert(Object.isSealed(descTarget) === true, "Object.seal");
Object.freeze(descTarget);
assert(Object.isFrozen(descTarget) === true, "Object.freeze");

assert(Array.isArray([1]) === true, "Array.isArray true");
assert(Array.isArray({}) === false, "Array.isArray false");
assert(Array.of(1, 2, 3).join(",") === "1,2,3", "Array.of");
assert(Array.from("abc").join("-") === "a-b-c", "Array.from string");
assert(Array.from([1, 2], function(x) { return x * 2; }).join(",") === "2,4", "Array.from map");

assert(String.fromCharCode(65, 66) === "AB", "String.fromCharCode");
assert(String.fromCodePoint(9731) === "☃", "String.fromCodePoint");
assert(String.raw({ raw: ["a", "\\n", "c"] }, "b") === "ab\\nc", "String.raw substitutions");
assert(String.raw({ raw: [] }) === "", "String.raw empty");
assert("abc".codePointAt(1) === 98, "String.codePointAt");
assert("a-a-a".replaceAll("-", ":") === "a:a:a", "String.replaceAll");
assert("café".normalize() === "café", "String.normalize");
assert("abc123".match("[a-z]+")[0] === "abc", "String.match");
assert("abc123".search("[0-9]+") === 3, "String.search");
assert("abc".at(-1) === "c", "String.at");
assert("abc".isWellFormed() === true, "String.isWellFormed");
assert("abc".toWellFormed() === "abc", "String.toWellFormed");

assert(Number.MAX_SAFE_INTEGER > 9000000000000000, "Number.MAX_SAFE_INTEGER");
assert(Number.MIN_SAFE_INTEGER < -9000000000000000, "Number.MIN_SAFE_INTEGER");
assert(Number.MAX_VALUE > 1e100, "Number.MAX_VALUE");
assert(Number.MIN_VALUE > 0, "Number.MIN_VALUE");
assert(Number.EPSILON > 0, "Number.EPSILON");
assert(Number.POSITIVE_INFINITY > Number.MAX_VALUE, "Number.POSITIVE_INFINITY");
assert(Number.NEGATIVE_INFINITY < -Number.MAX_VALUE, "Number.NEGATIVE_INFINITY");
assert(Number.isNaN(Number.NaN) === true, "Number.NaN/isNaN");
assert(Number.isInteger(3) === true, "Number.isInteger");
assert(Number.isFinite(3) === true, "Number.isFinite");
assert(Number.isSafeInteger(9007199254740991) === true, "Number.isSafeInteger");
assert(Number.parseFloat("1.5") === 1.5, "Number.parseFloat");
assert(Number.parseInt("10", 2) === 2, "Number.parseInt");
assert(15.toString(16) === "f", "Number.toString");
assert(1.25.toFixed(1) === "1.2", "Number.toFixed");
assert(1234.toPrecision(3) === "1.23e+03", "Number.toPrecision");
assert(12.toExponential(1) === "1.2e+01", "Number.toExponential");

"ok";
`
	evaluated := testEval(input)
	testString(t, evaluated, "ok")
}

func TestEval_BuiltinsDocs_JSONStringMatchAllBooleanWrapper(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}

let pretty = JSON.stringify({ a: 1, b: [2] }, null, 2);
assert(pretty.includes("\n  \"a\""), "JSON.stringify space");
let replaced = JSON.stringify({ keep: 1, bump: 2 }, function(k, v) {
  if (k === "bump") {
    return v + 10;
  }
  return v;
});
assert(JSON.parse(replaced).bump === 12, "JSON.stringify replacer");
let revived = JSON.parse("{\"a\":1,\"b\":[2]}", function(k, v) {
  if (k === "a") {
    return 9;
  }
  if (k === "0") {
    return v + 3;
  }
  return v;
});
assert(revived.a === 9, "JSON.parse reviver object");
assert(revived.b[0] === 5, "JSON.parse reviver array");

let matches = "a1 b2".matchAll(new RegExp("([a-z])([0-9])", "g"));
assert(matches.length === 2, "String.matchAll length");
assert(matches[0][1] === "a", "String.matchAll group");
assert("a1 b2".replaceAll(new RegExp("[0-9]", "g"), "#") === "a# b#", "String.replaceAll regexp");

let boxedTrue = new Boolean(true);
let boxedFalse = new Boolean(false);
assert(boxedTrue.valueOf() === true, "Boolean.valueOf true");
assert(boxedFalse.valueOf() === false, "Boolean.valueOf false");
assert(boxedFalse.toString() === "false", "Boolean.toString");
assert(Boolean(boxedFalse) === true, "Boolean boxed truthiness");

"ok";
`
	evaluated := testEval(input)
	testString(t, evaluated, "ok")
}

func TestEval_ConstCannotBeReassigned(t *testing.T) {
	evaluated := testEval(`const x = 1; x = 2;`)
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("want error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "assignment to constant") {
		t.Fatalf("unexpected error: %s", err.Inspect())
	}
}

func TestEval_AssignUndeclaredFails(t *testing.T) {
	evaluated := testEval(`x = 1;`)
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("want error, got %T", evaluated)
	}
	if err.Name != "ReferenceError" {
		t.Fatalf("unexpected error: %s", err.Inspect())
	}
}

func TestEval_BreakAndContinueInLoops(t *testing.T) {
	input := `
let sum = 0;
let i = 0;
while (i < 6) {
  i = i + 1;
  if (i === 1) { continue; }
  if (i === 5) { break; }
  sum = sum + i;
}
sum;
`
	evaluated := testEval(input)
	testNumber(t, evaluated, "9")
}

func TestEval_BreakOutsideLoopFails(t *testing.T) {
	evaluated := testEval(`break;`)
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("want error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "break outside loop") {
		t.Fatalf("unexpected error: %s", err.Inspect())
	}
}

func TestEval_ContinueOutsideLoopFails(t *testing.T) {
	evaluated := testEval(`continue;`)
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("want error, got %T", evaluated)
	}
	if !strings.Contains(err.Message, "continue outside loop") {
		t.Fatalf("unexpected error: %s", err.Inspect())
	}
}

func TestEval_ErrorObjectFields(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`let e = new Error("boom"); e.name;`, "Error"},
		{`let e = new Error("boom"); e.message;`, "boom"},
		{`let e = new TypeError("bad"); e.name;`, "TypeError"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testString(t, evaluated, tt.expected)
	}
}

func TestEval_ErrorObjectStack(t *testing.T) {
	evaluated := testEval(`let e = new SyntaxError("bad syntax"); e.stack;`)
	stack, ok := evaluated.(*object.String)
	if !ok {
		t.Fatalf("want stack string, got %T", evaluated)
	}
	if !strings.Contains(stack.Value, "SyntaxError: bad syntax") {
		t.Fatalf("unexpected stack: %q", stack.Value)
	}
}

func TestEval_ThrowScriptErrorCatchFields(t *testing.T) {
	input := `
try {
  throw new TypeError("bad input");
} catch (e) {
  e.name + ":" + e.message;
}
`
	evaluated := testEval(input)
	testString(t, evaluated, "TypeError:bad input")
}

func TestEval_RuntimeErrorHasNameAndStack(t *testing.T) {
	evaluated := testEval(`missingValue;`)
	err, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("want runtime error, got %T", evaluated)
	}
	if err.Name != "ReferenceError" {
		t.Fatalf("want ReferenceError, got %q", err.Name)
	}
	if !strings.Contains(err.Stack, "ReferenceError") {
		t.Fatalf("stack should include error name, got %q", err.Stack)
	}
}

func TestEval_PromiseConstructorThenCatchFinally(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`new Promise(function(resolve, reject) { resolve(2); }).then(function(x) { return x + 3; });`, "5"},
		{`Promise.reject(new TypeError("bad")).catch(function(e) { return e.name + ":" + e.message; });`, "TypeError:bad"},
		{`let done = false; Promise.resolve(1).finally(function() { done = true; }).then(function(x) { return done ? x + 1 : 0; });`, "2"},
	}

	for _, tt := range tests {
		evaluated := waitIfPromise(testEval(tt.input))
		testStringOrNumber(t, evaluated, tt.expected)
	}
}

func TestEval_AsyncRuntimeErrorRejectsPromise(t *testing.T) {
	input := `
async function fail() {
  throw new TypeError("bad async");
}
let handle = function(e) { return e.name + ":" + e.message; };
fail().catch(handle);
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "TypeError:bad async")
}

func TestEval_AwaitRejectedPromiseCanBeCaught(t *testing.T) {
	input := `
async function main() {
  try {
    await Promise.reject(new ReferenceError("missing"));
    return "no";
  } catch (e) {
    return e.name + ":" + e.message;
  }
}
main();
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "ReferenceError:missing")
}

func TestEval_PromiseThenPropagatesReturnedRejection(t *testing.T) {
	input := `
Promise.resolve(1)
  .then(function(x) { return Promise.reject(new Error("nested")); })
  .catch(function(e) { return e.message; });
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "nested")
}

func TestEval_DateBuiltinBasic(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}
let d = new Date("2020-01-02T03:04:05.006Z");
assert(d.toISOString() === "2020-01-02T03:04:05.006Z", "Date.toISOString");
assert(d.getTime() === 1577934245006, "Date.getTime");
assert(d.valueOf() === 1577934245006, "Date.valueOf");
assert(d.getUTCFullYear() === 2020, "Date.getUTCFullYear");
assert(d.getUTCMonth() === 0, "Date.getUTCMonth");
assert(d.getUTCDate() === 2, "Date.getUTCDate");
assert(d.getUTCHours() === 3, "Date.getUTCHours");
d.setTime(0);
assert(d.toISOString() === "1970-01-01T00:00:00.000Z", "Date.setTime");
assert(d.setUTCFullYear(2021, 1, 3) === 1612310400000, "Date.setUTCFullYear return");
assert(d.toISOString() === "2021-02-03T00:00:00.000Z", "Date.setUTCFullYear");
d.setUTCHours(4, 5, 6, 7);
assert(d.toISOString() === "2021-02-03T04:05:06.007Z", "Date.setUTCHours");
d.setUTCMonth(2, 4);
d.setUTCDate(5);
d.setUTCMinutes(6, 7, 8);
d.setUTCSeconds(9, 10);
d.setUTCMilliseconds(11);
assert(d.toISOString() === "2021-03-05T04:06:09.011Z", "Date UTC setters");
assert(Date.UTC(2020, 0, 2, 3, 4, 5, 6) === 1577934245006, "Date.UTC");
assert(Date.parse("2020-01-02T03:04:05.006Z") === 1577934245006, "Date.parse");
assert(new Date(0).toLocaleDateString().length > 0, "Date.toLocaleDateString");
"ok";
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "ok")
}

func TestEval_RegExpBuiltinAndStringInterop(t *testing.T) {
	input := `
function assert(cond, label) {
  if (!cond) {
    throw new Error(label);
  }
}
let re = new RegExp("a([0-9]+)", "i");
assert(re.test("xxA12yy") === true, "RegExp.test");
assert(re.exec("xxA12yy")[0] === "A12", "RegExp.exec full");
assert(re.exec("xxA12yy")[1] === "12", "RegExp.exec capture");
assert("xxA12yy".match(re)[1] === "12", "String.match RegExp");
assert("xxA12yy".search(re) === 2, "String.search RegExp");
assert("a1 a2".replace(re, "x") === "x a2", "String.replace RegExp first");
assert("a1 a2".replace(new RegExp("a[0-9]", "g"), "x") === "x x", "String.replace RegExp global");
assert(new RegExp("a+", "g").global === true, "RegExp.global");
assert(re.ignoreCase === true, "RegExp.ignoreCase");
assert(re.source === "a([0-9]+)", "RegExp.source");
assert(re.flags === "i", "RegExp.flags");
"ok";
`
	evaluated := waitIfPromise(testEval(input))
	testString(t, evaluated, "ok")
}

func waitIfPromise(obj object.Object) object.Object {
	if promise, ok := obj.(*object.Promise); ok {
		return promise.Wait()
	}
	return obj
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
