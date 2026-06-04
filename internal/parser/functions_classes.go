package parser

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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

func (p *Parser) parseClassExprDecl() *ast.ClassDecl {
	tokLit := p.cur.Literal
	p.nextToken() // class
	name := ""
	if p.curTokenIs(lexer.TOKEN_IDENT) {
		name = p.cur.Literal
		p.nextToken()
	}

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
