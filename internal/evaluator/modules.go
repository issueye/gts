package evaluator

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// ============================================================================
// Import / Export
// ============================================================================

func evalImport(n *ast.ImportDecl, env *object.Environment) object.Object {
	exports, err := env.VM().Import(env, unquoteModulePath(n.Source))
	if err != nil {
		return object.NewError(n.Pos(), "ImportError: %v", err)
	}
	if exports == nil {
		return object.NewError(n.Pos(), "ImportError: module loader is not configured")
	}

	if n.Namespace != "" {
		env.Set(n.Namespace, exports)
	}
	if n.Default != "" {
		value := getExport(exports, "default")
		if value == object.UNDEFINED {
			if isNativeModulePath(unquoteModulePath(n.Source)) {
				value = exports
			} else {
				return object.NewError(n.Pos(), "ImportError: module %s has no default export", n.Source)
			}
		}
		env.Set(n.Default, value)
	}
	for _, name := range n.Names {
		value := getExport(exports, name)
		if value == object.UNDEFINED {
			return object.NewError(n.Pos(), "ImportError: module %s has no export %s", n.Source, name)
		}
		env.Set(name, value)
	}
	for exported, local := range n.Aliases {
		value := getExport(exports, exported)
		if value == object.UNDEFINED {
			return object.NewError(n.Pos(), "ImportError: module %s has no export %s", n.Source, exported)
		}
		env.Set(local, value)
	}
	return object.UNDEFINED
}

func evalExport(n *ast.ExportDecl, env *object.Environment) object.Object {
	if len(n.Specifiers) > 0 {
		for _, spec := range n.Specifiers {
			value, ok := env.Get(spec.Name)
			if !ok {
				return object.NewError(n.Pos(), "ExportError: %s was not defined", spec.Name)
			}
			setExport(env, spec.Alias, value)
		}
		return object.UNDEFINED
	}
	if n.Decl == nil {
		return object.UNDEFINED
	}

	if n.IsDefault {
		stmt, ok := n.Decl.(*ast.ExprStmt)
		if !ok {
			return object.NewError(n.Pos(), "ExportError: default export must be an expression")
		}
		value := Eval(stmt.Expr, env)
		if object.IsRuntimeError(value) {
			return value
		}
		setExport(env, "default", value)
		return value
	}

	result := Eval(n.Decl, env)
	if object.IsRuntimeError(result) {
		return result
	}
	for _, name := range exportedNames(n.Decl) {
		value, ok := env.Get(name)
		if !ok {
			return object.NewError(n.Pos(), "ExportError: %s was not defined", name)
		}
		setExport(env, name, value)
	}
	return result
}

func exportedNames(stmt ast.Statement) []string {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		return []string{s.Name}
	case *ast.ConstStmt:
		return []string{s.Name}
	case *ast.VarStmt:
		return []string{s.Name}
	case *ast.FuncDecl:
		return []string{s.Name}
	case *ast.ClassDecl:
		return []string{s.Name}
	default:
		return nil
	}
}

func setExport(env *object.Environment, name string, value object.Object) {
	exportsObj, ok := env.Get("exports")
	if !ok {
		return
	}
	if exports, ok := exportsObj.(*object.Hash); ok {
		key := &object.String{Value: name}
		exports.SetMember(key, value)
	}
}

func getExport(exports object.Object, name string) object.Object {
	hash, ok := exports.(*object.Hash)
	if !ok {
		return object.UNDEFINED
	}
	key := &object.String{Value: name}
	if pair, ok := hash.Pairs[hashKey(key)]; ok {
		return pair.Value
	}
	return object.UNDEFINED
}

func isNativeModulePath(path string) bool {
	return strings.HasPrefix(path, "@std/")
}

func unquoteModulePath(path string) string {
	if len(path) >= 2 {
		first := path[0]
		last := path[len(path)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return path[1 : len(path)-1]
		}
	}
	return path
}
