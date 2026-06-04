package parser

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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
