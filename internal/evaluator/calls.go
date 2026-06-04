package evaluator

import (
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

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
		if value, ok := getHashKeyOk(o, &object.String{Value: name}); ok {
			return value
		}
		if name == "toString" {
			return &object.Builtin{Name: "Object.toString", Fn: builtinNativeToString, Extra: o}
		}
		return object.UNDEFINED
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
	case *object.Boolean:
		if name == "toString" {
			return &object.Builtin{Name: "Boolean.toString", Fn: builtinNativeToString, Extra: o}
		}
	case *object.Null:
		if name == "toString" {
			return &object.Builtin{Name: "Null.toString", Fn: builtinNativeToString, Extra: o}
		}
	case *object.Undefined:
		if name == "toString" {
			return &object.Builtin{Name: "Undefined.toString", Fn: builtinNativeToString, Extra: o}
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
	value, ok := getHashKeyOk(obj, key)
	if !ok {
		return object.UNDEFINED
	}
	return value
}

func getHashKeyOk(obj object.Object, key object.Object) (object.Object, bool) {
	switch o := obj.(type) {
	case *object.Hash:
		if pair, ok := o.Pairs[hashKey(key)]; ok {
			return pair.Value, true
		}
		if o.Proto != nil {
			return getHashKeyOk(o.Proto, key)
		}
		return object.UNDEFINED, false
	default:
		return object.UNDEFINED, false
	}
}

func hashKey(o object.Object) object.HashKey {
	return object.HashKeyFor(o)
}
