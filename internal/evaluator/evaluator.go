package evaluator

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
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
		exports.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: value}
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
		ReturnT:    n.ReturnT,
		Pos:        n.Pos(),
	}
	env.ObjectManager().Register(fn)
	env.Set(n.Name, fn)
	return fn
}

func evalClassDecl(n *ast.ClassDecl, env *object.Environment) object.Object {
	cls := evalClass(n, env)
	if object.IsRuntimeError(cls) {
		return cls
	}
	if n.Name != "" {
		env.Set(n.Name, cls)
	}
	return cls
}

func evalClass(n *ast.ClassDecl, env *object.Environment) object.Object {
	cls := &object.Class{
		Name:        n.Name,
		Methods:     make(map[string]*object.Function),
		Fields:      make(map[string]object.Object),
		FieldTypes:  make(map[string]*ast.TypeAnnotation),
		Statics:     make(map[string]object.Object),
		StaticTypes: make(map[string]*ast.TypeAnnotation),
		Pos:         n.Pos(),
	}
	env.ObjectManager().Register(cls)
	// Resolve super class
	if n.Super != nil {
		superVal := Eval(n.Super, env)
		if superClass, ok := superVal.(*object.Class); ok {
			cls.Super = superClass
			// Copy parent methods
			for k, v := range superClass.Methods {
				if k == "constructor" {
					continue
				}
				cls.Methods[k] = v
			}
			for k, v := range superClass.Fields {
				cls.Fields[k] = v
			}
			for k, v := range superClass.FieldTypes {
				cls.FieldTypes[k] = v
			}
		} else if builtin, ok := superVal.(*object.Builtin); ok && isErrorClassName(builtin.Name) {
			cls.Super = nativeErrorClass(env, builtin.Name, n.Pos())
		} else {
			return object.NewError(n.Pos(), "TypeError: superclass must be a class")
		}
	}
	// Parse members
	for _, m := range n.Body.Members {
		switch m.Kind {
		case "constructor", "method":
			if m.IsStatic {
				fn := &object.Function{
					Name:       m.Name,
					Parameters: m.Params,
					Body:       m.Body,
					Env:        env,
					IsAsync:    m.IsAsync,
					Pos:        m.Pos_,
				}
				env.ObjectManager().Register(fn)
				cls.Statics[m.Name] = fn
				continue
			}
			fn := &object.Function{
				Name:       m.Name,
				Parameters: m.Params,
				Body:       m.Body,
				Env:        env,
				IsAsync:    m.IsAsync,
				Pos:        m.Pos_,
			}
			env.ObjectManager().Register(fn)
			cls.Methods[m.Name] = fn
		case "field":
			var val object.Object = object.UNDEFINED
			if m.DefaultVal != nil {
				val = Eval(m.DefaultVal, env)
				if object.IsRuntimeError(val) {
					return val
				}
			}
			if err := checkType(env, m.Pos_, m.TypeAnno, val); err != nil {
				return err
			}
			if m.IsStatic {
				cls.Statics[m.Name] = val
				cls.StaticTypes[m.Name] = m.TypeAnno
			} else {
				cls.Fields[m.Name] = val
				cls.FieldTypes[m.Name] = m.TypeAnno
			}
		}
	}
	return cls
}

func nativeErrorClass(env *object.Environment, name string, pos ast.Position) *object.Class {
	cls := &object.Class{
		Name:        name,
		Methods:     make(map[string]*object.Function),
		Fields:      make(map[string]object.Object),
		FieldTypes:  make(map[string]*ast.TypeAnnotation),
		Statics:     make(map[string]object.Object),
		StaticTypes: make(map[string]*ast.TypeAnnotation),
		Pos:         pos,
	}
	cls.NativeConstructor = func(env *object.Environment, inst *object.Instance, pos ast.Position, args []object.Object) object.Object {
		message := ""
		if len(args) > 0 {
			message = args[0].Inspect()
		}
		err := object.NewNamedError(pos, name, message)
		inst.Props["name"] = &object.String{Value: err.Name}
		inst.Props["message"] = &object.String{Value: err.Message}
		inst.Props["stack"] = &object.String{Value: err.Stack}
		return object.UNDEFINED
	}
	env.ObjectManager().Register(cls)
	return cls
}

func isErrorClassName(name string) bool {
	switch name {
	case "Error", "TypeError", "RangeError", "ReferenceError", "SyntaxError":
		return true
	default:
		return false
	}
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
			if object.IsRuntimeError(val) {
				return val
			}
			hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: val}
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
		hash.Pairs[hashKey(key)] = object.HashPair{Key: key, Value: val}
	}
	return hash
}

// ============================================================================
// Prefix / Infix Expressions
// ============================================================================

func evalPrefix(n *ast.PrefixExpr, env *object.Environment) object.Object {
	right := Eval(n.Right, env)
	if object.IsRuntimeError(right) {
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
	if object.IsRuntimeError(left) {
		return left
	}
	right := Eval(n.Right, env)
	if object.IsRuntimeError(right) {
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
	case "instanceof":
		return evalInstanceOf(left, right)
	case "in":
		return evalIn(left, right, n.Pos())
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

func evalIn(left, right object.Object, pos ast.Position) object.Object {
	if _, ok := left.(*object.String); !ok {
		return object.NewError(pos, "TypeError: left operand of 'in' must be string")
	}
	switch r := right.(type) {
	case *object.Hash:
		_, ok := r.Pairs[hashKey(left)]
		return object.NativeBool(ok)
	case *object.Array:
		idx, ok := left.(*object.String)
		if !ok {
			return object.FALSE
		}
		i, err := strconv.Atoi(idx.Value)
		return object.NativeBool(err == nil && i >= 0 && i < len(r.Elements))
	case *object.Instance:
		key := left.(*object.String).Value
		if _, ok := r.Props[key]; ok {
			return object.TRUE
		}
		_, ok := r.Class.Methods[key]
		return object.NativeBool(ok)
	default:
		return object.NewError(pos, "TypeError: right operand of 'in' must be object")
	}
}

func evalInstanceOf(left, right object.Object) object.Object {
	if err, ok := left.(*object.Error); ok {
		if builtin, ok := right.(*object.Builtin); ok && isErrorClassName(builtin.Name) {
			return object.NativeBool(err.Name == builtin.Name || builtin.Name == "Error")
		}
	}
	inst, ok := left.(*object.Instance)
	if !ok {
		return object.FALSE
	}
	if builtin, ok := right.(*object.Builtin); ok && isErrorClassName(builtin.Name) {
		return object.NativeBool(instanceExtendsNativeError(inst, builtin.Name))
	}
	cls, ok := right.(*object.Class)
	if !ok {
		return object.FALSE
	}
	for current := inst.Class; current != nil; current = current.Super {
		if current == cls {
			return object.TRUE
		}
	}
	return object.FALSE
}

func isErrorInstance(inst *object.Instance) bool {
	return instanceExtendsNativeError(inst, "Error")
}

func instanceExtendsNativeError(inst *object.Instance, name string) bool {
	for current := inst.Class; current != nil; current = current.Super {
		if current.NativeConstructor != nil && isErrorClassName(current.Name) {
			return name == "Error" || current.Name == name
		}
	}
	return false
}

func runtimeErrorFromInstance(pos ast.Position, inst *object.Instance) *object.Error {
	name := inst.Class.Name
	if prop, ok := inst.Props["name"].(*object.String); ok && prop.Value != "" {
		name = prop.Value
	}
	message := ""
	if prop, ok := inst.Props["message"].(*object.String); ok {
		message = prop.Value
	}
	err := object.NewNamedError(pos, name, message)
	if prop, ok := inst.Props["stack"].(*object.String); ok && prop.Value != "" {
		err.Stack = prop.Value
	}
	err.Runtime = true
	err.Thrown = inst
	return err
}

// ============================================================================
// Ternary / Assign
// ============================================================================

func evalTernary(n *ast.TernaryExpr, env *object.Environment) object.Object {
	cond := Eval(n.Cond, env)
	if object.IsRuntimeError(cond) {
		return cond
	}
	if object.IsTruthy(cond) {
		return Eval(n.Consequent, env)
	}
	return Eval(n.Alternate, env)
}

func evalAssign(n *ast.AssignExpr, env *object.Environment) object.Object {
	right := Eval(n.Right, env)
	if object.IsRuntimeError(right) {
		return right
	}
	switch left := n.Left.(type) {
	case *ast.Ident:
		if n.Op == "=" {
			if anno, ok := env.TypeOf(left.TokenLit); ok {
				if err := checkType(env, n.Pos(), anno, right); err != nil {
					return err
				}
			}
			if _, ok, isConst := env.Assign(left.TokenLit, right); !ok {
				return object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
			} else if isConst {
				return object.NewError(left.Pos(), "TypeError: assignment to constant '%s'", left.TokenLit)
			}
		} else {
			existing, ok := env.Get(left.TokenLit)
			if !ok {
				return object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
			}
			right = evalCompoundAssign(existing, right, n.Op, n.Pos())
			if object.IsRuntimeError(right) {
				return right
			}
			if anno, ok := env.TypeOf(left.TokenLit); ok {
				if err := checkType(env, n.Pos(), anno, right); err != nil {
					return err
				}
			}
			if _, ok, isConst := env.Assign(left.TokenLit, right); !ok {
				return object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
			} else if isConst {
				return object.NewError(left.Pos(), "TypeError: assignment to constant '%s'", left.TokenLit)
			}
		}
		return right
	case *ast.MemberExpr:
		obj := Eval(left.Object, env)
		if object.IsRuntimeError(obj) {
			return obj
		}
		if hash, ok := obj.(*object.Hash); ok {
			name := left.Property.(*ast.Ident).TokenLit
			if hash.Frozen {
				return object.NewError(left.Pos(), "TypeError: cannot assign to frozen object")
			}
			if hash.Sealed {
				if _, ok := hash.Pairs[hashKey(&object.String{Value: name})]; !ok {
					return object.NewError(left.Pos(), "TypeError: cannot add property to sealed object")
				}
			}
			hash.Pairs[hashKey(&object.String{Value: name})] = object.HashPair{Key: &object.String{Value: name}, Value: right}
			return right
		}
		if inst, ok := obj.(*object.Instance); ok {
			name := left.Property.(*ast.Ident).TokenLit
			if anno, ok := inst.Class.FieldTypes[name]; ok {
				if err := checkType(env, n.Pos(), anno, right); err != nil {
					return err
				}
			}
			inst.Props[name] = right
			return right
		}
		if cls, ok := obj.(*object.Class); ok {
			name := left.Property.(*ast.Ident).TokenLit
			if anno, ok := cls.StaticTypes[name]; ok {
				if err := checkType(env, n.Pos(), anno, right); err != nil {
					return err
				}
			}
			if _, ok := cls.Statics[name]; ok {
				cls.Statics[name] = right
				return right
			}
			return object.NewError(left.Pos(), "TypeError: '%s' is not a static member of %s", name, cls.Name)
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
		if hash, ok := arr.(*object.Hash); ok {
			idx := Eval(left.Index, env)
			if hash.Frozen {
				return object.NewError(left.Pos(), "TypeError: cannot assign to frozen object")
			}
			if hash.Sealed {
				if _, ok := hash.Pairs[hashKey(idx)]; !ok {
					return object.NewError(left.Pos(), "TypeError: cannot add property to sealed object")
				}
			}
			hash.Pairs[hashKey(idx)] = object.HashPair{Key: idx, Value: right}
			return right
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
	if _, ok := n.Callee.(*ast.SuperExpr); ok {
		args := make([]object.Object, len(n.Args))
		for i, a := range n.Args {
			args[i] = Eval(a, env)
			if object.IsRuntimeError(args[i]) {
				return args[i]
			}
		}
		return callSuperConstructor(n.Pos(), env, args)
	}
	callee := Eval(n.Callee, env)
	if object.IsRuntimeError(callee) {
		return callee
	}
	args := make([]object.Object, len(n.Args))
	for i, a := range n.Args {
		args[i] = Eval(a, env)
		if object.IsRuntimeError(args[i]) {
			return args[i]
		}
	}
	return applyFunction(callee, env, args, n.Pos())
}

func applyFunction(fn object.Object, env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	switch f := fn.(type) {
	case *object.Function:
		scope := f.Env.NewScope()
		if err := bindFunctionParams(scope, env, f.Parameters, args, pos); err != nil {
			return err
		}
		if f.IsAsync {
			promise := env.ObjectManager().NewPromise()
			vm := env.VM()
			vm.AsyncAdd(1)
			vm.Go(func() {
				defer vm.AsyncDone()
				result := Eval(f.Body, scope)
				if rv, ok := result.(*object.ReturnValue); ok {
					if err := checkType(env, pos, f.ReturnT, rv.Value); err != nil {
						promise.Reject(err)
					} else {
						promise.Resolve(rv.Value)
					}
				} else if object.IsRuntimeError(result) {
					promise.Reject(result)
				} else {
					if err := checkType(env, pos, f.ReturnT, normalizeReturn(result)); err != nil {
						promise.Reject(err)
					} else {
						promise.Resolve(result)
					}
				}
			})
			return promise
		}
		result := Eval(f.Body, scope)
		if rv, ok := result.(*object.ReturnValue); ok {
			if err := checkType(env, pos, f.ReturnT, rv.Value); err != nil {
				return err
			}
			return rv.Value
		}
		if object.IsRuntimeError(result) {
			return result
		}
		if err := checkType(env, pos, f.ReturnT, normalizeReturn(result)); err != nil {
			return err
		}
		return result
	case *object.Builtin:
		env.Extra = f.Extra
		result := f.Fn(env, pos, args...)
		env.Extra = nil
		return result
	case *object.Hash:
		if promiseConstructor, ok := getHashKey(f, &object.String{Value: "__promiseConstructor"}).(*object.Boolean); ok && promiseConstructor.Value {
			return constructPromise(env, args, pos)
		}
		if call, ok := getHashKey(f, &object.String{Value: "__call"}).(*object.Builtin); ok {
			return applyFunction(call, env, args, pos)
		}
		return object.NewError(pos, "TypeError: object is not a function")
	case *object.Class:
		inst := &object.Instance{Class: f, Props: make(map[string]object.Object), Pos: pos}
		env.ObjectManager().Register(inst)
		for k, v := range f.Fields {
			inst.Props[k] = v
		}
		superCalled := false
		con, hasConstructor := f.Methods["constructor"]
		if f.Super != nil && (!hasConstructor || !constructorCallsSuper(con.Body)) {
			result := callClassConstructor(f.Super, inst, env, args, pos)
			if object.IsRuntimeError(result) {
				return result
			}
			superCalled = true
		}
		if hasConstructor {
			scope := con.Env.NewScope()
			scope.Set("this", inst)
			scope.ConstructorClass = f
			scope.SuperCalled = &superCalled
			if err := bindFunctionParams(scope, env, con.Parameters, args, pos); err != nil {
				return err
			}
			result := Eval(con.Body, scope)
			if object.IsRuntimeError(result) {
				return result
			}
		}
		return inst
	default:
		return object.NewError(pos, "TypeError: %s is not a function", fn.Type())
	}
}

func bindFunctionParams(scope, caller *object.Environment, params []*ast.Param, args []object.Object, pos ast.Position) *object.Error {
	for i, p := range params {
		var value object.Object
		if i < len(args) {
			if p.Spread {
				rest := make([]object.Object, len(args)-i)
				copy(rest, args[i:])
				value = caller.ObjectManager().NewArray(rest)
				if err := checkType(caller, pos, p.TypeAnno, value); err != nil {
					return err
				}
				scope.Set(p.Name, value)
				break
			}
			value = args[i]
		} else if p.Default != nil {
			value = Eval(p.Default, scope)
			if object.IsRuntimeError(value) {
				if err, ok := value.(*object.Error); ok {
					return err
				}
				return object.NewError(pos, "%s", value.Inspect())
			}
		} else {
			value = object.UNDEFINED
		}
		if err := checkType(caller, pos, p.TypeAnno, value); err != nil {
			return err
		}
		scope.Set(p.Name, value)
	}
	return nil
}

func callClassConstructor(cls *object.Class, inst *object.Instance, env *object.Environment, args []object.Object, pos ast.Position) object.Object {
	if cls.NativeConstructor != nil {
		return cls.NativeConstructor(env, inst, pos, args)
	}
	con, ok := cls.Methods["constructor"]
	if !ok {
		return object.UNDEFINED
	}
	scope := con.Env.NewScope()
	scope.Set("this", inst)
	scope.ConstructorClass = cls
	if err := bindFunctionParams(scope, env, con.Parameters, args, pos); err != nil {
		return err
	}
	result := Eval(con.Body, scope)
	if rv, ok := result.(*object.ReturnValue); ok {
		return rv.Value
	}
	return result
}

func constructorCallsSuper(block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.Statements {
		if nodeCallsSuper(stmt) {
			return true
		}
	}
	return false
}

func nodeCallsSuper(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.ExprStmt:
		return nodeCallsSuper(n.Expr)
	case *ast.CallExpr:
		if _, ok := n.Callee.(*ast.SuperExpr); ok {
			return true
		}
		if nodeCallsSuper(n.Callee) {
			return true
		}
		for _, arg := range n.Args {
			if nodeCallsSuper(arg) {
				return true
			}
		}
	case *ast.BlockStmt:
		for _, stmt := range n.Statements {
			if nodeCallsSuper(stmt) {
				return true
			}
		}
	case *ast.IfStmt:
		return nodeCallsSuper(n.Cond) || nodeCallsSuper(n.Consequence) || nodeCallsSuper(n.Alternative)
	case *ast.ReturnStmt:
		return nodeCallsSuper(n.Value)
	case *ast.AssignExpr:
		return nodeCallsSuper(n.Left) || nodeCallsSuper(n.Right)
	case *ast.MemberExpr:
		return nodeCallsSuper(n.Object) || nodeCallsSuper(n.Property)
	case *ast.IndexExpr:
		return nodeCallsSuper(n.Left) || nodeCallsSuper(n.Index)
	case *ast.ArrayLit:
		for _, elem := range n.Elements {
			if nodeCallsSuper(elem) {
				return true
			}
		}
	case *ast.ObjectLit:
		for _, prop := range n.Properties {
			if nodeCallsSuper(prop.Key) || nodeCallsSuper(prop.Value) {
				return true
			}
		}
	case *ast.InfixExpr:
		return nodeCallsSuper(n.Left) || nodeCallsSuper(n.Right)
	case *ast.PrefixExpr:
		return nodeCallsSuper(n.Right)
	case *ast.TernaryExpr:
		return nodeCallsSuper(n.Cond) || nodeCallsSuper(n.Consequent) || nodeCallsSuper(n.Alternate)
	case *ast.NewExpr:
		if nodeCallsSuper(n.Callee) {
			return true
		}
		for _, arg := range n.Args {
			if nodeCallsSuper(arg) {
				return true
			}
		}
	case *ast.AwaitExpr:
		return nodeCallsSuper(n.Value)
	}
	return false
}

func normalizeReturn(value object.Object) object.Object {
	if value == nil {
		return object.UNDEFINED
	}
	return value
}

func evalMember(n *ast.MemberExpr, env *object.Environment) object.Object {
	if super, ok := n.Object.(*ast.SuperExpr); ok {
		if prop, ok := n.Property.(*ast.Ident); ok {
			return evalSuper(&ast.SuperExpr{Pos_: super.Pos_, TokenLit: super.TokenLit, Method: prop.TokenLit}, env)
		}
	}
	obj := Eval(n.Object, env)
	if object.IsRuntimeError(obj) {
		return obj
	}
	prop := n.Property.(*ast.Ident).TokenLit
	return getProperty(obj, prop, n.Pos())
}

func evalIndex(n *ast.IndexExpr, env *object.Environment) object.Object {
	left := Eval(n.Left, env)
	if object.IsRuntimeError(left) {
		return left
	}
	idx := Eval(n.Index, env)
	if object.IsRuntimeError(idx) {
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
		return object.NewError(pos, "TypeError: '%s' is not a property of %s", name, o.Class.Name)
	case *object.Class:
		if v, ok := o.Statics[name]; ok {
			return v
		}
		if m, ok := o.Methods[name]; ok {
			return m
		}
		return object.NewError(pos, "TypeError: '%s' is not a static member of %s", name, o.Name)
	case *object.Error:
		switch name {
		case "name":
			errName := o.Name
			if errName == "" {
				errName = "Error"
			}
			return &object.String{Value: errName}
		case "message":
			return &object.String{Value: o.Message}
		case "stack":
			stack := o.Stack
			if stack == "" {
				stack = o.FormatStack()
			}
			return &object.String{Value: stack}
		}
		return object.UNDEFINED
	case *object.String:
		switch name {
		case "length":
			return &object.Number{Value: float64(len(o.Value))}
		default:
			if fn, ok := stringMethods[name]; ok {
				return &object.Builtin{Name: "String." + name, Fn: fn, Extra: o}
			}
		}
	case *object.Number:
		if fn, ok := numberMethods[name]; ok {
			return &object.Builtin{Name: "Number." + name, Fn: fn, Extra: o}
		}
	case *object.Date:
		if fn, ok := dateMethods[name]; ok {
			return &object.Builtin{Name: "Date." + name, Fn: fn, Extra: o}
		}
	case *object.RegExp:
		switch name {
		case "source":
			return &object.String{Value: o.Source}
		case "flags":
			return &object.String{Value: o.Flags}
		case "global":
			return object.NativeBool(strings.Contains(o.Flags, "g"))
		case "ignoreCase":
			return object.NativeBool(strings.Contains(o.Flags, "i"))
		default:
			if fn, ok := regexpMethods[name]; ok {
				return &object.Builtin{Name: "RegExp." + name, Fn: fn, Extra: o}
			}
		}
	case *object.BooleanObject:
		if fn, ok := booleanObjectMethods[name]; ok {
			return &object.Builtin{Name: "Boolean." + name, Fn: fn, Extra: o}
		}
	case *object.Array:
		switch name {
		case "length":
			return &object.Number{Value: float64(len(o.Elements))}
		default:
			if fn, ok := arrayMethods[name]; ok {
				return &object.Builtin{Name: "Array." + name, Fn: fn, Extra: o}
			}
		}
	case *object.Map:
		switch name {
		case "size":
			return &object.Number{Value: float64(len(o.Entries))}
		default:
			if fn, ok := mapMethods[name]; ok {
				return &object.Builtin{Name: "Map." + name, Fn: fn, Extra: o}
			}
		}
	case *object.Set:
		switch name {
		case "size":
			return &object.Number{Value: float64(len(o.Values))}
		default:
			if fn, ok := setMethods[name]; ok {
				return &object.Builtin{Name: "Set." + name, Fn: fn, Extra: o}
			}
		}
	case *object.Promise:
		if fn, ok := promiseMethods[name]; ok {
			return &object.Builtin{Name: "Promise." + name, Fn: fn, Extra: o}
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
		if o.Proto != nil {
			return getHashKey(o.Proto, key)
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
