package parser

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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
	if p.curTokenIs(lexer.TOKEN_IDENT) && !p.peekTokenIs(lexer.TOKEN_LBRACE) {
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
