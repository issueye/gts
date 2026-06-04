package evaluator

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

// ============================================================================
// Identifiers
// ============================================================================

func evalIdent(n *ast.Ident, env *object.Environment) object.Object {
	val, ok := env.Get(n.TokenLit)
	if ok {
		return val
	}
	return object.NewError(n.Pos(), "ReferenceError: '%s' is not defined", n.TokenLit)
}

// ============================================================================
// Literals
// ============================================================================

func evalStringLit(n *ast.StringLit) object.Object {
	lit := n.TokenLit
	if len(lit) < 2 {
		return &object.String{Value: ""}
	}
	inner := lit[1 : len(lit)-1]
	inner = unescapeString(inner)
	return &object.String{Value: inner}
}

func unescapeString(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '\'':
				b.WriteByte('\'')
			default:
				b.WriteByte(s[i+1])
			}
			i += 2
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func evalTemplate(n *ast.TemplateLit, env *object.Environment) object.Object {
	lit := n.TokenLit
	if len(lit) < 2 || lit[0] != '`' {
		return &object.String{Value: lit}
	}
	inner := lit[1:]
	if strings.HasSuffix(inner, "`") {
		inner = inner[:len(inner)-1]
	}
	var result strings.Builder
	i := 0
	for i < len(inner) {
		if i+1 < len(inner) && inner[i] == '$' && inner[i+1] == '{' {
			end := findTemplateExprEnd(inner, i+2)
			if end < 0 {
				return object.NewError(n.Pos(), "SyntaxError: unterminated template expression")
			}
			exprStr := strings.TrimSpace(inner[i+2 : end])
			if exprStr != "" {
				val := evalTemplateExpression(exprStr, env, n.Pos())
				if object.IsRuntimeError(val) {
					return val
				}
				result.WriteString(val.Inspect())
			}
			i = end + 1
			continue
		}
		result.WriteByte(inner[i])
		i++
	}
	return &object.String{Value: result.String()}
}

func findTemplateExprEnd(input string, start int) int {
	depth := 0
	quote := byte(0)
	escape := false
	for i := start; i < len(input); i++ {
		ch := input[i]
		if quote != 0 {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func evalTemplateExpression(expr string, env *object.Environment, pos ast.Position) object.Object {
	if strings.HasSuffix(expr, "`") {
		expr = strings.TrimSuffix(expr, "`")
	}
	const resultName = "__gts_template_expr"
	l := lexer.New("let " + resultName + " = " + expr + ";")
	p := parser.New(l, pos.File)
	prog := p.ParseProgram()
	if len(l.Errors()) > 0 {
		return object.NewError(pos, "SyntaxError: %s", strings.Join(l.Errors(), "\n"))
	}
	if len(prog.Errors) > 0 {
		return object.NewError(pos, "SyntaxError: %s", strings.Join(prog.Errors, "\n"))
	}
	scope := env.NewScope()
	result := Eval(prog, scope)
	if object.IsRuntimeError(result) {
		return result
	}
	value, ok := scope.Get(resultName)
	if !ok {
		return object.UNDEFINED
	}
	return value
}

func evalArray(n *ast.ArrayLit, env *object.Environment) object.Object {
	elems := make([]object.Object, len(n.Elements))
	for i, e := range n.Elements {
		val := Eval(e, env)
		if object.IsRuntimeError(val) {
			return val
		}
		elems[i] = val
	}
	return env.ObjectManager().NewArrayAt(elems, n.Pos())
}

func evalObject(n *ast.ObjectLit, env *object.Environment) object.Object {
	hash := env.ObjectManager().NewHashAt(n.Pos())
	for _, p := range n.Properties {
		if p.Spread {
			val := Eval(p.Value, env)
			if h, ok := val.(*object.Hash); ok {
				for _, pair := range h.OrderedPairs() {
					hash.SetMember(pair.Key, pair.Value)
				}
			}
			continue
		}
		if p.Shorthand {
			name := p.Key.(*ast.Ident).TokenLit
			key := &object.String{Value: name}
			val := Eval(p.Value, env)
			if object.IsRuntimeError(val) {
				return val
			}
			hash.SetMember(key, val)
			continue
		}
		key := evalPropertyKey(p.Key, env)
		if object.IsRuntimeError(key) {
			return key
		}
		val := Eval(p.Value, env)
		if object.IsRuntimeError(val) {
			return val
		}
		hash.SetMember(key, val)
	}
	return hash
}
