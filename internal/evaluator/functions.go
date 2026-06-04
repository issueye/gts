package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// ============================================================================
// Function / Arrow expressions
// ============================================================================

func evalFuncExpr(n *ast.FuncExpr, env *object.Environment) object.Object {
	fn := &object.Function{
		Name:       n.Name,
		Parameters: n.Params,
		Body:       n.Body,
		Env:        env,
		IsAsync:    n.IsAsync,
		ReturnT:    n.ReturnT,
		Pos:        n.Pos(),
	}
	env.ObjectManager().Register(fn)
	return fn
}

func evalArrowFunc(n *ast.ArrowFuncExpr, env *object.Environment) object.Object {
	block, ok := n.Body.(*ast.BlockStmt)
	if !ok {
		block = &ast.BlockStmt{Statements: []ast.Statement{
			&ast.ReturnStmt{Value: n.Body.(ast.Expression)},
		}}
	}
	fn := &object.Function{
		Name:       "",
		Parameters: n.Params,
		Body:       block,
		Env:        env,
		IsAsync:    n.IsAsync,
		ReturnT:    n.ReturnT,
		Pos:        n.Pos(),
	}
	env.ObjectManager().Register(fn)
	return fn
}

// ============================================================================
// New expression
// ============================================================================

func evalNew(n *ast.NewExpr, env *object.Environment) object.Object {
	callee := Eval(n.Callee, env)
	if object.IsRuntimeError(callee) {
		return callee
	}
	args := make([]object.Object, len(n.Args))
	for i, a := range n.Args {
		args[i] = Eval(a, env)
	}
	if hash, ok := callee.(*object.Hash); ok {
		if marker, ok := getHashKey(hash, &object.String{Value: "__constructBoolean"}).(*object.Boolean); ok && marker.Value {
			return constructBooleanObject(args)
		}
	}
	return applyFunction(callee, env, args, n.Pos())
}
