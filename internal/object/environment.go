package object

import (
	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/safemap"
)

type envBinding struct {
	Value Object
	Const bool
	Type  *ast.TypeAnnotation
}

// Environment is a scope for variable bindings.
type Environment struct {
	bindings         safemap.SafeSortedMap[string, envBinding]
	parent           *Environment
	vm               *VirtualMachine
	Extra            Object // bound context for method dispatch (array/string instance)
	ConstructorClass *Class
	SuperCalled      *bool
	ModuleDir        string
	Pos              ast.Position
}

func NewEnvironment() *Environment {
	return NewVirtualMachine().NewEnvironment()
}

func NewEnvironmentWithVM(vm *VirtualMachine) *Environment {
	if vm == nil {
		vm = NewVirtualMachine()
	}
	env := &Environment{vm: vm}
	env.bindings.SetLess(func(a, b string) bool { return a < b })
	return env
}

func (e *Environment) VM() *VirtualMachine {
	if e.vm != nil {
		return e.vm
	}
	if e.parent != nil {
		return e.parent.VM()
	}
	e.vm = NewVirtualMachine()
	return e.vm
}

func (e *Environment) ObjectManager() *ObjectManager {
	return e.VM().ObjectManager()
}

func (e *Environment) Get(name string) (Object, bool) {
	for env := e; env != nil; env = env.parent {
		if binding, ok := env.bindings.Get(name); ok {
			return binding.Value, true
		}
	}
	return e.VM().GetGlobalConst(name)
}

func (e *Environment) Set(name string, val Object) Object {
	e.bindings.Set(name, envBinding{Value: val})
	return val
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.bindings.Set(name, envBinding{Value: val, Const: true})
	return val
}

func (e *Environment) SetTyped(name string, val Object, anno *ast.TypeAnnotation) Object {
	e.bindings.Set(name, envBinding{Value: val, Type: anno})
	return val
}

func (e *Environment) SetTypedConst(name string, val Object, anno *ast.TypeAnnotation) Object {
	e.bindings.Set(name, envBinding{Value: val, Const: true, Type: anno})
	return val
}

func (e *Environment) TypeOf(name string) (*ast.TypeAnnotation, bool) {
	for env := e; env != nil; env = env.parent {
		if binding, ok := env.bindings.Get(name); ok {
			return binding.Type, binding.Type != nil
		}
	}
	return nil, false
}

// SetUp sets a variable in the nearest ancestor scope where it exists,
// creating it in the current scope if not found anywhere.
func (e *Environment) SetUp(name string, val Object) Object {
	for env := e; env != nil; env = env.parent {
		if binding, ok := env.bindings.Get(name); ok {
			if binding.Const {
				return nil
			}
			binding.Value = val
			binding.Type = nil
			env.bindings.Set(name, binding)
			return val
		}
		if env.parent == nil {
			if env.VM().HasGlobalConst(name) {
				return nil
			}
			env.bindings.Set(name, envBinding{Value: val})
			return val
		}
	}
	return val
}

func (e *Environment) Assign(name string, val Object) (Object, bool, bool) {
	for env := e; env != nil; env = env.parent {
		if binding, ok := env.bindings.Get(name); ok {
			if binding.Const {
				return nil, true, true
			}
			binding.Value = val
			env.bindings.Set(name, binding)
			return val, true, false
		}
	}
	if e.VM().HasGlobalConst(name) {
		return nil, true, true
	}
	return nil, false, false
}

// SetHere sets only in this environment (not parent).
func (e *Environment) SetHere(name string, val Object) Object {
	e.bindings.Set(name, envBinding{Value: val})
	return val
}

// Has checks if name exists in this environment.
func (e *Environment) Has(name string) bool {
	for env := e; env != nil; env = env.parent {
		if env.bindings.Has(name) {
			return true
		}
	}
	return e.VM().HasGlobalConst(name)
}

// NewScope creates a child environment (for block scope).
func (e *Environment) NewScope() *Environment {
	env := NewEnvironmentWithVM(e.VM())
	env.parent = e
	env.ConstructorClass = e.ConstructorClass
	env.SuperCalled = e.SuperCalled
	env.ModuleDir = e.ModuleDir
	return env
}

// Parent returns the parent environment.
func (e *Environment) Parent() *Environment {
	return e.parent
}

// All returns all keys in this scope (not parent).
func (e *Environment) Keys() []string {
	return e.bindings.SortedKeys()
}
