package lexer

// Token represents a lexical token with position info.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
	Offset  int
}

// TokenType categorizes each lexeme.
type TokenType string

const (
	// --- Literals ---
	TOKEN_IDENT    TokenType = "IDENT"
	TOKEN_NUMBER   TokenType = "NUMBER"
	TOKEN_STRING   TokenType = "STRING"
	TOKEN_TEMPLATE TokenType = "TEMPLATE"

	// --- Keywords ---
	TOKEN_LET       TokenType = "LET"       // let
	TOKEN_CONST     TokenType = "CONST"     // const
	TOKEN_VAR       TokenType = "VAR"       // var
	TOKEN_FUNCTION  TokenType = "FUNCTION"  // function
	TOKEN_CLASS     TokenType = "CLASS"     // class
	TOKEN_EXTENDS   TokenType = "EXTENDS"   // extends
	TOKEN_IF        TokenType = "IF"        // if
	TOKEN_ELSE      TokenType = "ELSE"      // else
	TOKEN_WHILE     TokenType = "WHILE"     // while
	TOKEN_FOR       TokenType = "FOR"       // for
	TOKEN_IN        TokenType = "IN"        // in
	TOKEN_OF        TokenType = "OF"        // of
	TOKEN_RETURN    TokenType = "RETURN"    // return
	TOKEN_BREAK     TokenType = "BREAK"     // break
	TOKEN_CONTINUE  TokenType = "CONTINUE"  // continue
	TOKEN_TRUE      TokenType = "TRUE"      // true
	TOKEN_FALSE     TokenType = "FALSE"     // false
	TOKEN_NULL      TokenType = "NULL"      // null
	TOKEN_UNDEFINED TokenType = "UNDEFINED" // undefined
	TOKEN_NEW       TokenType = "NEW"       // new
	TOKEN_THIS      TokenType = "THIS"      // this
	TOKEN_SUPER     TokenType = "SUPER"     // super
	TOKEN_TRY       TokenType = "TRY"       // try
	TOKEN_CATCH     TokenType = "CATCH"     // catch
	TOKEN_FINALLY   TokenType = "FINALLY"   // finally
	TOKEN_THROW     TokenType = "THROW"     // throw
	TOKEN_ASYNC     TokenType = "ASYNC"     // async
	TOKEN_AWAIT     TokenType = "AWAIT"     // await
	TOKEN_IMPORT    TokenType = "IMPORT"    // import
	TOKEN_EXPORT    TokenType = "EXPORT"    // export
	TOKEN_FROM      TokenType = "FROM"      // from
	TOKEN_AS        TokenType = "AS"        // as
	TOKEN_DELETE    TokenType = "DELETE"    // delete
	TOKEN_TYPEOF    TokenType = "TYPEOF"    // typeof
	TOKEN_INSTANCEOF TokenType = "INSTANCEOF" // instanceof
	TOKEN_VOID      TokenType = "VOID"      // void
	TOKEN_STATIC    TokenType = "STATIC"    // static
	TOKEN_MATCH     TokenType = "MATCH"     // match

	// --- Single-character operators ---
	TOKEN_PLUS     TokenType = "PLUS"     // +
	TOKEN_MINUS    TokenType = "MINUS"    // -
	TOKEN_STAR     TokenType = "STAR"     // *
	TOKEN_SLASH    TokenType = "SLASH"    // /
	TOKEN_PERCENT  TokenType = "PERCENT"  // %
	TOKEN_BANG     TokenType = "BANG"     // !
	TOKEN_AMP      TokenType = "AMP"      // &
	TOKEN_PIPE     TokenType = "PIPE"     // |
	TOKEN_CARET    TokenType = "CARET"    // ^
	TOKEN_TILDE    TokenType = "TILDE"    // ~
	TOKEN_QUESTION TokenType = "QUESTION" // ?
	TOKEN_COLON    TokenType = "COLON"    // :
	TOKEN_DOT      TokenType = "DOT"      // .

	// --- Multi-character operators ---
	TOKEN_PLUS_PLUS  TokenType = "PLUS_PLUS"   // ++
	TOKEN_MINUS_MINUS TokenType = "MINUS_MINUS" // --
	TOKEN_POW        TokenType = "POW"        // **
	TOKEN_EQ         TokenType = "EQ"         // =
	TOKEN_EQ_EQ_EQ   TokenType = "EQ_EQ_EQ"   // ===
	TOKEN_NEQ_EQ     TokenType = "NEQ_EQ"     // !==
	TOKEN_LT         TokenType = "LT"         // <
	TOKEN_LT_EQ      TokenType = "LT_EQ"      // <=
	TOKEN_GT         TokenType = "GT"         // >
	TOKEN_GT_EQ      TokenType = "GT_EQ"      // >=
	TOKEN_AND_AND    TokenType = "AND_AND"    // &&
	TOKEN_OR_OR      TokenType = "OR_OR"      // ||
	TOKEN_QM_QM      TokenType = "QM_QM"      // ??
	TOKEN_ARROW      TokenType = "ARROW"      // =>
	TOKEN_ELLIPSIS   TokenType = "ELLIPSIS"   // ...
	TOKEN_DOT_DOT    TokenType = "DOT_DOT"    // ..
	TOKEN_DOT_DOT_EQ TokenType = "DOT_DOT_EQ" // ..=
	TOKEN_QM_DOT     TokenType = "QM_DOT"     // ?.

	// --- Compound assignment ---
	TOKEN_PLUS_EQ    TokenType = "PLUS_EQ"
	TOKEN_MINUS_EQ   TokenType = "MINUS_EQ"
	TOKEN_STAR_EQ    TokenType = "STAR_EQ"
	TOKEN_SLASH_EQ   TokenType = "SLASH_EQ"
	TOKEN_PERCENT_EQ TokenType = "PERCENT_EQ"
	TOKEN_POW_EQ     TokenType = "POW_EQ"
	TOKEN_LSHIFT     TokenType = "LSHIFT"     // <<
	TOKEN_RSHIFT     TokenType = "RSHIFT"     // >>
	TOKEN_URSHIFT    TokenType = "URSHIFT"    // >>>
	TOKEN_LSHIFT_EQ  TokenType = "LSHIFT_EQ"
	TOKEN_RSHIFT_EQ  TokenType = "RSHIFT_EQ"
	TOKEN_URSHIFT_EQ TokenType = "URSHIFT_EQ"
	TOKEN_AMP_EQ     TokenType = "AMP_EQ"
	TOKEN_PIPE_EQ    TokenType = "PIPE_EQ"
	TOKEN_CARET_EQ   TokenType = "CARET_EQ"

	// --- Delimiters ---
	TOKEN_LPAREN TokenType = "LPAREN" // (
	TOKEN_RPAREN TokenType = "RPAREN" // )
	TOKEN_LBRACE TokenType = "LBRACE" // {
	TOKEN_RBRACE TokenType = "RBRACE" // }
	TOKEN_LBRACK TokenType = "LBRACK" // [
	TOKEN_RBRACK TokenType = "RBRACK" // ]
	TOKEN_COMMA  TokenType = "COMMA"  // ,
	TOKEN_SEMI   TokenType = "SEMI"   // ;

	// --- Special ---
	TOKEN_EOF     TokenType = "EOF"
	TOKEN_ILLEGAL TokenType = "ILLEGAL"
)

var keywords = map[string]TokenType{
	"let":        TOKEN_LET,
	"const":      TOKEN_CONST,
	"var":        TOKEN_VAR,
	"function":   TOKEN_FUNCTION,
	"class":      TOKEN_CLASS,
	"extends":    TOKEN_EXTENDS,
	"if":         TOKEN_IF,
	"else":       TOKEN_ELSE,
	"while":      TOKEN_WHILE,
	"for":        TOKEN_FOR,
	"in":         TOKEN_IN,
	"of":         TOKEN_OF,
	"return":     TOKEN_RETURN,
	"break":      TOKEN_BREAK,
	"continue":   TOKEN_CONTINUE,
	"true":       TOKEN_TRUE,
	"false":      TOKEN_FALSE,
	"null":       TOKEN_NULL,
	"undefined":  TOKEN_UNDEFINED,
	"new":        TOKEN_NEW,
	"this":       TOKEN_THIS,
	"super":      TOKEN_SUPER,
	"try":        TOKEN_TRY,
	"catch":      TOKEN_CATCH,
	"finally":    TOKEN_FINALLY,
	"throw":      TOKEN_THROW,
	"async":      TOKEN_ASYNC,
	"await":      TOKEN_AWAIT,
	"import":     TOKEN_IMPORT,
	"export":     TOKEN_EXPORT,
	"from":       TOKEN_FROM,
	"as":         TOKEN_AS,
	"delete":     TOKEN_DELETE,
	"typeof":     TOKEN_TYPEOF,
	"instanceof": TOKEN_INSTANCEOF,
	"void":       TOKEN_VOID,
	"static":     TOKEN_STATIC,
	"match":      TOKEN_MATCH,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}

func IsKeyword(t TokenType) bool {
	switch t {
	case TOKEN_LET, TOKEN_CONST, TOKEN_VAR, TOKEN_FUNCTION, TOKEN_CLASS,
		TOKEN_EXTENDS, TOKEN_IF, TOKEN_ELSE, TOKEN_WHILE, TOKEN_FOR,
		TOKEN_IN, TOKEN_OF, TOKEN_RETURN, TOKEN_BREAK, TOKEN_CONTINUE,
		TOKEN_TRUE, TOKEN_FALSE, TOKEN_NULL, TOKEN_UNDEFINED,
		TOKEN_NEW, TOKEN_THIS, TOKEN_SUPER, TOKEN_TRY, TOKEN_CATCH,
		TOKEN_FINALLY, TOKEN_THROW, TOKEN_ASYNC, TOKEN_AWAIT,
		TOKEN_IMPORT, TOKEN_EXPORT, TOKEN_FROM, TOKEN_AS,
		TOKEN_DELETE, TOKEN_TYPEOF, TOKEN_INSTANCEOF, TOKEN_VOID,
		TOKEN_STATIC, TOKEN_MATCH:
		return true
	}
	return false
}

func (t TokenType) String() string {
	if t == "" {
		return "EOF"
	}
	return string(t)
}
