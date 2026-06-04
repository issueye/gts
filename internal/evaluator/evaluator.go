package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/typechecker"
)

const (
	breakSignal    = "__break__"
	continueSignal = "__continue__"
)

type ImportFn func(env *object.Environment, path string) (object.Object, error)

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
		return &object.ReturnValue{Value: &object.Error{Message: breakSignal, Name: "SyntaxError", Pos: n.Pos(), Runtime: true}}
	case *ast.ContinueStmt:
		return &object.ReturnValue{Value: &object.Error{Message: continueSignal, Name: "SyntaxError", Pos: n.Pos(), Runtime: true}}
	case *ast.ThrowStmt:
		val := Eval(n.Value, env)
		if object.IsRuntimeError(val) {
			return val
		}
		if inst, ok := val.(*object.Instance); ok && isErrorInstance(inst) {
			return runtimeErrorFromInstance(n.Pos(), inst)
		}
		if err, ok := val.(*object.Error); ok {
			err.Runtime = true
			if err.Pos.IsZero() {
				err.Pos = n.Pos()
			}
			if err.Stack == "" {
				err.Stack = err.FormatStack()
			}
			return err
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
	case *ast.RegExpLit:
		return evalRegExpLit(n)
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

	case *ast.ImportDecl:
		return evalImport(n, env)
	case *ast.ExportDecl:
		return evalExport(n, env)
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
			if err, ok := rv.Value.(*object.Error); ok {
				switch err.Message {
				case breakSignal:
					return object.NewError(err.Pos, "SyntaxError: break outside loop")
				case continueSignal:
					return object.NewError(err.Pos, "SyntaxError: continue outside loop")
				}
			}
			return rv.Value
		}
		if object.IsRuntimeError(result) {
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
		if object.IsRuntimeError(val) {
			return val
		}
	}
	if err := checkType(env, n.Pos(), n.TypeAnno, val); err != nil {
		return err
	}
	env.SetTyped(n.Name, val, n.TypeAnno)
	return object.UNDEFINED
}

func evalConst(n *ast.ConstStmt, env *object.Environment) object.Object {
	var val object.Object = object.UNDEFINED
	if n.Value != nil {
		val = Eval(n.Value, env)
		if object.IsRuntimeError(val) {
			return val
		}
	}
	if err := checkType(env, n.Pos(), n.TypeAnno, val); err != nil {
		return err
	}
	env.SetTypedConst(n.Name, val, n.TypeAnno)
	return object.UNDEFINED
}

func evalVar(n *ast.VarStmt, env *object.Environment) object.Object {
	var val object.Object = object.UNDEFINED
	if n.Value != nil {
		val = Eval(n.Value, env)
		if object.IsRuntimeError(val) {
			return val
		}
	}
	if err := checkType(env, n.Pos(), n.TypeAnno, val); err != nil {
		return err
	}
	env.SetTyped(n.Name, val, n.TypeAnno)
	return object.UNDEFINED
}

func checkType(env *object.Environment, pos ast.Position, anno *ast.TypeAnnotation, value object.Object) *object.Error {
	if env == nil || !env.VM().TypeCheck() || anno == nil {
		return nil
	}
	return typechecker.Check(pos, anno, value)
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
		if object.IsRuntimeError(result) {
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
	if object.IsRuntimeError(cond) {
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
		if object.IsRuntimeError(cond) {
			return cond
		}
		if !object.IsTruthy(cond) {
			break
		}
		result = Eval(n.Body, env.NewScope())
		if rv, ok := result.(*object.ReturnValue); ok {
			if signal := controlSignal(rv); signal == breakSignal {
				break
			} else if signal == continueSignal {
				continue
			}
			return rv
		}
		if object.IsRuntimeError(result) {
			return result
		}
	}
	return object.UNDEFINED
}

func evalFor(n *ast.ForStmt, env *object.Environment) object.Object {
	scope := env.NewScope()
	if n.Init != nil {
		result := Eval(n.Init, scope)
		if object.IsRuntimeError(result) {
			return result
		}
	}
	for {
		if n.Cond != nil {
			cond := Eval(n.Cond, scope)
			if object.IsRuntimeError(cond) {
				return cond
			}
			if !object.IsTruthy(cond) {
				break
			}
		}
		result := Eval(n.Body, scope.NewScope())
		if rv, ok := result.(*object.ReturnValue); ok {
			if signal := controlSignal(rv); signal == breakSignal {
				break
			} else if signal == continueSignal {
				// continue to post expression below
			} else {
				return rv
			}
		}
		if object.IsRuntimeError(result) {
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
	if object.IsRuntimeError(iterable) {
		return iterable
	}
	it, ok, err := newRuntimeIterator(iterable, object.IterateKeys, env, n.Pos())
	if err != nil {
		return err
	}
	if !ok {
		return object.NewError(n.Pos(), "cannot iterate over %s", iterable.Type())
	}
	return evalIteratorLoop(n.Name, n.Body, env, it)
}

func evalForOf(n *ast.ForOfStmt, env *object.Environment) object.Object {
	iterable := Eval(n.Iterable, env)
	if object.IsRuntimeError(iterable) {
		return iterable
	}
	it, ok, err := newRuntimeIterator(iterable, object.IterateValues, env, n.Pos())
	if err != nil {
		return err
	}
	if !ok {
		return object.NewError(n.Pos(), "cannot for-of over %s", iterable.Type())
	}
	return evalIteratorLoop(n.Name, n.Body, env, it)
}

func newRuntimeIterator(obj object.Object, kind object.IterationKind, env *object.Environment, pos ast.Position) (object.Iterator, bool, object.Object) {
	if protocol := getIteratorProtocol(obj, kind); protocol != nil {
		result := applyFunction(protocol, env, nil, pos)
		if object.IsRuntimeError(result) {
			return nil, false, result
		}
		it, ok := object.NewIterator(result, object.IterateValues)
		if !ok {
			return nil, false, object.NewError(pos, "TypeError: iterator protocol must return an iterable object")
		}
		return it, true, nil
	}

	it, ok := object.NewIterator(obj, kind)
	return it, ok, nil
}

func getIteratorProtocol(obj object.Object, kind object.IterationKind) object.Object {
	name := "__iterator"
	if kind == object.IterateKeys {
		name = "__keyIterator"
	}

	switch o := obj.(type) {
	case *object.Hash:
		if fn := getHashKey(o, &object.String{Value: name}); fn != object.UNDEFINED {
			return fn
		}
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
	}
	return nil
}

func evalIteratorLoop(name string, body ast.Node, env *object.Environment, it object.Iterator) object.Object {
	scope := env.NewScope()
	for {
		value, ok := it.Next()
		if !ok {
			break
		}
		scope.Set(name, value)
		result := Eval(body, scope.NewScope())
		if rv, ok := result.(*object.ReturnValue); ok {
			if signal := controlSignal(rv); signal == breakSignal {
				break
			} else if signal == continueSignal {
				continue
			}
			return rv
		}
		if object.IsRuntimeError(result) {
			return result
		}
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
	if object.IsRuntimeError(val) {
		return val
	}
	return &object.ReturnValue{Value: val}
}

func evalLabeled(n *ast.LabeledStmt, env *object.Environment) object.Object {
	result := Eval(n.Stmt, env)
	if rv, ok := result.(*object.ReturnValue); ok {
		if err, ok2 := rv.Value.(*object.Error); ok2 {
			if err.Message == breakSignal || err.Message == continueSignal {
				return rv
			}
		}
	}
	return result
}

func controlSignal(rv *object.ReturnValue) string {
	if rv == nil {
		return ""
	}
	err, ok := rv.Value.(*object.Error)
	if !ok {
		return ""
	}
	if err.Message == breakSignal || err.Message == continueSignal {
		return err.Message
	}
	return ""
}

// ============================================================================
// Try / Catch / Finally
// ============================================================================

func evalTry(n *ast.TryStmt, env *object.Environment) object.Object {
	result := Eval(n.Block, env.NewScope())
	if object.IsRuntimeError(result) && n.Catch != nil {
		scope := env.NewScope()
		if n.Catch.Name != "" {
			if err, ok := result.(*object.Error); ok {
				if err.Thrown != nil {
					scope.Set(n.Catch.Name, err.Thrown)
					result = Eval(n.Catch.Body, scope)
					if n.Finalizer != nil {
						Eval(n.Finalizer, env.NewScope())
					}
					return result
				}
				caught := *err
				caught.Runtime = false
				scope.Set(n.Catch.Name, &caught)
			} else {
				scope.Set(n.Catch.Name, result)
			}
		}
		result = Eval(n.Catch.Body, scope)
	}
	if n.Finalizer != nil {
		Eval(n.Finalizer, env.NewScope())
	}
	return result
}
