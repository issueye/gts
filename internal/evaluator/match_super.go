package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// ============================================================================
// Await (stub)
// ============================================================================

func evalAwait(n *ast.AwaitExpr, env *object.Environment) object.Object {
	val := Eval(n.Value, env)
	if promise, ok := val.(*object.Promise); ok {
		result := promise.Wait()
		if promise.State() == object.PROMISE_REJECTED {
			if err, ok := result.(*object.Error); ok {
				err.Runtime = true
				if err.Pos.IsZero() {
					err.Pos = n.Pos()
				}
				if err.Stack == "" {
					err.Stack = err.FormatStack()
				}
				return err
			}
			return object.NewError(n.Pos(), "%s", result.Inspect())
		}
		return result
	}
	return val
}

// ============================================================================
// Match expression
// ============================================================================

func evalMatch(n *ast.MatchExpr, env *object.Environment) object.Object {
	subject := Eval(n.Expr, env)
	if object.IsRuntimeError(subject) {
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

func callSuperConstructor(pos ast.Position, env *object.Environment, args []object.Object) object.Object {
	this, _ := env.Get("this")
	inst, ok := this.(*object.Instance)
	if !ok {
		return object.NewError(pos, "ReferenceError: super is not available")
	}
	current := env.ConstructorClass
	if current == nil {
		current = inst.Class
	}
	if current.Super == nil {
		return object.NewError(pos, "ReferenceError: super is not available")
	}
	if env.SuperCalled != nil && *env.SuperCalled {
		return object.NewError(pos, "ReferenceError: super constructor already called")
	}
	if env.SuperCalled != nil {
		*env.SuperCalled = true
	}
	return callClassConstructor(current.Super, inst, env, args, pos)
}
