package parser

import (
	"fmt"
	"strconv"

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
	file   string
	errors []string

	prefixFns map[lexer.TokenType]prefixFn
	infixFns  map[lexer.TokenType]infixFn
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
	if len(p.buf) > 0 {
		tok := p.buf[0]
		p.buf = p.buf[1:]
		return tok
	}
	return p.l.NextToken()
}

func (p *Parser) unreadTokens(tokens []lexer.Token) {
	if len(tokens) == 0 {
		return
	}
	p.buf = append(append([]lexer.Token{}, tokens...), p.buf...)
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

func (p *Parser) parseStatement() ast.Statement {
	switch p.cur.Type {
	case lexer.TOKEN_LET:
		return p.parseVarDecl("let")
	case lexer.TOKEN_CONST:
		return p.parseVarDecl("const")
	case lexer.TOKEN_VAR:
		return p.parseVarDecl("var")
	case lexer.TOKEN_IF:
		return p.parseIf()
	case lexer.TOKEN_WHILE:
		return p.parseWhile()
	case lexer.TOKEN_FOR:
		return p.parseFor()
	case lexer.TOKEN_RETURN:
		return p.parseReturn()
	case lexer.TOKEN_BREAK:
		return p.parseBreak()
	case lexer.TOKEN_CONTINUE:
		return p.parseContinue()
	case lexer.TOKEN_THROW:
		return p.parseThrow()
	case lexer.TOKEN_TRY:
		return p.parseTry()
	case lexer.TOKEN_LBRACE:
		return p.parseBlock()
	case lexer.TOKEN_FUNCTION:
		return p.parseFuncDecl()
	case lexer.TOKEN_CLASS:
		return p.parseClassDecl()
	case lexer.TOKEN_ASYNC:
		return p.parseAsyncFuncDecl()
	case lexer.TOKEN_IMPORT:
		return p.parseImport()
	case lexer.TOKEN_EXPORT:
		return p.parseExport()
	case lexer.TOKEN_MATCH:
		me := p.parseMatch()
		p.skipSemicolon()
		return &ast.ExprStmt{Pos_: p.pos(), Expr: me}
	case lexer.TOKEN_IDENT:
		if p.peekTokenIs(lexer.TOKEN_COLON) {
			name := p.cur.Literal
			p.nextToken() // ident
			p.nextToken() // colon
			stmt := p.parseStatement()
			return &ast.LabeledStmt{Pos_: p.pos(), Label: name, Stmt: stmt}
		}
		fallthrough
	default:
		expr := p.parseExpression(PREC_COMMA)
		if expr == nil {
			p.sync()
			return nil
		}
		p.skipSemicolon()
		return &ast.ExprStmt{Pos_: p.pos(), Expr: expr}
	}
}

func (p *Parser) consumeSemi() bool {
	if p.curTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken()
		return true
	}
	return false
}

func (p *Parser) skipSemicolon() {
	if p.curTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken()
	}
}

// ============================================================================
// Variable Declarations
// ============================================================================

func (p *Parser) parseVarDecl(kind string) ast.Statement {
	tokLit := p.cur.Literal
	p.nextToken()         // skip let/const/var
	name := p.cur.Literal // identifier

	var tAnno *ast.TypeAnnotation
	p.nextToken() // advance past identifier, cur = : = or ;
	if p.curTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()         // cur = type name
		tAnno = p.parseType() // parseType advances to next token
	}

	var val ast.Expression
	if p.curTokenIs(lexer.TOKEN_EQ) {
		p.nextToken() // cur = start of value expression
		val = p.parseExpression(PREC_COMMA)
	}

	p.skipSemicolon()

	switch kind {
	case "let":
		return &ast.LetStmt{Pos_: p.pos(), TokenLit: tokLit, Name: name, TypeAnno: tAnno, Value: val}
	case "const":
		return &ast.ConstStmt{Pos_: p.pos(), TokenLit: tokLit, Name: name, TypeAnno: tAnno, Value: val}
	default:
		return &ast.VarStmt{Pos_: p.pos(), TokenLit: tokLit, Name: name, TypeAnno: tAnno, Value: val}
	}
}

// ============================================================================
// Block
// ============================================================================

func (p *Parser) parseBlock() *ast.BlockStmt {
	tokLit := p.cur.Literal
	p.nextToken() // {
	stmts := make([]ast.Statement, 0)
	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		s := p.parseStatement()
		if s != nil {
			stmts = append(stmts, s)
		}
	}
	p.nextToken() // }
	return &ast.BlockStmt{Pos_: p.pos(), TokenLit: tokLit, Statements: stmts}
}

// ============================================================================
// If / While / For
// ============================================================================

func (p *Parser) parseIf() *ast.IfStmt {
	tokLit := p.cur.Literal
	if !p.peekTokenIs(lexer.TOKEN_LPAREN) {
		p.addError("expected ( after if")
		return nil
	}
	p.nextToken() // skip if, cur = (
	p.nextToken() // skip (, cur = condition
	cond := p.parseExpression(PREC_COMMA)
	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.addError("expected ) after if condition")
	}
	p.nextToken() // skip ), cur = {
	cons := p.parseBlock()
	var alt ast.Statement
	if p.curTokenIs(lexer.TOKEN_ELSE) {
		p.nextToken() // skip else
		if p.curTokenIs(lexer.TOKEN_IF) {
			alt = p.parseIf()
		} else {
			alt = p.parseBlock()
		}
	}
	return &ast.IfStmt{Pos_: p.pos(), TokenLit: tokLit, Cond: cond, Consequence: cons, Alternative: alt}
}

func (p *Parser) parseWhile() *ast.WhileStmt {
	tokLit := p.cur.Literal
	if !p.peekTokenIs(lexer.TOKEN_LPAREN) {
		p.addError("expected ( after while")
		return nil
	}
	p.nextToken() // while, cur = (
	p.nextToken() // (, cur = condition
	cond := p.parseExpression(PREC_COMMA)
	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.addError("expected ) after while condition")
	}
	p.nextToken() // ), cur = {
	body := p.parseBlock()
	return &ast.WhileStmt{Pos_: p.pos(), TokenLit: tokLit, Cond: cond, Body: body}
}

func (p *Parser) parseFor() ast.Statement {
	tokLit := p.cur.Literal
	if !p.peekTokenIs(lexer.TOKEN_LPAREN) {
		p.addError("expected ( after for")
		return nil
	}
	p.nextToken() // for
	p.nextToken() // (, cur = first token in for header

	// for-in / for-of: "for (let x in arr)" or "for (let x of arr)"
	maybeForIn := p.curTokenIs(lexer.TOKEN_LET) || p.curTokenIs(lexer.TOKEN_CONST) || p.curTokenIs(lexer.TOKEN_VAR) || p.curTokenIs(lexer.TOKEN_IDENT)
	if maybeForIn {
		saveCur, savePeek := p.cur, p.peek
		consumed := make([]lexer.Token, 0, 2)
		advance := func() {
			p.nextToken()
			consumed = append(consumed, p.peek)
		}
		if p.curTokenIs(lexer.TOKEN_LET) || p.curTokenIs(lexer.TOKEN_CONST) || p.curTokenIs(lexer.TOKEN_VAR) {
			advance() // skip let/const/var
		}
		name := p.cur.Literal
		advance() // skip ident
		if p.curTokenIs(lexer.TOKEN_IN) {
			p.nextToken() // skip in
			iterable := p.parseExpression(PREC_COMMA)
			if !p.curTokenIs(lexer.TOKEN_RPAREN) {
				p.addError("expected )")
			}
			p.nextToken() // skip )
			body := p.parseBlock()
			return &ast.ForInStmt{Pos_: p.pos(), TokenLit: tokLit, Name: name, Iterable: iterable, Body: body}
		}
		if p.curTokenIs(lexer.TOKEN_OF) {
			p.nextToken() // skip of
			iterable := p.parseExpression(PREC_COMMA)
			if !p.curTokenIs(lexer.TOKEN_RPAREN) {
				p.addError("expected )")
			}
			p.nextToken() // skip )
			body := p.parseBlock()
			return &ast.ForOfStmt{Pos_: p.pos(), TokenLit: tokLit, Name: name, Iterable: iterable, Body: body}
		}
		// Not for-in/for-of, backtrack
		p.cur = saveCur
		p.peek = savePeek
		p.unreadTokens(consumed)
	}

	// C-style for: for (init; cond; post) body
	var init ast.Statement
	var cond ast.Expression
	var post ast.Expression

	if !p.curTokenIs(lexer.TOKEN_SEMI) {
		if p.curTokenIs(lexer.TOKEN_LET) || p.curTokenIs(lexer.TOKEN_CONST) || p.curTokenIs(lexer.TOKEN_VAR) {
			init = p.parseVarDecl("let")
		} else {
			init = &ast.ExprStmt{Pos_: p.pos(), Expr: p.parseExpression(PREC_COMMA)}
		}
	}
	// parseVarDecl or parseExpression already consumed its semicolon; cur is now at next part

	if !p.curTokenIs(lexer.TOKEN_SEMI) {
		cond = p.parseExpression(PREC_COMMA)
	}
	if p.curTokenIs(lexer.TOKEN_SEMI) {
		p.nextToken() // skip ;
	} else if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.addError("expected ; or ) in for")
	}

	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		post = p.parseExpression(PREC_COMMA)
	}
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken() // skip )
	} else {
		p.addError("expected ) after for")
	}
	body := p.parseBlock()
	return &ast.ForStmt{Pos_: p.pos(), TokenLit: tokLit, Init: init, Cond: cond, Post: post, Body: body}
}

// ============================================================================
// Return / Break / Continue / Throw
// ============================================================================

func (p *Parser) parseReturn() *ast.ReturnStmt {
	tokLit := p.cur.Literal
	p.nextToken()
	var val ast.Expression
	if !p.curTokenIs(lexer.TOKEN_SEMI) && !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		val = p.parseExpression(PREC_COMMA)
	}
	p.skipSemicolon()
	return &ast.ReturnStmt{Pos_: p.pos(), TokenLit: tokLit, Value: val}
}

func (p *Parser) parseBreak() *ast.BreakStmt {
	tokLit := p.cur.Literal
	label := ""
	if p.peekTokenIs(lexer.TOKEN_IDENT) {
		p.nextToken()
		label = p.cur.Literal
	}
	p.nextToken()
	p.skipSemicolon()
	return &ast.BreakStmt{Pos_: p.pos(), TokenLit: tokLit, Label: label}
}

func (p *Parser) parseContinue() *ast.ContinueStmt {
	tokLit := p.cur.Literal
	label := ""
	if p.peekTokenIs(lexer.TOKEN_IDENT) {
		p.nextToken()
		label = p.cur.Literal
	}
	p.nextToken()
	p.skipSemicolon()
	return &ast.ContinueStmt{Pos_: p.pos(), TokenLit: tokLit, Label: label}
}

func (p *Parser) parseThrow() *ast.ThrowStmt {
	tokLit := p.cur.Literal
	p.nextToken()
	val := p.parseExpression(PREC_COMMA)
	p.skipSemicolon()
	return &ast.ThrowStmt{Pos_: p.pos(), TokenLit: tokLit, Value: val}
}

// ============================================================================
// Try / Catch / Finally
// ============================================================================

func (p *Parser) parseTry() *ast.TryStmt {
	tokLit := p.cur.Literal
	p.nextToken() // skip try, cur = {
	block := p.parseBlock()

	var catch *ast.CatchClause
	if p.curTokenIs(lexer.TOKEN_CATCH) {
		p.nextToken() // catch, cur = (
		p.nextToken() // (, cur = ident
		catch = &ast.CatchClause{Pos_: p.pos()}
		if p.curTokenIs(lexer.TOKEN_IDENT) {
			catch.Name = p.cur.Literal
			if p.peekTokenIs(lexer.TOKEN_COLON) {
				p.nextToken() // ident
				p.nextToken() // colon
				catch.TypeAnno = p.parseType()
			}
		}
		p.nextToken() // past ident or type, cur = )
		if !p.curTokenIs(lexer.TOKEN_RPAREN) {
			p.addError("expected ) after catch")
		}
		p.nextToken() // skip ), cur = {
		catch.Body = p.parseBlock()
	}

	var finalizer *ast.BlockStmt
	if p.curTokenIs(lexer.TOKEN_FINALLY) {
		p.nextToken() // skip finally, cur = {
		finalizer = p.parseBlock()
	}
	return &ast.TryStmt{Pos_: p.pos(), TokenLit: tokLit, Block: block, Catch: catch, Finalizer: finalizer}
}

// ============================================================================
// Function / Class Declaration
// ============================================================================

func (p *Parser) parseFuncDecl() *ast.FuncDecl {
	tokLit := p.cur.Literal
	p.expectPeek(lexer.TOKEN_IDENT)
	name := p.cur.Literal
	p.nextToken()
	params, retT := p.parseFuncParams()
	body := p.parseBlock()
	return &ast.FuncDecl{Pos_: p.pos(), TokenLit: tokLit, Name: name, Params: params, ReturnT: retT, Body: body}
}

func (p *Parser) parseAsyncFuncDecl() *ast.FuncDecl {
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_FUNCTION) {
		fn := p.parseFuncDecl()
		fn.IsAsync = true
		return fn
	}
	return nil
}

func (p *Parser) parseFuncParams() ([]*ast.Param, *ast.TypeAnnotation) {
	if !p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.addError("expected (")
		return nil, nil
	}
	p.nextToken() // (
	params := make([]*ast.Param, 0)
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken() // )
		var retT *ast.TypeAnnotation
		if p.curTokenIs(lexer.TOKEN_COLON) {
			p.nextToken()
			retT = p.parseType()
		}
		return params, retT
	}
	for {
		spread := false
		if p.curTokenIs(lexer.TOKEN_ELLIPSIS) {
			spread = true
			p.nextToken()
		}
		param := p.parseParam(spread)
		if p.curTokenIs(lexer.TOKEN_EQ) {
			p.nextToken()
			param.Default = p.parseExpression(PREC_COMMA)
		}
		params = append(params, param)
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.addError("expected )")
	}
	p.nextToken() // )
	var retT *ast.TypeAnnotation
	if p.curTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		retT = p.parseType()
	}
	return params, retT
}

func (p *Parser) parseClassDecl() *ast.ClassDecl {
	tokLit := p.cur.Literal
	if !p.peekTokenIs(lexer.TOKEN_IDENT) {
		p.addError("expected class name")
		return nil
	}
	p.nextToken() // class
	name := p.cur.Literal
	p.nextToken() // name

	var super ast.Expression
	if p.curTokenIs(lexer.TOKEN_EXTENDS) {
		p.nextToken() // extends
		super = p.parseExpression(PREC_COMMA)
	}
	body := p.parseClassBody()
	return &ast.ClassDecl{Pos_: p.pos(), TokenLit: tokLit, Name: name, Super: super, Body: body}
}

func (p *Parser) parseClassBody() *ast.ClassBody {
	if !p.curTokenIs(lexer.TOKEN_LBRACE) {
		p.addError("expected {")
		return nil
	}
	p.nextToken() // {
	body := &ast.ClassBody{Pos_: p.pos()}
	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		member := p.parseClassMember()
		if member != nil {
			body.Members = append(body.Members, member)
		}
	}
	p.nextToken() // }
	return body
}

func (p *Parser) parseClassMember() *ast.ClassMember {
	if p.curTokenIs(lexer.TOKEN_SEMI) || p.curTokenIs(lexer.TOKEN_RBRACE) {
		return nil
	}
	isStatic := false
	isAsync := false
	if p.curTokenIs(lexer.TOKEN_STATIC) {
		isStatic = true
		p.nextToken()
	}
	if p.curTokenIs(lexer.TOKEN_ASYNC) {
		isAsync = true
		p.nextToken()
	}
	name := p.cur.Literal
	p.nextToken() // skip name

	var params []*ast.Param
	var body *ast.BlockStmt
	var tAnno *ast.TypeAnnotation
	var defaultVal ast.Expression
	kind := "field"

	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		// method or constructor
		if name == "constructor" {
			kind = "constructor"
		} else {
			kind = "method"
		}
		params, _ = p.parseFuncParams()
		body = p.parseBlock()
	} else if p.curTokenIs(lexer.TOKEN_COLON) {
		// field with type
		p.nextToken()
		tAnno = p.parseType()
		if p.curTokenIs(lexer.TOKEN_EQ) {
			p.nextToken()
			defaultVal = p.parseExpression(PREC_COMMA)
		}
		p.skipSemicolon()
	} else if p.curTokenIs(lexer.TOKEN_EQ) {
		// field with default value
		p.nextToken()
		defaultVal = p.parseExpression(PREC_COMMA)
		p.skipSemicolon()
	} else {
		// field without type or value
		p.skipSemicolon()
	}

	return &ast.ClassMember{Pos_: p.pos(), IsStatic: isStatic, IsAsync: isAsync, Name: name, Params: params, Body: body, TypeAnno: tAnno, DefaultVal: defaultVal, Kind: kind}
}

// ============================================================================
// Import / Export
// ============================================================================

func (p *Parser) parseImport() *ast.ImportDecl {
	tokLit := p.cur.Literal
	p.nextToken() // import
	decl := &ast.ImportDecl{Pos_: p.pos(), TokenLit: tokLit, Aliases: make(map[string]string)}
	if p.curTokenIs(lexer.TOKEN_STAR) {
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_AS) {
			p.addError("expected as after * in namespace import")
			return decl
		}
		p.nextToken()
		decl.Namespace = p.cur.Literal
		p.nextToken()
	}
	if p.curTokenIs(lexer.TOKEN_IDENT) && !p.peekTokenIs(lexer.TOKEN_FROM) && !p.peekTokenIs(lexer.TOKEN_LBRACE) {
		decl.Default = p.cur.Literal
		if p.peekTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			p.nextToken()
		} else {
			p.nextToken()
		}
	}
	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		p.nextToken()
		for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
			name := p.cur.Literal
			p.nextToken()
			if p.curTokenIs(lexer.TOKEN_AS) {
				p.nextToken()
				decl.Aliases[name] = p.cur.Literal
				p.nextToken()
			} else {
				decl.Names = append(decl.Names, name)
			}
			if p.curTokenIs(lexer.TOKEN_COMMA) {
				p.nextToken()
			}
		}
		p.nextToken() // }
	}
	if !p.curTokenIs(lexer.TOKEN_FROM) {
		p.addError("expected from in import")
		return decl
	}
	p.nextToken() // from
	decl.Source = p.cur.Literal
	p.nextToken() // string
	p.skipSemicolon()
	return decl
}

func (p *Parser) parseExport() *ast.ExportDecl {
	tokLit := p.cur.Literal
	p.nextToken() // export
	isDefault := false
	if p.curTokenIs(lexer.TOKEN_DEFAULT) {
		isDefault = true
		p.nextToken()
	}
	var decl ast.Statement
	var specs []ast.ExportSpec
	if !isDefault {
		if p.curTokenIs(lexer.TOKEN_LBRACE) {
			p.nextToken()
			for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
				name := p.cur.Literal
				alias := name
				p.nextToken()
				if p.curTokenIs(lexer.TOKEN_AS) {
					p.nextToken()
					alias = p.cur.Literal
					p.nextToken()
				}
				specs = append(specs, ast.ExportSpec{Name: name, Alias: alias})
				if p.curTokenIs(lexer.TOKEN_COMMA) {
					p.nextToken()
				}
			}
			p.nextToken()
			p.skipSemicolon()
		} else {
			decl = p.parseStatement()
		}
	} else {
		expr := p.parseExpression(PREC_COMMA)
		decl = &ast.ExprStmt{Pos_: p.pos(), Expr: expr}
		p.skipSemicolon()
	}
	return &ast.ExportDecl{Pos_: p.pos(), TokenLit: tokLit, IsDefault: isDefault, Decl: decl, Specifiers: specs}
}

// ============================================================================
// Expression Parsing (Pratt)
// ============================================================================

func (p *Parser) parseExpression(prec int) ast.Expression {
	prefix := p.prefixFns[p.cur.Type]
	if prefix == nil {
		p.addError(fmt.Sprintf("no prefix parser for %s (%q)", p.cur.Type, p.cur.Literal))
		return nil
	}
	start := p.cur
	left := prefix()
	// Simple prefix parsers leave cur on the token they consumed; compound
	// parsers like function/new/array/object/paren already advance to the next
	// token after the full expression.
	if p.cur == start {
		p.nextToken()
	}
	for prec < p.curPrecedence() && !p.curTokenIs(lexer.TOKEN_SEMI) {
		infix := p.infixFns[p.cur.Type]
		if infix == nil {
			return left
		}
		left = infix(left)
	}
	return left
}

// ——— Prefix Parsers ———

func (p *Parser) parseIdent() ast.Expression {
	return &ast.Ident{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseNumber() ast.Expression {
	lit := p.cur.Literal
	val, _ := strconv.ParseFloat(lit, 64)
	isInt := true
	for _, c := range lit {
		if c == '.' || c == 'e' || c == 'E' {
			isInt = false
			break
		}
	}
	return &ast.NumberLit{Pos_: p.pos(), TokenLit: lit, Value: val, IsInt: isInt}
}

func (p *Parser) parseString() ast.Expression {
	return &ast.StringLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseTemplate() ast.Expression {
	return &ast.TemplateLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseBool() ast.Expression {
	return &ast.BoolLit{Pos_: p.pos(), TokenLit: p.cur.Literal, Value: p.curTokenIs(lexer.TOKEN_TRUE)}
}

func (p *Parser) parseNull() ast.Expression {
	return &ast.NullLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseUndefined() ast.Expression {
	return &ast.UndefinedLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseThis() ast.Expression {
	return &ast.ThisExpr{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parseSuper() ast.Expression {
	return &ast.SuperExpr{Pos_: p.pos(), TokenLit: p.cur.Literal}
}

func (p *Parser) parsePrefix() ast.Expression {
	op := p.cur.Literal
	tokLit := p.cur.Literal
	p.nextToken()
	right := p.parseExpression(PREC_PREFIX)
	return &ast.PrefixExpr{Pos_: p.pos(), TokenLit: tokLit, Op: op, Right: right}
}

func (p *Parser) parseAwait() ast.Expression {
	tokLit := p.cur.Literal
	p.nextToken()
	value := p.parseExpression(PREC_PREFIX)
	return &ast.AwaitExpr{Pos_: p.pos(), TokenLit: tokLit, Value: value}
}

func (p *Parser) parseParenOrArrow() ast.Expression {
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken() // )
		if p.curTokenIs(lexer.TOKEN_ARROW) {
			return p.parseArrowLambda(nil)
		}
		return nil
	}
	// Try to parse as arrow parameter list
	savedCur := p.cur
	savedPeek := p.peek
	params := p.parseParamList()
	if params != nil && p.curTokenIs(lexer.TOKEN_ARROW) {
		return p.parseArrowLambda(params)
	}
	// Not arrow, backtrack and parse as parenthesized expression
	p.cur = savedCur
	p.peek = savedPeek
	expr := p.parseExpression(PREC_COMMA)
	p.expectPeek(lexer.TOKEN_RPAREN)
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_ARROW) {
		return p.parseArrowLambda([]*ast.Param{{Name: "_", Spread: false}})
	}
	return expr
}

func (p *Parser) parseArrowLambda(params []*ast.Param) ast.Expression {
	tokLit := "=>"
	p.nextToken() // arrow
	isAsync := false
	var retT *ast.TypeAnnotation
	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		body := p.parseBlock()
		return &ast.ArrowFuncExpr{Pos_: p.pos(), TokenLit: tokLit, Params: params, Body: body, IsAsync: isAsync, ReturnT: retT}
	}
	body := p.parseExpression(PREC_COMMA)
	return &ast.ArrowFuncExpr{Pos_: p.pos(), TokenLit: tokLit, Params: params, Body: body, IsAsync: isAsync, ReturnT: retT}
}

func (p *Parser) parseParamList() []*ast.Param {
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken()
		return []*ast.Param{}
	}
	params := make([]*ast.Param, 0)
	for {
		spread := false
		if p.curTokenIs(lexer.TOKEN_ELLIPSIS) {
			spread = true
			p.nextToken()
		}
		if !p.curTokenIs(lexer.TOKEN_IDENT) {
			return nil // not a parameter, must be expression
		}
		param := p.parseParam(spread)
		if p.curTokenIs(lexer.TOKEN_EQ) {
			p.nextToken()
			param.Default = p.parseExpression(PREC_COMMA)
		}
		params = append(params, param)
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			continue
		}
		if p.curTokenIs(lexer.TOKEN_RPAREN) {
			p.nextToken()
			return params
		}
		return nil
	}
}

func (p *Parser) parseParam(spread bool) *ast.Param {
	param := &ast.Param{Pos_: p.pos(), Name: p.cur.Literal, Spread: spread}
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_QUESTION) {
		param.Optional = true
		p.nextToken()
	}
	if p.curTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		param.TypeAnno = p.parseType()
	} else if param.Optional {
		param.TypeAnno = &ast.TypeAnnotation{Kind: ast.TK_PRIMITIVE, Name: "any", Optional: true}
	}
	if param.Optional && param.TypeAnno != nil {
		param.TypeAnno.Optional = true
	}
	return param
}

func (p *Parser) parseArray() ast.Expression {
	tokLit := p.cur.Literal
	p.nextToken()
	elems := make([]ast.Expression, 0)
	if p.curTokenIs(lexer.TOKEN_RBRACK) {
		p.nextToken()
		return &ast.ArrayLit{Pos_: p.pos(), TokenLit: tokLit, Elements: elems}
	}
	for {
		if p.curTokenIs(lexer.TOKEN_ELLIPSIS) {
			p.nextToken()
			elems = append(elems, &ast.SpreadExpr{Pos_: p.pos(), TokenLit: "...", Value: p.parseExpression(PREC_COMMA)})
		} else {
			elems = append(elems, p.parseExpression(PREC_COMMA))
		}
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			if p.curTokenIs(lexer.TOKEN_RBRACK) {
				break
			}
			continue
		}
		break
	}
	if p.curTokenIs(lexer.TOKEN_RBRACK) {
		p.nextToken()
	}
	return &ast.ArrayLit{Pos_: p.pos(), TokenLit: tokLit, Elements: elems}
}

func (p *Parser) parseObjectOrBlock() ast.Expression {
	tokLit := p.cur.Literal
	p.nextToken() // {
	props := make([]*ast.Property, 0)
	if p.curTokenIs(lexer.TOKEN_RBRACE) {
		p.nextToken() // }
		return &ast.ObjectLit{Pos_: p.pos(), TokenLit: tokLit, Properties: props}
	}
	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		prop := p.parseProperty()
		if prop != nil {
			props = append(props, prop)
		}
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
		}
	}
	if p.curTokenIs(lexer.TOKEN_RBRACE) {
		p.nextToken() // }
	}
	return &ast.ObjectLit{Pos_: p.pos(), TokenLit: tokLit, Properties: props}
}

func (p *Parser) parseProperty() *ast.Property {
	if p.curTokenIs(lexer.TOKEN_ELLIPSIS) {
		p.nextToken()
		return &ast.Property{Spread: true, Value: p.parseExpression(PREC_COMMA)}
	}
	if p.curTokenIs(lexer.TOKEN_LBRACK) {
		p.nextToken()
		key := p.parseExpression(PREC_COMMA)
		if !p.curTokenIs(lexer.TOKEN_RBRACK) {
			p.addError("expected ]")
		}
		p.nextToken() // skip ]
		if !p.curTokenIs(lexer.TOKEN_COLON) {
			p.addError("expected : after computed key")
		}
		p.nextToken() // skip :
		val := p.parseExpression(PREC_COMMA)
		return &ast.Property{Pos_: p.pos(), Key: key, Value: val, Computed: true}
	}

	// Parse key as a simple primary (not full expression)
	var key ast.Expression
	switch p.cur.Type {
	case lexer.TOKEN_IDENT:
		key = &ast.Ident{Pos_: p.pos(), TokenLit: p.cur.Literal}
	case lexer.TOKEN_STRING:
		key = &ast.StringLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
	case lexer.TOKEN_NUMBER:
		key = &ast.NumberLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
	default:
		p.addError("expected property key")
		return nil
	}
	p.nextToken() // advance past key

	// Method shorthand: key(args) { body }
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		if ident, ok := key.(*ast.Ident); ok {
			params, _ := p.parseFuncParams()
			body := p.parseBlock()
			return &ast.Property{
				Key:   ident,
				Value: &ast.FuncExpr{Pos_: p.pos(), TokenLit: "function", Name: ident.TokenLit, Params: params, Body: body},
			}
		}
	}

	// Key-value: key: value
	if p.curTokenIs(lexer.TOKEN_COLON) {
		p.nextToken()
		return &ast.Property{Pos_: p.pos(), Key: key, Value: p.parseExpression(PREC_COMMA)}
	}

	// Shorthand: { key } or { key, }
	return &ast.Property{Pos_: p.pos(), Key: key, Value: key, Shorthand: true}
}

// parseMethod is no longer needed as it's inlined.
// Remove the standalone function. (Will check compilation)

func (p *Parser) parseNew() ast.Expression {
	tokLit := p.cur.Literal
	p.nextToken()
	callee := p.parseExpression(PREC_CALL)
	var args []ast.Expression
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.nextToken()
		args = p.parseCallArgs()
		if !p.curTokenIs(lexer.TOKEN_RPAREN) {
			p.addError("expected ) after new args")
		} else {
			p.nextToken()
		}
	}
	return &ast.NewExpr{Pos_: p.pos(), TokenLit: tokLit, Callee: callee, Args: args}
}

func (p *Parser) parseIndex(left ast.Expression) ast.Expression {
	p.nextToken()
	idx := p.parseExpression(PREC_COMMA)
	if !p.curTokenIs(lexer.TOKEN_RBRACK) {
		p.addError("expected ]")
	} else {
		p.nextToken()
	}
	return &ast.IndexExpr{Pos_: p.pos(), TokenLit: "[]", Left: left, Index: idx}
}

func (p *Parser) parseFunction() ast.Expression {
	tokLit := p.cur.Literal
	name := ""
	if p.peekTokenIs(lexer.TOKEN_IDENT) {
		p.nextToken()
		name = p.cur.Literal
	}
	p.nextToken()
	params, retT := p.parseFuncParams()
	body := p.parseBlock()
	return &ast.FuncExpr{Pos_: p.pos(), TokenLit: tokLit, Name: name, Params: params, ReturnT: retT, Body: body}
}

func (p *Parser) parseAsyncFunc() ast.Expression {
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_FUNCTION) {
		fn := p.parseFunction()
		fn.(*ast.FuncExpr).IsAsync = true
		return fn
	}
	// async (params) => body
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		return p.parseParenOrArrow()
	}
	return p.parseIdent()
}

func (p *Parser) parseClassExpr() ast.Expression {
	decl := p.parseClassDecl()
	return &ast.ClassDecl{Pos_: p.pos(), TokenLit: decl.TokenLit, Name: decl.Name, Super: decl.Super, Body: decl.Body}
}

// ——— Infix Parsers ———

func (p *Parser) parseInfix(left ast.Expression) ast.Expression {
	op := p.cur.Literal
	prec := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(prec)
	return &ast.InfixExpr{Pos_: p.pos(), TokenLit: op, Op: op, Left: left, Right: right}
}

func (p *Parser) parseAssign(left ast.Expression) ast.Expression {
	op := p.cur.Literal
	p.nextToken()
	right := p.parseExpression(PREC_ASSIGN - 1)
	return &ast.AssignExpr{Pos_: p.pos(), TokenLit: op, Op: op, Left: left, Right: right}
}

func (p *Parser) parseTernary(left ast.Expression) ast.Expression {
	p.nextToken() // skip ?
	cons := p.parseExpression(PREC_COMMA)
	if !p.curTokenIs(lexer.TOKEN_COLON) {
		p.addError("expected : in ternary")
		return left
	}
	p.nextToken() // skip :
	alt := p.parseExpression(PREC_COMMA)
	return &ast.TernaryExpr{Pos_: p.pos(), TokenLit: "?:", Cond: left, Consequent: cons, Alternate: alt}
}

func (p *Parser) parseCall(callee ast.Expression) ast.Expression {
	p.nextToken() // skip (
	args := p.parseCallArgs()
	if !p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.addError("expected )")
	} else {
		p.nextToken() // skip )
	}
	return &ast.CallExpr{Pos_: p.pos(), TokenLit: "()", Callee: callee, Args: args}
}

func (p *Parser) parseCallArgs() []ast.Expression {
	args := make([]ast.Expression, 0)
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		return args
	}
	for {
		if p.curTokenIs(lexer.TOKEN_ELLIPSIS) {
			p.nextToken()
			args = append(args, &ast.SpreadExpr{Pos_: p.pos(), TokenLit: "...", Value: p.parseExpression(PREC_COMMA)})
		} else {
			args = append(args, p.parseExpression(PREC_COMMA))
		}
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	return args
}

func (p *Parser) parseMember(left ast.Expression) ast.Expression {
	p.nextToken()
	prop := &ast.Ident{Pos_: p.pos(), TokenLit: p.cur.Literal}
	p.nextToken()
	return &ast.MemberExpr{Pos_: p.pos(), TokenLit: ".", Object: left, Property: prop, Computed: false}
}

func (p *Parser) parseOptional(left ast.Expression) ast.Expression {
	p.nextToken() // skip ?.
	if p.curTokenIs(lexer.TOKEN_LBRACK) {
		p.nextToken()
		idx := p.parseExpression(PREC_COMMA)
		if p.curTokenIs(lexer.TOKEN_RBRACK) {
			p.nextToken()
		}
		return &ast.OptionalExpr{Pos_: p.pos(), TokenLit: "?.[]", Object: left, Property: idx, Computed: true}
	}
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.nextToken()
		args := p.parseCallArgs()
		if p.curTokenIs(lexer.TOKEN_RPAREN) {
			p.nextToken()
		}
		return &ast.OptionalExpr{Pos_: p.pos(), TokenLit: "?.()", Object: left, IsCall: true, Args: args}
	}
	prop := &ast.Ident{Pos_: p.pos(), TokenLit: p.cur.Literal}
	p.nextToken()
	return &ast.OptionalExpr{Pos_: p.pos(), TokenLit: "?.", Object: left, Property: prop}
}

func (p *Parser) parseArrowInfix(left ast.Expression) ast.Expression {
	ident, ok := left.(*ast.Ident)
	if !ok {
		p.nextToken() // skip stray =>
		return left
	}
	params := []*ast.Param{{Pos_: ident.Pos_, Name: ident.TokenLit}}
	p.nextToken() // skip =>
	var body ast.Node
	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		body = p.parseBlock()
	} else {
		body = p.parseExpression(PREC_COMMA)
	}
	return &ast.ArrowFuncExpr{Pos_: p.pos(), TokenLit: "=>", Params: params, Body: body}
}

func (p *Parser) parsePostfix(left ast.Expression) ast.Expression {
	op := p.cur.Literal
	p.nextToken()
	return &ast.InfixExpr{Pos_: p.pos(), TokenLit: op, Op: op, Left: left, Right: nil}
}

func (p *Parser) parseMatch() ast.Expression {
	start := p.pos()
	tokLit := p.cur.Literal
	p.nextToken() // skip match
	expr := p.parseExpression(PREC_COMMA)
	if !p.curTokenIs(lexer.TOKEN_LBRACE) {
		p.addError("expected { after match subject")
		return nil
	}
	p.nextToken() // {
	arms := make([]*ast.MatchArm, 0)
	for !p.curTokenIs(lexer.TOKEN_RBRACE) && !p.curTokenIs(lexer.TOKEN_EOF) {
		arm := p.parseMatchArm()
		if arm != nil {
			arms = append(arms, arm)
		}
		if p.curTokenIs(lexer.TOKEN_COMMA) {
			p.nextToken()
			continue
		}
		if p.curTokenIs(lexer.TOKEN_RBRACE) {
			break
		}
		// Recovery: skip to next pattern or }
		p.nextToken()
	}
	p.nextToken() // }
	return &ast.MatchExpr{Pos_: start, TokenLit: tokLit, Expr: expr, Arms: arms}
}

func (p *Parser) parseMatchArm() *ast.MatchArm {
	pat := p.parsePattern()
	var guard ast.Expression
	if p.curTokenIs(lexer.TOKEN_IF) {
		p.nextToken()
		guard = p.parseExpression(PREC_COMMA)
	}
	if !p.curTokenIs(lexer.TOKEN_ARROW) {
		p.addError("expected => in match arm")
		return nil
	}
	p.nextToken()
	var body ast.Node
	if p.curTokenIs(lexer.TOKEN_LBRACE) {
		body = p.parseBlock()
	} else {
		body = p.parseExpression(PREC_COMMA)
	}
	return &ast.MatchArm{Pos_: p.pos(), Pattern: pat, Guard: guard, Body: body}
}

func (p *Parser) parsePattern() ast.Pattern {
	// Parse first primary pattern
	primary := p.parsePrimaryPattern()
	if primary == nil {
		return nil
	}
	// After primary parse, cur is the token after the primary pattern

	// OR pattern: primary | primary | ...
	if p.curTokenIs(lexer.TOKEN_PIPE) {
		alts := []ast.Pattern{primary}
		for p.curTokenIs(lexer.TOKEN_PIPE) {
			p.nextToken()
			alt := p.parsePrimaryPattern()
			if alt != nil {
				alts = append(alts, alt)
			}
		}
		return &ast.OrPattern{Pos_: p.pos(), TokenLit: "|", Alternatives: alts}
	}

	return primary
}

func (p *Parser) parsePrimaryPattern() ast.Pattern {
	switch p.cur.Type {
	case lexer.TOKEN_NUMBER:
		tok := p.cur.Literal
		num := &ast.NumberLit{Pos_: p.pos(), TokenLit: tok, Value: parseFloat(tok)}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_DOT_DOT) || p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ) {
			inclusive := p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ)
			p.nextToken()
			end := p.parseLiteralExpr()
			return &ast.RangePattern{Pos_: num.Pos_, TokenLit: "..", Start: num, End: end, Inclusive: inclusive}
		}
		if p.curTokenIs(lexer.TOKEN_PIPE) {
			return p.parseOrPatternContinue(&ast.LiteralPattern{Pos_: num.Pos_, TokenLit: tok, Value: num})
		}
		return &ast.LiteralPattern{Pos_: num.Pos_, TokenLit: tok, Value: num}
	case lexer.TOKEN_STRING:
		tok := p.cur.Literal
		str := &ast.StringLit{Pos_: p.pos(), TokenLit: tok}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_PIPE) {
			return p.parseOrPatternContinue(&ast.LiteralPattern{Pos_: str.Pos_, TokenLit: tok, Value: str})
		}
		return &ast.LiteralPattern{Pos_: str.Pos_, TokenLit: tok, Value: str}
	case lexer.TOKEN_TRUE, lexer.TOKEN_FALSE:
		tok := p.cur.Literal
		b := &ast.BoolLit{Pos_: p.pos(), TokenLit: tok, Value: p.curTokenIs(lexer.TOKEN_TRUE)}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_PIPE) {
			return p.parseOrPatternContinue(&ast.LiteralPattern{Pos_: b.Pos_, TokenLit: tok, Value: b})
		}
		return &ast.LiteralPattern{Pos_: b.Pos_, TokenLit: tok, Value: b}
	case lexer.TOKEN_NULL:
		tok := p.cur.Literal
		n := &ast.NullLit{Pos_: p.pos(), TokenLit: tok}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_PIPE) {
			return p.parseOrPatternContinue(&ast.LiteralPattern{Pos_: n.Pos_, TokenLit: tok, Value: n})
		}
		return &ast.LiteralPattern{Pos_: n.Pos_, TokenLit: tok, Value: n}
	case lexer.TOKEN_UNDEFINED:
		tok := p.cur.Literal
		u := &ast.UndefinedLit{Pos_: p.pos(), TokenLit: tok}
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_PIPE) {
			return p.parseOrPatternContinue(&ast.LiteralPattern{Pos_: u.Pos_, TokenLit: tok, Value: u})
		}
		return &ast.LiteralPattern{Pos_: u.Pos_, TokenLit: tok, Value: u}
	case lexer.TOKEN_IDENT:
		if p.cur.Literal == "_" {
			tokLit := p.cur.Literal
			p.nextToken()
			if p.curTokenIs(lexer.TOKEN_DOT_DOT) || p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ) {
				inclusive := p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ)
				p.nextToken()
				end := p.parseExpression(PREC_COMMA)
				return &ast.RangePattern{Pos_: p.pos(), TokenLit: "..", Start: &ast.Ident{Pos_: p.pos(), TokenLit: tokLit}, End: end, Inclusive: inclusive}
			}
			return &ast.WildcardPattern{Pos_: p.pos(), TokenLit: tokLit}
		}
		name := p.cur.Literal
		p.nextToken()
		if p.curTokenIs(lexer.TOKEN_DOT_DOT) || p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ) {
			inclusive := p.curTokenIs(lexer.TOKEN_DOT_DOT_EQ)
			p.nextToken()
			end := p.parseExpression(PREC_COMMA)
			return &ast.RangePattern{Pos_: p.pos(), TokenLit: "..", Start: &ast.Ident{Pos_: p.pos(), TokenLit: name}, End: end, Inclusive: inclusive}
		}
		return &ast.IdentPattern{Pos_: p.pos(), TokenLit: name, Name: name}
	default:
		p.addError("unexpected token in pattern: " + string(p.cur.Type))
		return nil
	}
}

func (p *Parser) parseOrPatternContinue(first ast.Pattern) ast.Pattern {
	alts := []ast.Pattern{first}
	for p.curTokenIs(lexer.TOKEN_PIPE) {
		p.nextToken()
		next := p.parsePrimaryPattern()
		if next != nil {
			alts = append(alts, next)
		}
	}
	if len(alts) == 1 {
		return alts[0]
	}
	return &ast.OrPattern{Pos_: p.pos(), TokenLit: "|", Alternatives: alts}
}

// parseLiteralExpr parses a single literal token without using the Pratt pipeline.
func (p *Parser) parseLiteralExpr() ast.Expression {
	switch p.cur.Type {
	case lexer.TOKEN_NUMBER:
		tok := p.cur.Literal
		p.nextToken()
		return &ast.NumberLit{Pos_: p.pos(), TokenLit: tok, Value: parseFloat(tok)}
	case lexer.TOKEN_STRING:
		tok := p.cur.Literal
		p.nextToken()
		return &ast.StringLit{Pos_: p.pos(), TokenLit: tok}
	default:
		return p.parseExpression(PREC_COMMA)
	}
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ============================================================================
// Type Annotation Parsing
// ============================================================================

func (p *Parser) parseType() *ast.TypeAnnotation {
	t := p.parsePrimaryType()
	if p.curTokenIs(lexer.TOKEN_PIPE) {
		union := &ast.TypeAnnotation{Kind: ast.TK_UNION, Union: []*ast.TypeAnnotation{t}}
		for p.curTokenIs(lexer.TOKEN_PIPE) {
			p.nextToken()
			union.Union = append(union.Union, p.parsePrimaryType())
		}
		t = union
	}
	if p.curTokenIs(lexer.TOKEN_QUESTION) {
		t.Optional = true
		p.nextToken()
	}
	return t
}

func (p *Parser) parsePrimaryType() *ast.TypeAnnotation {
	t := &ast.TypeAnnotation{Kind: ast.TK_PRIMITIVE, Name: p.cur.Literal}
	p.nextToken()
	if p.curTokenIs(lexer.TOKEN_LBRACK) && p.peekTokenIs(lexer.TOKEN_RBRACK) {
		t = &ast.TypeAnnotation{Kind: ast.TK_ARRAY, ArrayOf: t}
		p.nextToken()
		p.nextToken()
		return t
	}
	if p.curTokenIs(lexer.TOKEN_QUESTION) {
		t.Optional = true
		p.nextToken()
		return t
	}
	return t
}
