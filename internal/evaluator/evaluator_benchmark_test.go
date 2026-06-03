package evaluator

import (
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

func benchmarkProgram(b *testing.B, input string) *ast.Program {
	b.Helper()
	l := lexer.New(input)
	p := parser.New(l, "<bench>")
	prog := p.ParseProgram()
	if len(prog.Errors) > 0 {
		b.Fatalf("parse errors: %s", strings.Join(prog.Errors, "\n"))
	}
	return prog
}

func benchmarkEvalParsed(b *testing.B, input string) object.Object {
	b.Helper()
	env := object.NewEnvironment()
	RegisterBuiltins(env)
	prog := benchmarkProgram(b, input)
	result := Eval(prog, env)
	if object.IsRuntimeError(result) {
		b.Fatalf("eval failed: %s", result.Inspect())
	}
	return result
}

func BenchmarkEvalArithmeticLoop(b *testing.B) {
	prog := benchmarkProgram(b, `
let total = 0;
for (let i = 0; i < 200; i = i + 1) {
  total = total + i;
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalGlobalBuiltinLoop(b *testing.B) {
	prog := benchmarkProgram(b, `
let total = 0;
for (let i = 0; i < 200; i = i + 1) {
  total = total + Math.abs(-i);
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkParseAndEvalArithmeticLoop(b *testing.B) {
	src := `
let total = 0;
for (let i = 0; i < 200; i = i + 1) {
  total = total + i;
}
total;
`

	b.ReportAllocs()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		l := lexer.New(src)
		p := parser.New(l, "<bench>")
		prog := p.ParseProgram()
		if len(prog.Errors) > 0 {
			b.Fatalf("parse errors: %s", strings.Join(prog.Errors, "\n"))
		}
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalRecursiveFunction(b *testing.B) {
	prog := benchmarkProgram(b, `
function fib(n) {
  if (n <= 1) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}
fib(12);
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalArrayObjectPipeline(b *testing.B) {
	prog := benchmarkProgram(b, `
let rows = [];
for (let i = 0; i < 80; i = i + 1) {
  rows.push({ id: i, value: i * 2, label: "row" });
}
let total = 0;
for (let i = 0; i < rows.length; i = i + 1) {
  total = total + rows[i].value;
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalJSONRoundTrip(b *testing.B) {
	prog := benchmarkProgram(b, `
let rows = [];
for (let i = 0; i < 50; i = i + 1) {
  rows.push({ id: i, name: "item", active: i % 2 === 0 });
}
let text = JSON.stringify({ rows: rows });
let parsed = JSON.parse(text);
parsed.rows.length;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkEvalClassMethods(b *testing.B) {
	prog := benchmarkProgram(b, `
class Counter {
  constructor(start) {
    this.value = start;
  }
  inc(step) {
    this.value = this.value + step;
    return this.value;
  }
}
let c = new Counter(0);
let total = 0;
for (let i = 0; i < 100; i = i + 1) {
  total = total + c.inc(1);
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := object.NewEnvironment()
		RegisterBuiltins(env)
		if result := Eval(prog, env); object.IsRuntimeError(result) {
			b.Fatalf("eval failed: %s", result.Inspect())
		}
	}
}

func BenchmarkRegisterBuiltinsSameVM(b *testing.B) {
	vm := object.NewVirtualMachine()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RegisterBuiltins(vm.NewEnvironment())
	}
}
