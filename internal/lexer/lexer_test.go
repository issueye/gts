package lexer

import (
	"testing"
)

func tokenEq(t *testing.T, input string, expected []struct {
	Type TokenType
	Lit  string
}) {
	l := New(input)
	for i, e := range expected {
		tok := l.NextToken()
		if tok.Type != e.Type {
			t.Fatalf("[%d] input=%q: want type %q, got %q (lit=%q)", i, input, e.Type, tok.Type, tok.Literal)
		}
		if tok.Literal != e.Lit {
			t.Fatalf("[%d] input=%q: want literal %q, got %q", i, input, e.Lit, tok.Literal)
		}
	}
}

func TestLexer_BasicTokens(t *testing.T) {
	input := `( ) { } [ ] , ; : . _ $`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_LPAREN, "("},
		{TOKEN_RPAREN, ")"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_LBRACK, "["},
		{TOKEN_RBRACK, "]"},
		{TOKEN_COMMA, ","},
		{TOKEN_SEMI, ";"},
		{TOKEN_COLON, ":"},
		{TOKEN_DOT, "."},
		{TOKEN_IDENT, "_"},
		{TOKEN_IDENT, "$"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Keywords(t *testing.T) {
	input := `let const var function class extends if else while for in of return break continue true false null undefined new this super try catch finally throw async await import export from as delete typeof instanceof void static match default`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_LET, "let"}, {TOKEN_CONST, "const"}, {TOKEN_VAR, "var"},
		{TOKEN_FUNCTION, "function"}, {TOKEN_CLASS, "class"}, {TOKEN_EXTENDS, "extends"},
		{TOKEN_IF, "if"}, {TOKEN_ELSE, "else"}, {TOKEN_WHILE, "while"}, {TOKEN_FOR, "for"},
		{TOKEN_IN, "in"}, {TOKEN_OF, "of"},
		{TOKEN_RETURN, "return"}, {TOKEN_BREAK, "break"}, {TOKEN_CONTINUE, "continue"},
		{TOKEN_TRUE, "true"}, {TOKEN_FALSE, "false"}, {TOKEN_NULL, "null"}, {TOKEN_UNDEFINED, "undefined"},
		{TOKEN_NEW, "new"}, {TOKEN_THIS, "this"}, {TOKEN_SUPER, "super"},
		{TOKEN_TRY, "try"}, {TOKEN_CATCH, "catch"}, {TOKEN_FINALLY, "finally"}, {TOKEN_THROW, "throw"},
		{TOKEN_ASYNC, "async"}, {TOKEN_AWAIT, "await"},
		{TOKEN_IMPORT, "import"}, {TOKEN_EXPORT, "export"}, {TOKEN_FROM, "from"}, {TOKEN_AS, "as"},
		{TOKEN_DELETE, "delete"}, {TOKEN_TYPEOF, "typeof"}, {TOKEN_INSTANCEOF, "instanceof"}, {TOKEN_VOID, "void"},
		{TOKEN_STATIC, "static"}, {TOKEN_MATCH, "match"}, {TOKEN_DEFAULT, "default"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Operators(t *testing.T) {
	input := `+ - * / % ++ -- ** = === !== < <= > >= && || ?? ! & | ^ ~ =>`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_PLUS, "+"}, {TOKEN_MINUS, "-"}, {TOKEN_STAR, "*"}, {TOKEN_SLASH, "/"}, {TOKEN_PERCENT, "%"},
		{TOKEN_PLUS_PLUS, "++"}, {TOKEN_MINUS_MINUS, "--"},
		{TOKEN_POW, "**"},
		{TOKEN_EQ, "="}, {TOKEN_EQ_EQ_EQ, "==="}, {TOKEN_NEQ_EQ, "!=="},
		{TOKEN_LT, "<"}, {TOKEN_LT_EQ, "<="}, {TOKEN_GT, ">"}, {TOKEN_GT_EQ, ">="},
		{TOKEN_AND_AND, "&&"}, {TOKEN_OR_OR, "||"}, {TOKEN_QM_QM, "??"},
		{TOKEN_BANG, "!"}, {TOKEN_AMP, "&"}, {TOKEN_PIPE, "|"}, {TOKEN_CARET, "^"}, {TOKEN_TILDE, "~"},
		{TOKEN_ARROW, "=>"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_CompoundAssignment(t *testing.T) {
	input := `+= -= *= /= %= **= <<= >>= >>>= &= |= ^= << >> >>> ?.`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_PLUS_EQ, "+="}, {TOKEN_MINUS_EQ, "-="}, {TOKEN_STAR_EQ, "*="},
		{TOKEN_SLASH_EQ, "/="}, {TOKEN_PERCENT_EQ, "%="}, {TOKEN_POW_EQ, "**="},
		{TOKEN_LSHIFT_EQ, "<<="}, {TOKEN_RSHIFT_EQ, ">>="}, {TOKEN_URSHIFT_EQ, ">>>="},
		{TOKEN_AMP_EQ, "&="}, {TOKEN_PIPE_EQ, "|="}, {TOKEN_CARET_EQ, "^="},
		{TOKEN_LSHIFT, "<<"}, {TOKEN_RSHIFT, ">>"}, {TOKEN_URSHIFT, ">>>"},
		{TOKEN_QM_DOT, "?."},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_RangeAndSpread(t *testing.T) {
	input := `.. ..= ...`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_DOT_DOT, ".."},
		{TOKEN_DOT_DOT_EQ, "..="},
		{TOKEN_ELLIPSIS, "..."},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Numbers(t *testing.T) {
	input := `42 0 3.14 1e3 2e-5 0xFF 0b1010 0o17`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_NUMBER, "42"},
		{TOKEN_NUMBER, "0"},
		{TOKEN_NUMBER, "3.14"},
		{TOKEN_NUMBER, "1e3"},
		{TOKEN_NUMBER, "2e-5"},
		{TOKEN_NUMBER, "0xFF"},
		{TOKEN_NUMBER, "0b1010"},
		{TOKEN_NUMBER, "0o17"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Strings(t *testing.T) {
	input := `"hello" 'world' "esc\\n" 'it\'s'`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_STRING, `"hello"`},
		{TOKEN_STRING, `'world'`},
		{TOKEN_STRING, `"esc\\n"`},
		{TOKEN_STRING, `'it\'s'`},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Template(t *testing.T) {
	input := "`hello ${name}`"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_TEMPLATE {
		t.Fatalf("want TEMPLATE, got %q", tok.Type)
	}
	if tok.Literal != "`hello ${name}`" {
		t.Fatalf("want template literal %q, got %q", "`hello ${name}`", tok.Literal)
	}
}

func TestLexer_TemplateExpressionDoesNotConsumeFollowingToken(t *testing.T) {
	input := "`value ${{ value: 1 }.value}`;"
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_TEMPLATE, "`value ${{ value: 1 }.value}`"},
		{TOKEN_SEMI, ";"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_RegExpLiteral(t *testing.T) {
	input := `let ansi = /\x1b\[[0-?]*[ -/]*[@-~]/g; let div = a / b;`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_LET, "let"},
		{TOKEN_IDENT, "ansi"},
		{TOKEN_EQ, "="},
		{TOKEN_REGEXP, `/\x1b\[[0-?]*[ -/]*[@-~]/g`},
		{TOKEN_SEMI, ";"},
		{TOKEN_LET, "let"},
		{TOKEN_IDENT, "div"},
		{TOKEN_EQ, "="},
		{TOKEN_IDENT, "a"},
		{TOKEN_SLASH, "/"},
		{TOKEN_IDENT, "b"},
		{TOKEN_SEMI, ";"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_Comments(t *testing.T) {
	input := `let x /* block */ = // line
5`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_LET, "let"},
		{TOKEN_IDENT, "x"},
		{TOKEN_EQ, "="},
		{TOKEN_NUMBER, "5"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_NestedBlockComments(t *testing.T) {
	input := `1 /* outer /* inner */ still */ 2`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_NUMBER, "1"},
		{TOKEN_NUMBER, "2"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}

func TestLexer_DoubleEq_Error(t *testing.T) {
	input := `a == b`
	l := New(input)
	l.NextToken() // a
	tok := l.NextToken()
	if tok.Type != TOKEN_ILLEGAL {
		t.Fatalf("want TOKEN_ILLEGAL for '==', got %q", tok.Type)
	}
	if tok.Literal != "==" {
		t.Fatalf("want literal '==', got %q", tok.Literal)
	}
	if len(l.Errors()) == 0 {
		t.Fatal("expected error for ==")
	}
}

func TestLexer_NotEq_Error(t *testing.T) {
	input := `a != b`
	l := New(input)
	l.NextToken() // a
	tok := l.NextToken()
	if tok.Type != TOKEN_ILLEGAL {
		t.Fatalf("want TOKEN_ILLEGAL for '!=', got %q", tok.Type)
	}
	if len(l.Errors()) == 0 {
		t.Fatal("expected error for !=")
	}
}

func TestLexer_PositionTracking(t *testing.T) {
	input := "let\nx"
	l := New(input)
	tok := l.NextToken()
	if tok.Line != 1 || tok.Column != 1 {
		t.Fatalf("'let' at 1:1, got %d:%d", tok.Line, tok.Column)
	}
	tok = l.NextToken()
	if tok.Line != 2 || tok.Column != 1 {
		t.Fatalf("'x' at 2:1, got %d:%d", tok.Line, tok.Column)
	}
}

func TestLexer_UnicodeIdent(t *testing.T) {
	input := `变量 _名字 $测试`
	l := New(input)
	tok := l.NextToken()
	if tok.Type != TOKEN_IDENT || tok.Literal != "变量" {
		t.Fatalf("want IDENT '变量', got %q %q", tok.Type, tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_IDENT || tok.Literal != "_名字" {
		t.Fatalf("want IDENT '_名字', got %q %q", tok.Type, tok.Literal)
	}
	tok = l.NextToken()
	if tok.Type != TOKEN_IDENT || tok.Literal != "$测试" {
		t.Fatalf("want IDENT '$测试', got %q %q", tok.Type, tok.Literal)
	}
}

func TestLexer_FullExample(t *testing.T) {
	input := `async function main(): void {
  let x: number = 42;
  let result: string = match x {
    n if n > 0 => "positive",
    _ => "other",
  };
  console.log(result);
}`
	tests := []struct {
		Type TokenType
		Lit  string
	}{
		{TOKEN_ASYNC, "async"}, {TOKEN_FUNCTION, "function"}, {TOKEN_IDENT, "main"},
		{TOKEN_LPAREN, "("}, {TOKEN_RPAREN, ")"}, {TOKEN_COLON, ":"}, {TOKEN_VOID, "void"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_LET, "let"}, {TOKEN_IDENT, "x"}, {TOKEN_COLON, ":"}, {TOKEN_IDENT, "number"},
		{TOKEN_EQ, "="}, {TOKEN_NUMBER, "42"}, {TOKEN_SEMI, ";"},
		{TOKEN_LET, "let"}, {TOKEN_IDENT, "result"}, {TOKEN_COLON, ":"}, {TOKEN_IDENT, "string"},
		{TOKEN_EQ, "="}, {TOKEN_MATCH, "match"}, {TOKEN_IDENT, "x"},
		{TOKEN_LBRACE, "{"},
		{TOKEN_IDENT, "n"}, {TOKEN_IF, "if"}, {TOKEN_IDENT, "n"}, {TOKEN_GT, ">"},
		{TOKEN_NUMBER, "0"}, {TOKEN_ARROW, "=>"}, {TOKEN_STRING, `"positive"`}, {TOKEN_COMMA, ","},
		{TOKEN_IDENT, "_"}, {TOKEN_ARROW, "=>"}, {TOKEN_STRING, `"other"`}, {TOKEN_COMMA, ","},
		{TOKEN_RBRACE, "}"}, {TOKEN_SEMI, ";"},
		{TOKEN_IDENT, "console"}, {TOKEN_DOT, "."}, {TOKEN_IDENT, "log"},
		{TOKEN_LPAREN, "("}, {TOKEN_IDENT, "result"}, {TOKEN_RPAREN, ")"}, {TOKEN_SEMI, ";"},
		{TOKEN_RBRACE, "}"},
		{TOKEN_EOF, ""},
	}
	tokenEq(t, input, tests)
}
