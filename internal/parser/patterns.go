package parser

import (
	"strconv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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
	var bindingName string
	var bindingPos ast.Position
	if p.curTokenIs(lexer.TOKEN_LPAREN) {
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_IDENT) {
			p.addError("expected identifier in match arm binding")
			return nil
		}
		bindingName = p.cur.Literal
		bindingPos = p.pos()
		p.nextToken()
		if !p.curTokenIs(lexer.TOKEN_RPAREN) {
			p.addError("expected ) after match arm binding")
			return nil
		}
		p.nextToken()
	}
	var guard ast.Expression
	if p.curTokenIs(lexer.TOKEN_IF) {
		p.nextToken()
		guard = p.parseExpression(PREC_ASSIGN)
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
	return &ast.MatchArm{Pos_: p.pos(), Pattern: pat, BindingName: bindingName, BindingPos: bindingPos, Guard: guard, Body: body}
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
