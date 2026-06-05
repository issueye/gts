package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/parser"
)

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
		result, err = r.waitPromise(promise, "main promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

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
		result, err = r.waitPromise(promise, "export promise")
		if err != nil {
			return nil, err
		}
	}
	if object.IsError(result) {
		return nil, errors.New(result.Inspect())
	}
	return result, nil
}

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

func readResolvedSource(resolved module.ResolvedModule) (string, error) {
	if resolved.PackageFile != "" {
		return packagefile.ReadNestedText(resolved.PackageFile, resolved.ArchivePath)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func resolvedModuleDir(resolved module.ResolvedModule) string {
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(filepath.Dir(resolved.ArchivePath))
	}
	return filepath.Dir(resolved.Path)
}

func evalStandalone(source, file string, env *object.Environment) (object.Object, error) {
	l := lexer.New(source)
	p := parser.New(l, file)
	program := p.ParseProgram()
	parseErrors := append([]string{}, l.Errors()...)
	parseErrors = append(parseErrors, program.Errors...)
	if len(parseErrors) > 0 {
		return nil, errors.New(parseErrors[0])
	}
	return evaluator.Eval(program, env), nil
}
