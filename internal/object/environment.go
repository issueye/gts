package object

import "github.com/issueye/goscript/internal/ast"

// Environment is a scope for variable bindings.
type Environment struct {
	store  map[string]Object
	consts map[string]bool
	parent *Environment
	vm     *VirtualMachine
	Extra  Object // bound context for method dispatch (array/string instance)
	Pos    ast.Position
}

func NewEnvironment() *Environment {
	return NewVirtualMachine().NewEnvironment()
}

func NewEnvironmentWithVM(vm *VirtualMachine) *Environment {
	if vm == nil {
		vm = NewVirtualMachine()
	}
	return &Environment{
		store:  make(map[string]Object),
		consts: make(map[string]bool),
		vm:     vm,
	}
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
	obj, ok := e.store[name]
	if !ok && e.parent != nil {
		return e.parent.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = false
	return val
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = true
	return val
}

// SetUp sets a variable in the nearest ancestor scope where it exists,
// creating it in the current scope if not found anywhere.
func (e *Environment) SetUp(name string, val Object) Object {
	if _, ok := e.store[name]; ok {
		if e.consts[name] {
			return nil
		}
		e.store[name] = val
		return val
	}
	if e.parent != nil {
		return e.parent.SetUp(name, val)
	}
	e.store[name] = val
	e.consts[name] = false
	return val
}

func (e *Environment) Assign(name string, val Object) (Object, bool, bool) {
	if _, ok := e.store[name]; ok {
		if e.consts[name] {
			return nil, true, true
		}
		e.store[name] = val
		return val, true, false
	}
	if e.parent != nil {
		return e.parent.Assign(name, val)
	}
	return nil, false, false
}

// SetHere sets only in this environment (not parent).
func (e *Environment) SetHere(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = false
	return val
}

// Has checks if name exists in this environment.
func (e *Environment) Has(name string) bool {
	_, ok := e.store[name]
	if !ok && e.parent != nil {
		return e.parent.Has(name)
	}
	return ok
}

// NewScope creates a child environment (for block scope).
func (e *Environment) NewScope() *Environment {
	env := NewEnvironmentWithVM(e.VM())
	env.parent = e
	return env
}

// Parent returns the parent environment.
func (e *Environment) Parent() *Environment {
	return e.parent
}

// All returns all keys in this scope (not parent).
func (e *Environment) Keys() []string {
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	return keys
}
