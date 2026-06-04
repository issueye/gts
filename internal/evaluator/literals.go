package evaluator

import (
	"strconv"
	"strings"
	"unicode/utf8"

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
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'v':
				b.WriteByte('\v')
			case '0':
				b.WriteByte(0)
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '\'':
				b.WriteByte('\'')
			case 'x':
				if i+3 < len(s) && isHexByte(s[i+2]) && isHexByte(s[i+3]) {
					value, _ := strconv.ParseInt(s[i+2:i+4], 16, 32)
					b.WriteByte(byte(value))
					i += 4
					continue
				}
				b.WriteByte('x')
			case 'u':
				if i+5 < len(s) && isHexByte(s[i+2]) && isHexByte(s[i+3]) && isHexByte(s[i+4]) && isHexByte(s[i+5]) {
					value, _ := strconv.ParseInt(s[i+2:i+6], 16, 32)
					b.WriteRune(rune(value))
					i += 6
					continue
				}
				if i+3 < len(s) && s[i+2] == '{' {
					if end := strings.IndexByte(s[i+3:], '}'); end >= 0 {
						hex := s[i+3 : i+3+end]
						if hex != "" && allHex(hex) {
							value, _ := strconv.ParseInt(hex, 16, 32)
							r := rune(value)
							if utf8.ValidRune(r) {
								b.WriteRune(r)
								i += 4 + end
								continue
							}
						}
					}
				}
				b.WriteByte('u')
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

func isHexByte(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func allHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isHexByte(s[i]) {
			return false
		}
	}
	return true
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
		start := i
		for i < len(inner) && !(i+1 < len(inner) && inner[i] == '$' && inner[i+1] == '{') {
			i++
		}
		result.WriteString(unescapeString(inner[start:i]))
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

func evalRegExpLit(n *ast.RegExpLit) object.Object {
	source, flags, ok := splitRegExpLiteral(n.TokenLit)
	if !ok {
		return object.NewError(n.Pos(), "SyntaxError: invalid regexp literal")
	}
	re, err := compileRegExp(n.Pos(), source, flags)
	if err != nil {
		return err
	}
	return re
}

func splitRegExpLiteral(lit string) (string, string, bool) {
	if len(lit) < 2 || lit[0] != '/' {
		return "", "", false
	}
	inClass := false
	escape := false
	for i := 1; i < len(lit); i++ {
		ch := lit[i]
		if escape {
			escape = false
			continue
		}
		if ch == '\\' {
			escape = true
			continue
		}
		if inClass {
			if ch == ']' {
				inClass = false
			}
			continue
		}
		if ch == '[' {
			inClass = true
			continue
		}
		if ch == '/' {
			return lit[1:i], lit[i+1:], true
		}
	}
	return "", "", false
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
