package evaluator

import (
	"fmt"
	"math"

	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch n := node.(type) {
	case *ast.Program:
		return evalProgram(n, env)

	// Statements
	case *ast.LetStmt:
		return evalLet(n, env)
	case *ast.ConstStmt:
		return evalConst(n, env)
	case *ast.VarStmt:
		return evalVar(n, env)
	case *ast.BlockStmt:
		return evalBlock(n, env.NewScope())
	case *ast.IfStmt:
		return evalIf(n, env)
	case *ast.WhileStmt:
		return evalWhile(n, env)
	case *ast.ForStmt:
		return evalFor(n, env)
	case *ast.ForInStmt:
		return evalForIn(n, env)
	case *ast.ForOfStmt:
		return evalForOf(n, env)
	case *ast.ReturnStmt:
		return evalReturn(n, env)
	case *ast.BreakStmt:
		return &object.ReturnValue{Value: &object.Error{Message: "break", Pos: n.Pos()}}
	case *ast.ContinueStmt:
		return &object.ReturnValue{Value: &object.Error{Message: "continue", Pos: n.Pos()}}
	case *ast.ThrowStmt:
		val := Eval(n.Value, env)
		if object.IsError(val) {
			return val
		}
		return object.NewError(n.Pos(), "%s", val.Inspect())
	case *ast.TryStmt:
		return evalTry(n, env)
	case *ast.ExprStmt:
		return Eval(n.Expr, env)
	case *ast.LabeledStmt:
		return evalLabeled(n, env)

	// Declarations
	case *ast.FuncDecl:
		return evalFuncDecl(n, env)
	case *ast.ClassDecl:
		return evalClassDecl(n, env)

	// Expressions
	case *ast.Ident:
		return evalIdent(n, env)
	case *ast.NumberLit:
		return &object.Number{Value: n.Value}
	case *ast.StringLit:
		return evalStringLit(n)
	case *ast.TemplateLit:
		return evalTemplate(n, env)
	case *ast.BoolLit:
		return object.NativeBool(n.Value)
	case *ast.NullLit:
		return object.NULL
	case *ast.UndefinedLit:
		return object.UNDEFINED
	case *ast.ThisExpr:
		t, _ := env.Get("this")
		return t
	case *ast.SuperExpr:
		return evalSuper(n, env)
	case *ast.ArrayLit:
		return evalArray(n, env)
	case *ast.ObjectLit:
		return evalObject(n, env)
	case *ast.PrefixExpr:
		return evalPrefix(n, env)
	case *ast.InfixExpr:
		return evalInfix(n, env)
	case *ast.TernaryExpr:
		return evalTernary(n, env)
	case *ast.AssignExpr:
		return evalAssign(n, env)
	case *ast.CallExpr:
		return evalCall(n, env)
	case *ast.MemberExpr:
		return evalMember(n, env)
	case *ast.IndexExpr:
		return evalIndex(n, env)
	case *ast.OptionalExpr:
		return evalOptional(n, env)
	case *ast.FuncExpr:
		return evalFuncExpr(n, env)
	case *ast.ArrowFuncExpr:
		return evalArrowFunc(n, env)
	case *ast.NewExpr:
		return evalNew(n, env)
	case *ast.AwaitExpr:
		return evalAwait(n, env)
	case *ast.MatchExpr:
		return evalMatch(n, env)

	// Import / Export (stub)
	case *ast.ImportDecl, *ast.ExportDecl:
		return object.UNDEFINED
	}

	return object.NewError(ast.Position{}, "unknown node type: %T", node)
}

// ============================================================================
// Program
// ============================================================================

func evalProgram(prog *ast.Program, env *object.Environment) object.Object {
	var result object.Object
	for _, stmt := range prog.Body {
		result = Eval(stmt, env)
		if rv, ok := result.(*object.ReturnValue); ok {
			return rv.Value
		}
		if object.IsError(result) {
			return result
		}
	}
	if result == nil {
		return object.UNDEFINED
	}
	return result
}

// ============================================================================
// Variable Declarations
// ============================================================================

func evalLet(n *ast.LetStmt, env *object.Environment) object.Object {
	var val object.Object = object.UNDEFINED
	if n.Value != nil {
		val = Eval(n.Value, env)
		if object.IsError(val) {
			return val
		}
	}
	env.Set(n.Name, val)
	return object.UNDEFINED
}

func evalConst(n *ast.ConstStmt, env *object.Environment) object.Object {
	var val object.Object = object.UNDEFINED
	if n.Value != nil {
		val = Eval(n.Value, env)
		if object.IsError(val) {
			return val
		}
	}
	env.Set(n.Name, val)
	return object.UNDEFINED
}

func evalVar(n *ast.VarStmt, env *object.Environment) object.Object {
	var val object.Object = object.UNDEFINED
	if n.Value != nil {
		val = Eval(n.Value, env)
		if object.IsError(val) {
			return val
		}
	}
	env.Set(n.Name, val)
	return object.UNDEFINED
}

// ============================================================================
// Block
// ============================================================================

func evalBlock(block *ast.BlockStmt, env *object.Environment) object.Object {
	var result object.Object
	for _, stmt := range block.Statements {
		result = Eval(stmt, env)
		if result != nil && result.Type() == object.RETURN_OBJ {
			return result
		}
		if object.IsError(result) {
			return result
		}
	}
	return result
}

// ============================================================================
// If / While / For
// ============================================================================

func evalIf(n *ast.IfStmt, env *object.Environment) object.Object {
	cond := Eval(n.Cond, env)
	if object.IsError(cond) {
		return cond
	}
	if object.IsTruthy(cond) {
		return Eval(n.Consequence, env.NewScope())
	}
	if n.Alternative != nil {
		return Eval(n.Alternative, env.NewScope())
	}
	return object.UNDEFINED
}

func evalWhile(n *ast.WhileStmt, env *object.Environment) object.Object {
	var result object.Object
	for {
		cond := Eval(n.Cond, env)
		if object.IsError(cond) {
			return cond
		}
		if !object.IsTruthy(cond) {
			break
		}
		result = Eval(n.Body, env.NewScope())
		if rv, ok := result.(*object.ReturnValue); ok {
			return rv
		}
		if object.IsError(result) {
			return result
		}
	}
	return object.UNDEFINED
}

func evalFor(n *ast.ForStmt, env *object.Environment) object.Object {
	scope := env.NewScope()
	if n.Init != nil {
		result := Eval(n.Init, scope)
		if object.IsError(result) {
			return result
		}
	}
	for {
		if n.Cond != nil {
			cond := Eval(n.Cond, scope)
			if object.IsError(cond) {
				return cond
			}
			if !object.IsTruthy(cond) {
				break
			}
		}
		result := Eval(n.Body, scope.NewScope())
		if rv, ok := result.(*object.ReturnValue); ok {
			return rv
		}
		if object.IsError(result) {
			return result
		}
		if n.Post != nil {
			Eval(n.Post, scope)
		}
	}
	return object.UNDEFINED
}

func evalForIn(n *ast.ForInStmt, env *object.Environment) object.Object {
	iterable := Eval(n.Iterable, env)
	if object.IsError(iterable) {
		return iterable
	}
	switch it := iterable.(type) {
	case *object.Array:
		scope := env.NewScope()
		for i := 0; i < len(it.Elements); i++ {
			scope.Set(n.Name, &object.String{Value: fmt.Sprintf("%d", i)})
			result := Eval(n.Body, scope.NewScope())
			if rv, ok := result.(*object.ReturnValue); ok {
				return rv
			}
			if object.IsError(result) {
				return result
			}
		}
	case *object.Hash:
		scope := env.NewScope()
		for _, pair := range it.Pairs {
			scope.Set(n.Name, pair.Key)
			result := Eval(n.Body, scope.NewScope())
			if rv, ok := result.(*object.ReturnValue); ok {
				return rv
			}
			if object.IsError(result) {
				return result
			}
		}
	case *object.String:
		scope := env.NewScope()
		for i := 0; i < len(it.Value); i++ {
			scope.Set(n.Name, &object.Number{Value: float64(i)})
			result := Eval(n.Body, scope.NewScope())
			if rv, ok := result.(*object.ReturnValue); ok {
				return rv
			}
			if object.IsError(result) {
				return result
			}
		}
	default:
		return object.NewError(n.Pos(), "cannot iterate over %s", it.Type())
	}
	return object.UNDEFINED
}

func evalForOf(n *ast.ForOfStmt, env *object.Environment) object.Object {
	iterable := Eval(n.Iterable, env)
	if object.IsError(iterable) {
		return iterable
	}
	switch it := iterable.(type) {
	case *object.Array:
		scope := env.NewScope()
		for _, elem := range it.Elements {
			scope.Set(n.Name, elem)
			result := Eval(n.Body, scope.NewScope())
			if rv, ok := result.(*object.ReturnValue); ok {
				return rv
			}
			if object.IsError(result) {
				return result
			}
		}
	case *object.String:
		scope := env.NewScope()
		for _, ch := range it.Value {
			scope.Set(n.Name, &object.String{Value: string(ch)})
			result := Eval(n.Body, scope.NewScope())
			if rv, ok := result.(*object.ReturnValue); ok {
				return rv
			}
			if object.IsError(result) {
				return result
			}
		}
	default:
		return object.NewError(n.Pos(), "cannot for-of over %s", it.Type())
	}
	return object.UNDEFINED
}

// ============================================================================
// Return / Labeled
// ============================================================================

func evalReturn(n *ast.ReturnStmt, env *object.Environment) object.Object {
	if n.Value == nil {
		return &object.ReturnValue{Value: object.UNDEFINED}
	}
	val := Eval(n.Value, env)
	if object.IsError(val) {
		return val
	}
	return &object.ReturnValue{Value: val}
}

func evalLabeled(n *ast.LabeledStmt, env *object.Environment) object.Object {
	result := Eval(n.Stmt, env)
	if rv, ok := result.(*object.ReturnValue); ok {
		if err, ok2 := rv.Value.(*object.Error); ok2 {
			if err.Message == "break" || err.Message == "continue" {
				return rv
			}
		}
	}
	return result
}

// ============================================================================
// Try / Catch / Finally
// ============================================================================

func evalTry(n *ast.TryStmt, env *object.Environment) object.Object {
	result := Eval(n.Block, env.NewScope())
	if object.IsError(result) && n.Catch != nil {
		scope := env.NewScope()
		if n.Catch.Name != "" {
			scope.Set(n.Catch.Name, result)
		}
		result = Eval(n.Catch.Body, scope)
	}
	if n.Finalizer != nil {
		Eval(n.Finalizer, env.NewScope())
	}
	return result
}

// ============================================================================
// Function / Class Declaration
// ============================================================================

func evalFuncDecl(n *ast.FuncDecl, env *object.Environment) object.Object {
	fn := &object.Function{
		Name:       n.Name,
		Parameters: n.Params,
		Body:       n.Body,
		Env:        env,
		IsAsync:    n.IsAsync,
		Pos:        n.Pos(),
	}
	env.Set(n.Name, fn)
	return fn
}

func evalClassDecl(n *ast.ClassDecl, env *object.Environment) object.Object {
	cls := &object.Class{
		Name:    n.Name,
		Methods: make(map[string]*object.Function),
		Fields:  make(map[string]object.Object),
		Pos:     n.Pos(),
	}
	for _, m := range n.Body.Members {
		switch m.Kind {
		case "constructor", "method":
			var methParams []*ast.Param
			var methBody *ast.BlockStmt
			if m.Body != nil {
				methParams = m.Params
				methBody = m.Body
			}
			fn := &object.Function{
				Name:       m.Name,
				Parameters: methParams,
				Body:       methBody,
				Env:        env,
				Pos:        m.Pos_,
			}
			cls.Methods[m.Name] = fn
		case "field":
		var val object.Object = object.UNDEFINED
			if m.DefaultVal != nil {
				val = Eval(m.DefaultVal, env)
			}
			cls.Fields[m.Name] = val
		}
	}
	env.Set(n.Name, cls)
	return cls
}

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
	inner := lit[1 : len(lit)-1]
	var result strings.Builder
	i := 0
	for i < len(inner) {
		if i+1 < len(inner) && inner[i] == '$' && inner[i+1] == '{' {
			end := strings.IndexByte(inner[i+2:], '}')
			if end >= 0 {
				exprStr := inner[i+2 : i+2+end]
				_ = exprStr // Template expression evaluation skipped for v0.1
				i += 2 + end + 1
				continue
			}
		}
		result.WriteByte(inner[i])
		i++
	}
	return &object.String{Value: result.String()}
}

func evalArray(n *ast.ArrayLit, env *object.Environment) object.Object {
	elems := make([]object.Object, len(n.Elements))
	for i, e := range n.Elements {
		val := Eval(e, env)
		if object.IsError(val) {
			return val
		}
		elems[i] = val
	}
	return &object.Array{Elements: elems, Pos: n.Pos()}
}

func evalObject(n *ast.ObjectLit, env *object.Environment) object.Object {
	hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair), Pos: n.Pos()}
	for _, p := range n.Properties {
		if p.Spread {
			val := Eval(p.Value, env)
			if h, ok := val.(*object.Hash); ok {
				for k, v := range h.Pairs {
					hash.Pairs[k] = v
				}
			}
			continue
		}
		if p.Shorthand {
			name := p.Key.(*ast.Ident).TokenLit
			key := &object.String{Value: name}
			val := Eval(p.Value, env)
			if object.IsError(val) {
				return val
			}
			hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: val}
			continue
		}
		key := evalPropertyKey(p.Key, env)
		if object.IsError(key) {
			return key
		}
		val := Eval(p.Value, env)
		if object.IsError(val) {
			return val
		}
		hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: val}
	}
	return hash
}

// ============================================================================
// Prefix / Infix Expressions
// ============================================================================

func evalPrefix(n *ast.PrefixExpr, env *object.Environment) object.Object {
	right := Eval(n.Right, env)
	if object.IsError(right) {
		return right
	}
	switch n.Op {
	case "!":
		return object.NativeBool(!object.IsTruthy(right))
	case "-":
		if num, ok := right.(*object.Number); ok {
			return &object.Number{Value: -num.Value}
		}
		return object.NewError(n.Pos(), "TypeError: cannot negate %s", right.Type())
	case "+":
		if num, ok := right.(*object.Number); ok {
			return num
		}
		return object.NewError(n.Pos(), "TypeError: cannot apply + to %s", right.Type())
	case "typeof":
		return &object.String{Value: string(right.Type())}
	case "void":
		return object.UNDEFINED
	case "delete":
		if _, ok := right.(*object.Hash); ok {
			return object.TRUE
		}
		return object.TRUE
	}
	return object.NewError(n.Pos(), "unknown prefix operator: %s", n.Op)
}

func evalInfix(n *ast.InfixExpr, env *object.Environment) object.Object {
	left := Eval(n.Left, env)
	if object.IsError(left) {
		return left
	}
	right := Eval(n.Right, env)
	if object.IsError(right) {
		return right
	}

	switch n.Op {
	case "+":
		return evalAdd(left, right, n.Pos())
	case "-":
		return evalNumberOp(left, right, n.Pos(), func(a, b float64) float64 { return a - b })
	case "*":
		return evalNumberOp(left, right, n.Pos(), func(a, b float64) float64 { return a * b })
	case "/":
		return evalNumberOp(left, right, n.Pos(), func(a, b float64) float64 { return a / b })
	case "%":
		return evalNumberOp(left, right, n.Pos(), math.Mod)
	case "**":
		return evalNumberOp(left, right, n.Pos(), math.Pow)
	case "===":
		return object.NativeBool(strictEqual(left, right))
	case "!==":
		return object.NativeBool(!strictEqual(left, right))
	case "<":
		return evalCompare(left, right, "<", n.Pos())
	case "<=":
		return evalCompare(left, right, "<=", n.Pos())
	case ">":
		return evalCompare(left, right, ">", n.Pos())
	case ">=":
		return evalCompare(left, right, ">=", n.Pos())
	case "&&":
		if !object.IsTruthy(left) {
			return left
		}
		return right
	case "||":
		if object.IsTruthy(left) {
			return left
		}
		return right
	case "??":
		if left == object.NULL || left == object.UNDEFINED {
			return right
		}
		return left
	default:
		return object.NewError(n.Pos(), "unknown infix operator: %s", n.Op)
	}
}

func evalAdd(left, right object.Object, pos ast.Position) object.Object {
	if object.IsNumber(left) && object.IsNumber(right) {
		return &object.Number{Value: left.(*object.Number).Value + right.(*object.Number).Value}
	}
	if object.IsString(left) && object.IsString(right) {
		return &object.String{Value: left.(*object.String).Value + right.(*object.String).Value}
	}
	if object.IsString(left) || object.IsString(right) {
		other := left
		if object.IsString(left) {
			other = right
		}
		return object.NewError(pos, "TypeError: cannot add string and %s — use template literals or String()", other.Type())
	}
	return object.NewError(pos, "TypeError: cannot add %s and %s — types must match", left.Type(), right.Type())
}

func evalNumberOp(left, right object.Object, pos ast.Position, fn func(float64, float64) float64) object.Object {
	l, ok := left.(*object.Number)
	if !ok {
		return object.NewError(pos, "TypeError: left operand must be number, got %s", left.Type())
	}
	r, ok := right.(*object.Number)
	if !ok {
		return object.NewError(pos, "TypeError: right operand must be number, got %s", right.Type())
	}
	return &object.Number{Value: fn(l.Value, r.Value)}
}

func strictEqual(a, b object.Object) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch a := a.(type) {
	case *object.Number:
		return a.Value == b.(*object.Number).Value
	case *object.String:
		return a.Value == b.(*object.String).Value
	case *object.Boolean:
		return a.Value == b.(*object.Boolean).Value
	case *object.Null:
		return true
	case *object.Undefined:
		return true
	default:
		return a == b
	}
}

func evalCompare(left, right object.Object, op string, pos ast.Position) object.Object {
	lNum, lIsNum := left.(*object.Number)
	rNum, rIsNum := right.(*object.Number)
	lStr, lIsStr := left.(*object.String)
	rStr, rIsStr := right.(*object.String)

	if lIsNum && rIsNum {
		switch op {
		case "<":
			return object.NativeBool(lNum.Value < rNum.Value)
		case "<=":
			return object.NativeBool(lNum.Value <= rNum.Value)
		case ">":
			return object.NativeBool(lNum.Value > rNum.Value)
		case ">=":
			return object.NativeBool(lNum.Value >= rNum.Value)
		}
	}
	if lIsStr && rIsStr {
		switch op {
		case "<":
			return object.NativeBool(lStr.Value < rStr.Value)
		case "<=":
			return object.NativeBool(lStr.Value <= rStr.Value)
		case ">":
			return object.NativeBool(lStr.Value > rStr.Value)
		case ">=":
			return object.NativeBool(lStr.Value >= rStr.Value)
		}
	}
	return object.NewError(pos, "TypeError: cannot compare %s and %s — types must match", left.Type(), right.Type())
}

// ============================================================================
// Ternary / Assign
// ============================================================================

func evalTernary(n *ast.TernaryExpr, env *object.Environment) object.Object {
	cond := Eval(n.Cond, env)
	if object.IsError(cond) {
		return cond
	}
	if object.IsTruthy(cond) {
		return Eval(n.Consequent, env)
	}
	return Eval(n.Alternate, env)
}

func evalAssign(n *ast.AssignExpr, env *object.Environment) object.Object {
	right := Eval(n.Right, env)
	if object.IsError(right) {
		return right
	}
	switch left := n.Left.(type) {
	case *ast.Ident:
		if n.Op == "=" {
			env.SetUp(left.TokenLit, right)
		} else {
			existing, ok := env.Get(left.TokenLit)
			if !ok {
				return object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
			}
			right = evalCompoundAssign(existing, right, n.Op, n.Pos())
			if object.IsError(right) {
				return right
			}
			env.Set(left.TokenLit, right)
		}
		return right
	case *ast.MemberExpr:
		obj := Eval(left.Object, env)
		if hash, ok := obj.(*object.Hash); ok {
			key := Eval(left.Property, env)
			hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: right}
			return right
		}
		return object.NewError(left.Pos(), "TypeError: cannot assign to property of %T", obj)
	case *ast.IndexExpr:
		arr := Eval(left.Left, env)
		if a, ok := arr.(*object.Array); ok {
			idx := Eval(left.Index, env)
			if num, ok := idx.(*object.Number); ok {
				i := int(num.Value)
				if i >= 0 && i < len(a.Elements) {
					a.Elements[i] = right
				}
				return right
			}
			return object.NewError(left.Pos(), "TypeError: array index must be number")
		}
		return object.NewError(left.Pos(), "TypeError: cannot index %s", arr.Type())
	}
	return object.NewError(n.Left.Pos(), "cannot assign to %T", n.Left)
}

func evalCompoundAssign(left, right object.Object, op string, pos ast.Position) object.Object {
	lNum, lOk := left.(*object.Number)
	rNum, rOk := right.(*object.Number)
	if lOk && rOk {
		switch op {
		case "+=":
			return &object.Number{Value: lNum.Value + rNum.Value}
		case "-=":
			return &object.Number{Value: lNum.Value - rNum.Value}
		case "*=":
			return &object.Number{Value: lNum.Value * rNum.Value}
		case "/=":
			return &object.Number{Value: lNum.Value / rNum.Value}
		case "%=":
			return &object.Number{Value: math.Mod(lNum.Value, rNum.Value)}
		}
	}
	if lOk {
		return object.NewError(pos, "TypeError: cannot %s with non-number", op)
	}
	lStr, lOk := left.(*object.String)
	if lOk && op == "+=" && object.IsString(right) {
		return &object.String{Value: lStr.Value + right.(*object.String).Value}
	}
	if lOk && op == "+=" {
		return object.NewError(pos, "TypeError: cannot += with different types")
	}
	return object.NewError(pos, "TypeError: compound assignment requires matching types")
}

// ============================================================================
// Call / Member / Index
// ============================================================================

func evalCall(n *ast.CallExpr, env *object.Environment) object.Object {
	callee := Eval(n.Callee, env)
	if object.IsError(callee) {
		return callee
	}
	args := make([]object.Object, len(n.Args))
	for i, a := range n.Args {
		args[i] = Eval(a, env)
		if object.IsError(args[i]) {
			return args[i]
		}
	}
	return applyFunction(callee, env, args, n.Pos())
}

func applyFunction(fn object.Object, env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		scope := f.Env.NewScope()
		for i, p := range f.Parameters {
			if i < len(args) {
				if p.Spread {
					rest := make([]object.Object, len(args)-i)
					copy(rest, args[i:])
					scope.Set(p.Name, &object.Array{Elements: rest})
					break
				}
				scope.Set(p.Name, args[i])
			} else if p.Default != nil {
				scope.Set(p.Name, Eval(p.Default, f.Env))
			} else {
				scope.Set(p.Name, object.UNDEFINED)
			}
		}
		result := Eval(f.Body, scope)
		if rv, ok := result.(*object.ReturnValue); ok {
			return rv.Value
		}
		return result
	case *object.Builtin:
		return f.Fn(env, pos, args...)
	case *object.Class:
		inst := &object.Instance{Class: f, Props: make(map[string]object.Object), Pos: pos}
		for k, v := range f.Fields {
			inst.Props[k] = v
		}
		if con, ok := f.Methods["constructor"]; ok {
			scope := con.Env.NewScope()
			scope.Set("this", inst)
			for i, p := range con.Parameters {
				if i < len(args) {
					scope.Set(p.Name, args[i])
				} else if p.Default != nil {
					scope.Set(p.Name, Eval(p.Default, con.Env))
				} else {
					scope.Set(p.Name, object.UNDEFINED)
				}
			}
			Eval(con.Body, scope)
		}
		return inst
	default:
		return object.NewError(pos, "TypeError: %s is not a function", fn.Type())
	}
}

func evalMember(n *ast.MemberExpr, env *object.Environment) object.Object {
	obj := Eval(n.Object, env)
	if object.IsError(obj) {
		return obj
	}
	prop := n.Property.(*ast.Ident).TokenLit
	return getProperty(obj, prop, n.Pos())
}

func evalIndex(n *ast.IndexExpr, env *object.Environment) object.Object {
	left := Eval(n.Left, env)
	if object.IsError(left) {
		return left
	}
	idx := Eval(n.Index, env)
	if object.IsError(idx) {
		return idx
	}
	switch l := left.(type) {
	case *object.Array:
		if num, ok := idx.(*object.Number); ok {
			i := int(num.Value)
			if i >= 0 && i < len(l.Elements) {
				return l.Elements[i]
			}
		}
		return object.UNDEFINED
	case *object.Hash:
		return getHashKey(l, idx)
	case *object.String:
		if num, ok := idx.(*object.Number); ok {
			i := int(num.Value)
			if i >= 0 && i < len(l.Value) {
				return &object.String{Value: string(l.Value[i])}
			}
		}
		return object.UNDEFINED
	case *object.Instance:
		if key, ok := idx.(*object.String); ok {
			if v, ok := l.Props[key.Value]; ok {
				return v
			}
			if m, ok := l.Class.Methods[key.Value]; ok {
				return m
			}
		}
		return object.UNDEFINED
	default:
		return object.NewError(n.Pos(), "TypeError: cannot index %s", left.Type())
	}
}

func evalOptional(n *ast.OptionalExpr, env *object.Environment) object.Object {
	obj := Eval(n.Object, env)
	if obj == object.NULL || obj == object.UNDEFINED {
		return object.UNDEFINED
	}
	if n.IsCall {
		if f, ok := obj.(*object.Function); ok {
			args := make([]object.Object, len(n.Args))
			for i, a := range n.Args {
				args[i] = Eval(a, env)
			}
			return applyFunction(f, env, args, n.Pos())
		}
		return object.UNDEFINED
	}
	switch prop := n.Property.(type) {
	case *ast.Ident:
		return getProperty(obj, prop.TokenLit, n.Pos())
	default:
		key := Eval(n.Property, env)
		return getHashKey(obj, key)
	}
}

func getProperty(obj object.Object, name string, pos ast.Position) object.Object {
	switch o := obj.(type) {
	case *object.Hash:
		return getHashKey(o, &object.String{Value: name})
	case *object.Instance:
		if v, ok := o.Props[name]; ok {
			return v
		}
		if m, ok := o.Class.Methods[name]; ok {
			bound := *m
			methodScope := m.Env.NewScope()
			methodScope.Set("this", o)
			bound.Env = methodScope
			return &bound
		}
	case *object.String:
		switch name {
		case "length":
			return &object.Number{Value: float64(len(o.Value))}
		}
	case *object.Array:
		switch name {
		case "length":
			return &object.Number{Value: float64(len(o.Elements))}
		}
	}
	return object.NewError(pos, "TypeError: cannot read property '%s' of %s", name, obj.Type())
}

func getHashKey(obj object.Object, key object.Object) object.Object {
	switch o := obj.(type) {
	case *object.Hash:
		if pair, ok := o.Pairs[hashKey(key)]; ok {
			return pair.Value
		}
		return object.UNDEFINED
	default:
		return object.UNDEFINED
	}
}

func hashKey(o object.Object) object.HashKey {
	switch o := o.(type) {
	case *object.String:
		return object.HashKey{Type: o.Type(), Value: o.Value}
	case *object.Number:
		return object.HashKey{Type: o.Type(), Value: fmt.Sprintf("%v", o.Value)}
	case *object.Boolean:
		if o.Value {
			return object.HashKey{Type: o.Type(), Value: "true"}
		}
		return object.HashKey{Type: o.Type(), Value: "false"}
	case *object.Null:
		return object.HashKey{Type: o.Type(), Value: "null"}
	default:
		return object.HashKey{Type: o.Type(), Value: o.Inspect()}
	}
}

// ============================================================================
// Function / Arrow expressions
// ============================================================================

func evalFuncExpr(n *ast.FuncExpr, env *object.Environment) object.Object {
	return &object.Function{
		Name:       n.Name,
		Parameters: n.Params,
		Body:       n.Body,
		Env:        env,
		Pos:        n.Pos(),
	}
}

func evalArrowFunc(n *ast.ArrowFuncExpr, env *object.Environment) object.Object {
	block, ok := n.Body.(*ast.BlockStmt)
	if !ok {
		block = &ast.BlockStmt{Statements: []ast.Statement{
			&ast.ReturnStmt{Value: n.Body.(ast.Expression)},
		}}
	}
	return &object.Function{
		Name:       "",
		Parameters: n.Params,
		Body:       block,
		Env:        env,
		Pos:        n.Pos(),
	}
}

// ============================================================================
// New expression
// ============================================================================

func evalNew(n *ast.NewExpr, env *object.Environment) object.Object {
	callee := Eval(n.Callee, env)
	if object.IsError(callee) {
		return callee
	}
	args := make([]object.Object, len(n.Args))
	for i, a := range n.Args {
		args[i] = Eval(a, env)
	}
	return applyFunction(callee, env, args, n.Pos())
}

// ============================================================================
// Await (stub)
// ============================================================================

func evalAwait(n *ast.AwaitExpr, env *object.Environment) object.Object {
	return Eval(n.Value, env)
}

// ============================================================================
// Match expression
// ============================================================================

func evalMatch(n *ast.MatchExpr, env *object.Environment) object.Object {
	subject := Eval(n.Expr, env)
	if object.IsError(subject) {
		return subject
	}
	for _, arm := range n.Arms {
		scope := env.NewScope()
		if matchPattern(arm.Pattern, subject, scope) {
			if arm.Guard != nil {
				guard := Eval(arm.Guard, scope)
				if !object.IsTruthy(guard) {
					continue
				}
			}
			return Eval(arm.Body.(ast.Node), scope)
		}
	}
	return object.NewError(n.Pos(), "MatchError: no arm matched for %s", subject.Inspect())
}

func matchPattern(pat ast.Pattern, value object.Object, scope *object.Environment) bool {
	switch p := pat.(type) {
	case *ast.LiteralPattern:
		v := Eval(p.Value, &object.Environment{})
		return strictEqual(v, value)
	case *ast.IdentPattern:
		scope.Set(p.Name, value)
		return true
	case *ast.WildcardPattern:
		return true
	case *ast.OrPattern:
		for _, alt := range p.Alternatives {
			if matchPattern(alt, value, scope) {
				return true
			}
		}
		return false
	case *ast.RangePattern:
		start := Eval(p.Start, &object.Environment{})
		end := Eval(p.End, &object.Environment{})
		vNum, ok := value.(*object.Number)
		if !ok {
			return false
		}
		sNum, sOk := start.(*object.Number)
		eNum, eOk := end.(*object.Number)
		if !sOk || !eOk {
			return false
		}
		if p.Inclusive {
			return vNum.Value >= sNum.Value && vNum.Value <= eNum.Value
		}
		return vNum.Value >= sNum.Value && vNum.Value < eNum.Value
	}
	return false
}

// ============================================================================
// Super
// ============================================================================

func evalPropertyKey(keyExpr ast.Expression, env *object.Environment) object.Object {
	switch k := keyExpr.(type) {
	case *ast.Ident:
		return &object.String{Value: k.TokenLit}
	case *ast.StringLit:
		return &object.String{Value: k.TokenLit[1 : len(k.TokenLit)-1]}
	case *ast.NumberLit:
		return &object.String{Value: k.TokenLit}
	default:
		return Eval(keyExpr, env)
	}
}

func evalSuper(n *ast.SuperExpr, env *object.Environment) object.Object {
	this, _ := env.Get("this")
	inst, ok := this.(*object.Instance)
	if !ok || inst.Class.Super == nil {
		return object.NewError(n.Pos(), "ReferenceError: super is not available")
	}
	if n.Method != "" {
		if m, ok := inst.Class.Super.Methods[n.Method]; ok {
			bound := *m
			methodScope := m.Env.NewScope()
			methodScope.Set("this", inst)
			bound.Env = methodScope
			return &bound
		}
	}
	return object.UNDEFINED
}
