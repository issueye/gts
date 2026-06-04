package parser

import (
	"fmt"
	"strconv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
)

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

func (p *Parser) parseRegExp() ast.Expression {
	return &ast.RegExpLit{Pos_: p.pos(), TokenLit: p.cur.Literal}
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
	mark := p.mark()
	params := p.parseParamList()
	if params != nil && p.curTokenIs(lexer.TOKEN_ARROW) {
		p.commit(mark)
		return p.parseArrowLambda(params)
	}
	// Not arrow, backtrack and parse as parenthesized expression
	p.rewind(mark)
	expr := p.parseExpression(PREC_COMMA)
	if p.curTokenIs(lexer.TOKEN_RPAREN) {
		p.nextToken()
	} else if p.expectPeek(lexer.TOKEN_RPAREN) {
		p.nextToken()
	}
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
	decl := p.parseClassExprDecl()
	if decl == nil {
		return nil
	}
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
