package parser

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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
	case lexer.TOKEN_RBRACE:
		p.addError("unexpected }")
		p.nextToken()
		return nil
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
