package sdk

import (
	"errors"
	"fmt"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/object"
)

// callMain invokes a top-level main() if present, awaiting any Promise it
// returns. Hosts reach this via RunFile(autoMain=true) / RunProject.
func (r *Runtime) callMain(env *object.Environment, file string) (object.Object, error) {
	mainFn, ok := env.Get("main")
	if !ok {
		return object.UNDEFINED, nil
	}
	if _, ok := mainFn.(*object.Function); !ok {
		return nil, fmt.Errorf("%s: top-level main is not a function", file)
	}
	pos := ast.Position{File: file}
	call := &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "main",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "main"},
	}
	result := evaluator.Eval(call, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.sess.WaitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

// callValue invokes a script function value with Go-side Value arguments. The
// arguments are staged as environment bindings and referenced by synthesized
// identifier AST nodes, so the call flows through the normal evaluator.
func (r *Runtime) callValue(fn object.Object, env *object.Environment, args []Value, file string) (object.Object, error) {
	pos := ast.Position{File: file}
	argExprs := make([]ast.Expression, len(args))
	for i, arg := range args {
		name := fmt.Sprintf("__sdk_arg_%d", i)
		env.Set(name, arg)
		argExprs[i] = &ast.Ident{Pos_: pos, TokenLit: name}
	}
	env.Set("__sdk_call_target", fn)
	call := &ast.CallExpr{
		Pos_:     pos,
		TokenLit: "__sdk_call_target",
		Callee:   &ast.Ident{Pos_: pos, TokenLit: "__sdk_call_target"},
		Args:     argExprs,
	}
	result := evaluator.Eval(call, env)
	if promise, ok := result.(*object.Promise); ok {
		var err error
		result, err = r.sess.WaitPromise(promise, "export promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

// exportedValue reads a named export from a module's exports object.
func exportedValue(exports object.Object, name string) (object.Object, error) {
	hash, ok := exports.(*object.Hash)
	if !ok {
		return nil, fmt.Errorf("module exports is not an object")
	}
	key := &object.String{Value: name}
	pair, ok := hash.Pairs[object.HashKeyFor(key)]
	if !ok || pair.Value == object.UNDEFINED || pair.Value == object.NULL {
		return nil, fmt.Errorf("module has no export %q", name)
	}
	return pair.Value, nil
}
