package evaluator

import (
	"math"
	"strconv"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

// ============================================================================
// Prefix / Infix Expressions
// ============================================================================

func evalPrefix(n *ast.PrefixExpr, env *object.Environment) object.Object {
	if n.Op == "++" || n.Op == "--" {
		return evalUpdate(n.Right, env, n.Op, false, n.Pos())
	}

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
		return &object.String{Value: typeofName(right)}
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

func typeofName(value object.Object) string {
	switch value.(type) {
	case *object.Undefined:
		return "undefined"
	case *object.Null:
		return "object"
	case *object.Boolean:
		return "boolean"
	case *object.Number:
		return "number"
	case *object.String:
		return "string"
	case *object.Function, *object.Builtin, *object.Class:
		return "function"
	default:
		return "object"
	}
}

func evalInfix(n *ast.InfixExpr, env *object.Environment) object.Object {
	if (n.Op == "++" || n.Op == "--") && n.Right == nil {
		return evalUpdate(n.Left, env, n.Op, true, n.Pos())
	}

	left := Eval(n.Left, env)
	if object.IsRuntimeError(left) {
		return left
	}

	switch n.Op {
	case "&&":
		if !object.IsTruthy(left) {
			return left
		}
		right := Eval(n.Right, env)
		if object.IsRuntimeError(right) {
			return right
		}
		return right
	case "||":
		if object.IsTruthy(left) {
			return left
		}
		right := Eval(n.Right, env)
		if object.IsRuntimeError(right) {
			return right
		}
		return right
	case "??":
		if left != object.NULL && left != object.UNDEFINED {
			return left
		}
		right := Eval(n.Right, env)
		if object.IsRuntimeError(right) {
			return right
		}
		return right
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
			hash.SetMember(&object.String{Value: name}, right)
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
			hash.SetMember(idx, right)
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

func evalUpdate(target ast.Expression, env *object.Environment, op string, postfix bool, pos ast.Position) object.Object {
	current, err := readUpdateTarget(target, env, pos)
	if err != nil {
		return err
	}

	num, ok := current.(*object.Number)
	if !ok {
		return object.NewError(pos, "TypeError: update operator requires number, got %s", current.Type())
	}

	delta := 1.0
	if op == "--" {
		delta = -1
	}
	next := &object.Number{Value: num.Value + delta}
	if err := writeUpdateTarget(target, env, next, pos); err != nil {
		return err
	}

	if postfix {
		return current
	}
	return next
}

func readUpdateTarget(target ast.Expression, env *object.Environment, pos ast.Position) (object.Object, *object.Error) {
	switch left := target.(type) {
	case *ast.Ident:
		value, ok := env.Get(left.TokenLit)
		if !ok {
			return nil, object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
		}
		return value, nil
	case *ast.MemberExpr:
		obj := Eval(left.Object, env)
		if object.IsRuntimeError(obj) {
			return nil, obj.(*object.Error)
		}
		name := left.Property.(*ast.Ident).TokenLit
		return getProperty(obj, name, left.Pos()), nil
	case *ast.IndexExpr:
		obj := Eval(left.Left, env)
		if object.IsRuntimeError(obj) {
			return nil, obj.(*object.Error)
		}
		idx := Eval(left.Index, env)
		if object.IsRuntimeError(idx) {
			return nil, idx.(*object.Error)
		}
		switch o := obj.(type) {
		case *object.Array:
			if num, ok := idx.(*object.Number); ok {
				i := int(num.Value)
				if i >= 0 && i < len(o.Elements) {
					return o.Elements[i], nil
				}
			}
			return object.UNDEFINED, nil
		case *object.Hash:
			return getHashKey(o, idx), nil
		default:
			return nil, object.NewError(left.Pos(), "TypeError: cannot index %s", obj.Type())
		}
	default:
		return nil, object.NewError(pos, "SyntaxError: invalid update target")
	}
}

func writeUpdateTarget(target ast.Expression, env *object.Environment, value object.Object, pos ast.Position) *object.Error {
	switch left := target.(type) {
	case *ast.Ident:
		if _, ok, isConst := env.Assign(left.TokenLit, value); !ok {
			return object.NewError(left.Pos(), "ReferenceError: '%s' is not defined", left.TokenLit)
		} else if isConst {
			return object.NewError(left.Pos(), "TypeError: assignment to constant '%s'", left.TokenLit)
		}
		return nil
	case *ast.MemberExpr:
		obj := Eval(left.Object, env)
		if object.IsRuntimeError(obj) {
			return obj.(*object.Error)
		}
		name := left.Property.(*ast.Ident).TokenLit
		if hash, ok := obj.(*object.Hash); ok {
			if hash.Frozen {
				return object.NewError(left.Pos(), "TypeError: cannot assign to frozen object")
			}
			if hash.Sealed {
				if _, ok := hash.Pairs[hashKey(&object.String{Value: name})]; !ok {
					return object.NewError(left.Pos(), "TypeError: cannot add property to sealed object")
				}
			}
			hash.SetMember(&object.String{Value: name}, value)
			return nil
		}
		if inst, ok := obj.(*object.Instance); ok {
			inst.Props[name] = value
			return nil
		}
		return object.NewError(left.Pos(), "TypeError: cannot assign to property of %T", obj)
	case *ast.IndexExpr:
		obj := Eval(left.Left, env)
		if object.IsRuntimeError(obj) {
			return obj.(*object.Error)
		}
		idx := Eval(left.Index, env)
		if object.IsRuntimeError(idx) {
			return idx.(*object.Error)
		}
		if arr, ok := obj.(*object.Array); ok {
			if num, ok := idx.(*object.Number); ok {
				i := int(num.Value)
				if i >= 0 && i < len(arr.Elements) {
					arr.Elements[i] = value
					return nil
				}
			}
			return object.NewError(left.Pos(), "TypeError: array index must be number")
		}
		if hash, ok := obj.(*object.Hash); ok {
			if hash.Frozen {
				return object.NewError(left.Pos(), "TypeError: cannot assign to frozen object")
			}
			if hash.Sealed {
				if _, ok := hash.Pairs[hashKey(idx)]; !ok {
					return object.NewError(left.Pos(), "TypeError: cannot add property to sealed object")
				}
			}
			hash.SetMember(idx, value)
			return nil
		}
		return object.NewError(left.Pos(), "TypeError: cannot index %s", obj.Type())
	default:
		return object.NewError(pos, "SyntaxError: invalid update target")
	}
}
