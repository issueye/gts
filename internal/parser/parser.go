package parser

import (
	"fmt"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

// Precedence levels for Pratt parser, from lowest to highest.
const (
	_ int = iota
	PREC_COMMA
	PREC_ASSIGN
	PREC_TERNARY
	PREC_OR_OR
	PREC_AND_AND
	PREC_BIT_OR
	PREC_BIT_XOR
	PREC_BIT_AND
	PREC_EQUALS
	PREC_COMPARE
	PREC_SHIFT
	PREC_SUM
	PREC_PRODUCT
	PREC_EXPONENT
	PREC_PREFIX
	PREC_POSTFIX
	PREC_CALL
)

type (
	prefixFn func() ast.Expression
	infixFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	cur    lexer.Token
	peek   lexer.Token
	buf    []lexer.Token
	marks  []*parserMark
	file   string
	errors []string

	prefixFns map[lexer.TokenType]prefixFn
	infixFns  map[lexer.TokenType]infixFn
}

type parserMark struct {
	cur      lexer.Token
	peek     lexer.Token
	buf      []lexer.Token
	captured []lexer.Token
}

func New(l *lexer.Lexer, file string) *Parser {
	p := &Parser{l: l, file: file, errors: nil}
	p.prefixFns = make(map[lexer.TokenType]prefixFn)
	p.infixFns = make(map[lexer.TokenType]infixFn)

	// Register prefix parsers
	p.registerPrefix(lexer.TOKEN_IDENT, p.parseIdent)
	p.registerPrefix(lexer.TOKEN_NUMBER, p.parseNumber)
	p.registerPrefix(lexer.TOKEN_STRING, p.parseString)
	p.registerPrefix(lexer.TOKEN_TEMPLATE, p.parseTemplate)
	p.registerPrefix(lexer.TOKEN_TRUE, p.parseBool)
	p.registerPrefix(lexer.TOKEN_FALSE, p.parseBool)
	p.registerPrefix(lexer.TOKEN_NULL, p.parseNull)
	p.registerPrefix(lexer.TOKEN_UNDEFINED, p.parseUndefined)
	p.registerPrefix(lexer.TOKEN_THIS, p.parseThis)
	p.registerPrefix(lexer.TOKEN_SUPER, p.parseSuper)
	p.registerPrefix(lexer.TOKEN_BANG, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_MINUS, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_PLUS, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_TILDE, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_TYPEOF, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_VOID, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_DELETE, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_PLUS_PLUS, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_MINUS_MINUS, p.parsePrefix)
	p.registerPrefix(lexer.TOKEN_AWAIT, p.parseAwait)
	p.registerPrefix(lexer.TOKEN_LPAREN, p.parseParenOrArrow)
	p.registerPrefix(lexer.TOKEN_LBRACK, p.parseArray)
	p.registerPrefix(lexer.TOKEN_LBRACE, p.parseObjectOrBlock)
	p.registerPrefix(lexer.TOKEN_NEW, p.parseNew)
	p.registerPrefix(lexer.TOKEN_MATCH, p.parseMatch)
	p.registerPrefix(lexer.TOKEN_FUNCTION, p.parseFunction)
	p.registerPrefix(lexer.TOKEN_ASYNC, p.parseAsyncFunc)
	p.registerPrefix(lexer.TOKEN_CLASS, p.parseClassExpr)

	// Register infix parsers
	for _, t := range []lexer.TokenType{
		lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_STAR, lexer.TOKEN_SLASH,
		lexer.TOKEN_PERCENT, lexer.TOKEN_POW,
		lexer.TOKEN_EQ_EQ_EQ, lexer.TOKEN_NEQ_EQ,
		lexer.TOKEN_LT, lexer.TOKEN_LT_EQ, lexer.TOKEN_GT, lexer.TOKEN_GT_EQ,
		lexer.TOKEN_AND_AND, lexer.TOKEN_OR_OR, lexer.TOKEN_QM_QM,
		lexer.TOKEN_AMP, lexer.TOKEN_PIPE, lexer.TOKEN_CARET,
		lexer.TOKEN_LSHIFT, lexer.TOKEN_RSHIFT, lexer.TOKEN_URSHIFT,
		lexer.TOKEN_IN, lexer.TOKEN_INSTANCEOF,
	} {
		p.registerInfix(t, p.parseInfix)
	}
	p.registerInfix(lexer.TOKEN_LPAREN, p.parseCall)
	p.registerInfix(lexer.TOKEN_DOT, p.parseMember)
	p.registerInfix(lexer.TOKEN_LBRACK, p.parseIndex)
	p.registerInfix(lexer.TOKEN_QM_DOT, p.parseOptional)
	p.registerInfix(lexer.TOKEN_PLUS_PLUS, p.parsePostfix)
	p.registerInfix(lexer.TOKEN_MINUS_MINUS, p.parsePostfix)
	p.registerInfix(lexer.TOKEN_QUESTION, p.parseTernary)
	p.registerInfix(lexer.TOKEN_ARROW, p.parseArrowInfix)
	p.registerInfix(lexer.TOKEN_EQ, p.parseAssign)
	for _, t := range []lexer.TokenType{
		lexer.TOKEN_PLUS_EQ, lexer.TOKEN_MINUS_EQ, lexer.TOKEN_STAR_EQ,
		lexer.TOKEN_SLASH_EQ, lexer.TOKEN_PERCENT_EQ, lexer.TOKEN_POW_EQ,
		lexer.TOKEN_LSHIFT_EQ, lexer.TOKEN_RSHIFT_EQ, lexer.TOKEN_URSHIFT_EQ,
		lexer.TOKEN_AMP_EQ, lexer.TOKEN_PIPE_EQ, lexer.TOKEN_CARET_EQ,
	} {
		p.registerInfix(t, p.parseAssign)
	}

	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return append([]string{}, p.errors...) }

func (p *Parser) registerPrefix(t lexer.TokenType, fn prefixFn) { p.prefixFns[t] = fn }
func (p *Parser) registerInfix(t lexer.TokenType, fn infixFn)   { p.infixFns[t] = fn }

func (p *Parser) nextToken() {
	p.cur = p.peek
	p.peek = p.readToken()
}

func (p *Parser) readToken() lexer.Token {
	var tok lexer.Token
	fromBuffer := false
	if len(p.buf) > 0 {
		tok = p.buf[0]
		p.buf = p.buf[1:]
		fromBuffer = true
	} else {
		tok = p.l.NextToken()
	}
	if !fromBuffer {
		for _, mark := range p.marks {
			mark.captured = append(mark.captured, tok)
		}
	}
	return tok
}

func (p *Parser) unreadTokens(tokens []lexer.Token) {
	if len(tokens) == 0 {
		return
	}
	p.buf = append(append([]lexer.Token{}, tokens...), p.buf...)
}

func (p *Parser) mark() *parserMark {
	m := &parserMark{
		cur:  p.cur,
		peek: p.peek,
		buf:  append([]lexer.Token{}, p.buf...),
	}
	p.marks = append(p.marks, m)
	return m
}

func (p *Parser) rewind(m *parserMark) {
	p.popMark(m)
	p.cur = m.cur
	p.peek = m.peek
	p.buf = append(append([]lexer.Token{}, m.buf...), m.captured...)
}

func (p *Parser) commit(m *parserMark) {
	p.popMark(m)
}

func (p *Parser) popMark(m *parserMark) {
	for i := len(p.marks) - 1; i >= 0; i = i - 1 {
		if p.marks[i] == m {
			p.marks = append(p.marks[:i], p.marks[i+1:]...)
			return
		}
	}
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool  { return p.cur.Type == t }
func (p *Parser) peekTokenIs(t lexer.TokenType) bool { return p.peek.Type == t }

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("%s: %s", p.pos(), msg))
}

func (p *Parser) pos() ast.Position {
	return ast.Position{File: p.file, Line: p.cur.Line, Col: p.cur.Column, Offset: p.cur.Offset}
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s (%q)", t, p.peek.Type, p.peek.Literal))
	return false
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.cur.Type]; ok {
		return pr
	}
	return 0
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peek.Type]; ok {
		return pr
	}
	return 0
}

var precedences = map[lexer.TokenType]int{
	lexer.TOKEN_COMMA:       PREC_COMMA,
	lexer.TOKEN_EQ:          PREC_ASSIGN,
	lexer.TOKEN_PLUS_EQ:     PREC_ASSIGN,
	lexer.TOKEN_MINUS_EQ:    PREC_ASSIGN,
	lexer.TOKEN_STAR_EQ:     PREC_ASSIGN,
	lexer.TOKEN_SLASH_EQ:    PREC_ASSIGN,
	lexer.TOKEN_PERCENT_EQ:  PREC_ASSIGN,
	lexer.TOKEN_POW_EQ:      PREC_ASSIGN,
	lexer.TOKEN_LSHIFT_EQ:   PREC_ASSIGN,
	lexer.TOKEN_RSHIFT_EQ:   PREC_ASSIGN,
	lexer.TOKEN_URSHIFT_EQ:  PREC_ASSIGN,
	lexer.TOKEN_AMP_EQ:      PREC_ASSIGN,
	lexer.TOKEN_PIPE_EQ:     PREC_ASSIGN,
	lexer.TOKEN_CARET_EQ:    PREC_ASSIGN,
	lexer.TOKEN_QUESTION:    PREC_TERNARY,
	lexer.TOKEN_OR_OR:       PREC_OR_OR,
	lexer.TOKEN_AND_AND:     PREC_AND_AND,
	lexer.TOKEN_PIPE:        PREC_BIT_OR,
	lexer.TOKEN_CARET:       PREC_BIT_XOR,
	lexer.TOKEN_AMP:         PREC_BIT_AND,
	lexer.TOKEN_EQ_EQ_EQ:    PREC_EQUALS,
	lexer.TOKEN_NEQ_EQ:      PREC_EQUALS,
	lexer.TOKEN_LT:          PREC_COMPARE,
	lexer.TOKEN_LT_EQ:       PREC_COMPARE,
	lexer.TOKEN_GT:          PREC_COMPARE,
	lexer.TOKEN_GT_EQ:       PREC_COMPARE,
	lexer.TOKEN_IN:          PREC_COMPARE,
	lexer.TOKEN_INSTANCEOF:  PREC_COMPARE,
	lexer.TOKEN_LSHIFT:      PREC_SHIFT,
	lexer.TOKEN_RSHIFT:      PREC_SHIFT,
	lexer.TOKEN_URSHIFT:     PREC_SHIFT,
	lexer.TOKEN_PLUS:        PREC_SUM,
	lexer.TOKEN_MINUS:       PREC_SUM,
	lexer.TOKEN_STAR:        PREC_PRODUCT,
	lexer.TOKEN_SLASH:       PREC_PRODUCT,
	lexer.TOKEN_PERCENT:     PREC_PRODUCT,
	lexer.TOKEN_POW:         PREC_EXPONENT,
	lexer.TOKEN_LPAREN:      PREC_CALL,
	lexer.TOKEN_DOT:         PREC_CALL,
	lexer.TOKEN_LBRACK:      PREC_CALL,
	lexer.TOKEN_QM_DOT:      PREC_CALL,
	lexer.TOKEN_PLUS_PLUS:   PREC_POSTFIX,
	lexer.TOKEN_MINUS_MINUS: PREC_POSTFIX,
	lexer.TOKEN_ARROW:       PREC_ASSIGN,
}

// ============================================================================
// Program & Top-Level
// ============================================================================

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{Pos_: p.pos()}
	for !p.curTokenIs(lexer.TOKEN_EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			prog.Body = append(prog.Body, stmt)
		}
		// parseStatement already leaves p.cur past the end of the statement
	}
	prog.Errors = p.Errors()
	return prog
}

// Synchronize skips to the next known statement boundary for error recovery.
func (p *Parser) sync() {
	for !p.curTokenIs(lexer.TOKEN_EOF) {
		if p.cur.Type == lexer.TOKEN_SEMI {
			p.nextToken()
			return
		}
		if p.cur.Type == lexer.TOKEN_RBRACE {
			return
		}
		if p.cur.Type == lexer.TOKEN_LBRACE {
			p.nextToken()
			return
		}
		switch p.cur.Type {
		case lexer.TOKEN_LET, lexer.TOKEN_CONST, lexer.TOKEN_VAR,
			lexer.TOKEN_FUNCTION, lexer.TOKEN_CLASS,
			lexer.TOKEN_IF, lexer.TOKEN_WHILE, lexer.TOKEN_FOR,
			lexer.TOKEN_RETURN, lexer.TOKEN_BREAK, lexer.TOKEN_CONTINUE,
			lexer.TOKEN_TRY, lexer.TOKEN_THROW,
			lexer.TOKEN_IMPORT, lexer.TOKEN_EXPORT, lexer.TOKEN_MATCH:
			return
		}
		p.nextToken()
	}
}

// ============================================================================
// Statement Parsing
// ============================================================================
