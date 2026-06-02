package parser

import (
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

func Parse(input string) *ast.Program {
	l := lexer.New(input)
	p := New(l, "<test>")
	return p.ParseProgram()
}

func TestParse_LetStmt(t *testing.T) {
	input := `let x = 5;`
	prog := Parse(input)
	checkErrors(t, prog)
	if len(prog.Body) != 1 {
		t.Fatalf("want 1 stmt, got %d", len(prog.Body))
	}
	stmt, ok := prog.Body[0].(*ast.LetStmt)
	if !ok {
		t.Fatalf("want LetStmt, got %T", prog.Body[0])
	}
	if stmt.Name != "x" {
		t.Fatalf("want name 'x', got %q", stmt.Name)
	}
	if stmt.Value == nil {
		t.Fatal("expected value")
	}
}

func TestParse_LetWithType(t *testing.T) {
	input := `let x: number = 42;`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.LetStmt)
	if stmt.TypeAnno == nil || stmt.TypeAnno.Name != "number" {
		t.Fatal("expected type annotation 'number'")
	}
}

func TestParse_ConstStmt(t *testing.T) {
	input := `const PI = 3.14;`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.ConstStmt)
	if !ok {
		t.Fatalf("want ConstStmt, got %T", prog.Body[0])
	}
}

func TestParse_VarStmt(t *testing.T) {
	input := `var x = 1;`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.VarStmt)
	if !ok {
		t.Fatalf("want VarStmt, got %T", prog.Body[0])
	}
}

func TestParse_FuncDecl(t *testing.T) {
	input := `function add(a: number, b: number): number { return a + b; }`
	prog := Parse(input)
	checkErrors(t, prog)
	fn, ok := prog.Body[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("want FuncDecl, got %T", prog.Body[0])
	}
	if fn.Name != "add" {
		t.Fatalf("want name 'add', got %q", fn.Name)
	}
	if len(fn.Params) != 2 {
		t.Fatalf("want 2 params, got %d", len(fn.Params))
	}
}

func TestParse_ReturnStmt(t *testing.T) {
	input := `return 42;`
	prog := Parse(input)
	checkErrors(t, prog)
	ret, ok := prog.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("want ReturnStmt, got %T", prog.Body[0])
	}
	if ret.Value == nil {
		t.Fatal("expected return value")
	}
}

func TestParse_IfStmt(t *testing.T) {
	input := `if (x > 0) { return 1; } else { return -1; }`
	prog := Parse(input)
	checkErrors(t, prog)
	ifStmt, ok := prog.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("want IfStmt, got %T", prog.Body[0])
	}
	if ifStmt.Alternative == nil {
		t.Fatal("expected else branch")
	}
}

func TestParse_WhileStmt(t *testing.T) {
	input := `while (i < 10) { i = i + 1; }`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("want WhileStmt, got %T", prog.Body[0])
	}
}

func TestParse_ForStmt(t *testing.T) {
	input := `for (let i = 0; i < 10; i = i + 1) { console.log(i); }`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("want ForStmt, got %T", prog.Body[0])
	}
}

func TestParse_ForInStmt(t *testing.T) {
	input := `for (let k in obj) { console.log(k); }`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.ForInStmt)
	if !ok {
		t.Fatalf("want ForInStmt, got %T", prog.Body[0])
	}
}

func TestParse_TryCatch(t *testing.T) {
	input := `try { f(); } catch (e) { console.log(e); }`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.TryStmt)
	if !ok {
		t.Fatalf("want TryStmt, got %T", prog.Body[0])
	}
}

func TestParse_Throw(t *testing.T) {
	input := `throw new Error("oops");`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.ThrowStmt)
	if !ok {
		t.Fatalf("want ThrowStmt, got %T", prog.Body[0])
	}
}

func TestParse_BreakContinue(t *testing.T) {
	input := `break outer; continue;`
	prog := Parse(input)
	checkErrors(t, prog)
	b, ok := prog.Body[0].(*ast.BreakStmt)
	if !ok {
		t.Fatalf("want BreakStmt, got %T", prog.Body[0])
	}
	if b.Label != "outer" {
		t.Fatalf("want label 'outer', got %q", b.Label)
	}
	_, ok = prog.Body[1].(*ast.ContinueStmt)
	if !ok {
		t.Fatalf("want ContinueStmt, got %T", prog.Body[1])
	}
}

func TestParse_LabeledStmt(t *testing.T) {
	input := `outer: for (let i = 0; i < 10; i = i + 1) { break outer; }`
	prog := Parse(input)
	checkErrors(t, prog)
	ls, ok := prog.Body[0].(*ast.LabeledStmt)
	if !ok {
		t.Fatalf("want LabeledStmt, got %T", prog.Body[0])
	}
	if ls.Label != "outer" {
		t.Fatalf("want label 'outer', got %q", ls.Label)
	}
}

func TestParse_InfixExpr(t *testing.T) {
	input := `a + b * c;`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	infix, ok := expr.(*ast.InfixExpr)
	if !ok {
		t.Fatalf("want InfixExpr, got %T", expr)
	}
	if infix.Op != "+" {
		t.Fatalf("want op '+', got %q", infix.Op)
	}
}

func TestParse_Precedence(t *testing.T) {
	input := `a + b * c;`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	infix := expr.(*ast.InfixExpr)
	// b * c should be the right subtree
	right := infix.Right.(*ast.InfixExpr)
	if right.Op != "*" {
		t.Fatalf("want op '*', got %q —— precedence wrong", right.Op)
	}
}

func TestParse_StrictEquals(t *testing.T) {
	input := `a === b; c !== d;`
	prog := Parse(input)
	checkErrors(t, prog)
	e1 := prog.Body[0].(*ast.ExprStmt).Expr.(*ast.InfixExpr)
	if e1.Op != "===" {
		t.Fatalf("want '===', got %q", e1.Op)
	}
	e2 := prog.Body[1].(*ast.ExprStmt).Expr.(*ast.InfixExpr)
	if e2.Op != "!==" {
		t.Fatalf("want '!==', got %q", e2.Op)
	}
}

func TestParse_Ternary(t *testing.T) {
	input := `a > 0 ? "yes" : "no";`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	_, ok := expr.(*ast.TernaryExpr)
	if !ok {
		t.Fatalf("want TernaryExpr, got %T", expr)
	}
}

func TestParse_CallExpr(t *testing.T) {
	input := `foo(a, b + c);`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("want CallExpr, got %T", expr)
	}
	if len(call.Args) != 2 {
		t.Fatalf("want 2 args, got %d", len(call.Args))
	}
}

func TestParse_MemberExpr(t *testing.T) {
	input := `obj.prop;`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	_, ok := expr.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("want MemberExpr, got %T", expr)
	}
}

func TestParse_IndexExpr(t *testing.T) {
	input := `arr[0];`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	_, ok := expr.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("want IndexExpr, got %T", expr)
	}
}

func TestParse_OptionalChain(t *testing.T) {
	input := `let a = obj?.prop; let b = obj?.method(); let c = arr?.[0];`
	prog := Parse(input)
	checkErrors(t, prog)
	if len(prog.Body) != 3 {
		t.Fatalf("want 3 stmts, got %d", len(prog.Body))
	}
}

func TestParse_ArrayLit(t *testing.T) {
	input := `let a = [1, 2, a + b, ...rest];`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.LetStmt)
	arr, ok := stmt.Value.(*ast.ArrayLit)
	if !ok {
		t.Fatalf("want ArrayLit, got %T", stmt.Value)
	}
	if len(arr.Elements) != 4 {
		t.Fatalf("want 4 elems, got %d", len(arr.Elements))
	}
}

func TestParse_ObjectLit(t *testing.T) {
	input := `let x = { a, b: 1, [k]: 2, greet() { return "hi"; }, ...other };`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.LetStmt)
	obj, ok := stmt.Value.(*ast.ObjectLit)
	if !ok {
		t.Fatalf("want ObjectLit, got %T", stmt.Value)
	}
	if len(obj.Properties) != 5 {
		t.Fatalf("want 5 props, got %d", len(obj.Properties))
	}
}

func TestParse_ArrowFunc(t *testing.T) {
	input := `let f = (a, b) => a + b;`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.LetStmt)
	_, ok := stmt.Value.(*ast.ArrowFuncExpr)
	if !ok {
		t.Fatalf("want ArrowFuncExpr, got %T", stmt.Value)
	}
}

func TestParse_NewExpr(t *testing.T) {
	input := `new User("Bob", 25);`
	prog := Parse(input)
	checkErrors(t, prog)
	expr := prog.Body[0].(*ast.ExprStmt).Expr
	_, ok := expr.(*ast.NewExpr)
	if !ok {
		t.Fatalf("want NewExpr, got %T", expr)
	}
}

func TestParse_MatchExpr(t *testing.T) {
	input := `
	match code {
		200 => "OK",
		301 | 302 => "Moved",
		n if n >= 500 => "Error",
		10..=20 => "range",
		_ => "unknown",
	}
	`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.ExprStmt)
	m, ok := stmt.Expr.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("want MatchExpr, got %T", stmt.Expr)
	}
	if len(m.Arms) != 5 {
		t.Fatalf("want 5 arms, got %d", len(m.Arms))
	}
}

func TestParse_MatchInAssign(t *testing.T) {
	input := `let label = match n { 1 => "one", _ => "other" };`
	prog := Parse(input)
	checkErrors(t, prog)
	stmt := prog.Body[0].(*ast.LetStmt)
	_, ok := stmt.Value.(*ast.MatchExpr)
	if !ok {
		t.Fatalf("want MatchExpr in assign, got %T", stmt.Value)
	}
}

func TestParse_MatchArmWithBlock(t *testing.T) {
	input := `match cmd { "quit" => { exit(); return; }, _ => {} }`
	prog := Parse(input)
	checkErrors(t, prog)
}

func TestParse_ClassDecl(t *testing.T) {
	input := `class Dog extends Animal { constructor(name: string) { super(name); } bark() { return "woof"; } }`
	prog := Parse(input)
	checkErrors(t, prog)
	c, ok := prog.Body[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("want ClassDecl, got %T", prog.Body[0])
	}
	if c.Name != "Dog" {
		t.Fatalf("want 'Dog', got %q", c.Name)
	}
	if c.Super == nil {
		t.Fatal("expected super class")
	}
}

func TestParse_ImportExport(t *testing.T) {
	input := `import { add } from "./math.gs"; export function main() {}`
	prog := Parse(input)
	checkErrors(t, prog)
	_, ok := prog.Body[0].(*ast.ImportDecl)
	if !ok {
		t.Fatalf("want ImportDecl, got %T", prog.Body[0])
	}
	_, ok = prog.Body[1].(*ast.ExportDecl)
	if !ok {
		t.Fatalf("want ExportDecl, got %T", prog.Body[1])
	}
}

func TestParse_FullExample(t *testing.T) {
	input := `
async function main(): void {
  let x: number = 42;
  let result: string = match x {
    n if n > 0 => "positive",
    _ => "other",
  };
  console.log(result);
}
`
	prog := Parse(input)
	errors := prog.Errors
	if len(errors) > 0 {
		for _, e := range errors {
			t.Logf("parser error: %s", e)
		}
		t.Fatalf("unexpected parser errors: %d", len(errors))
	}
	if len(prog.Body) != 1 {
		t.Fatalf("want 1 stmt, got %d", len(prog.Body))
	}
}

func TestParse_PositionOnNodes(t *testing.T) {
	input := `let x = 42;`
	prog := Parse(input)
	checkErrors(t, prog)

	stmt := prog.Body[0].(*ast.LetStmt)
	// Line must be correct for error reporting
	if stmt.Pos().Line != 1 {
		t.Fatalf("want line 1, got %d", stmt.Pos().Line)
	}
	if stmt.Pos().IsZero() {
		t.Fatal("position should not be zero")
	}
}

func TestParse_ErrorWithPosition(t *testing.T) {
	input := `let a == b;`
	l := lexer.New(input)
	p := New(l, "<test>")
	prog := p.ParseProgram()
	if len(prog.Errors) == 0 {
		t.Fatal("expected errors for ==")
	}
	for _, e := range prog.Errors {
		t.Logf("parser error: %s", e)
	}
}

func TestParse_MultiLinePosition(t *testing.T) {
	input := "let a = 1;\nlet b = 2;\nlet c = 3;"
	prog := Parse(input)
	checkErrors(t, prog)
	// Third statement should be on line 3
	s3 := prog.Body[2].(*ast.LetStmt)
	pos := s3.Pos()
	if pos.Line != 3 {
		t.Fatalf("third let on line 3, got line %d", pos.Line)
	}
}

func checkErrors(t *testing.T, prog *ast.Program) {
	t.Helper()
	if len(prog.Errors) > 0 {
		for _, e := range prog.Errors {
			t.Logf("parser error: %s", e)
		}
		t.Fatalf("unexpected parser errors: %d", len(prog.Errors))
	}
}
