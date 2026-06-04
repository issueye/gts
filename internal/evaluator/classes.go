package evaluator

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

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
