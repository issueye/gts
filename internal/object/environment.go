package object

import (
	"sync"

	"github.com/issueye/goscript/internal/ast"
)

// Environment is a scope for variable bindings.
type Environment struct {
	mu               sync.RWMutex
	store            map[string]Object
	consts           map[string]bool
	types            map[string]*ast.TypeAnnotation
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
	return &Environment{
		vm: vm,
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
	for env := e; env != nil; env = env.parent {
		env.mu.RLock()
		if obj, ok := env.store[name]; ok {
			env.mu.RUnlock()
			return obj, true
		}
		env.mu.RUnlock()
	}
	return e.VM().GetGlobalConst(name)
}

func (e *Environment) Set(name string, val Object) Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.store == nil {
		e.store = make(map[string]Object)
	}
	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = false
	if e.types != nil {
		delete(e.types, name)
	}
	return val
}

func (e *Environment) SetConst(name string, val Object) Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.store == nil {
		e.store = make(map[string]Object)
	}
	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = true
	if e.types != nil {
		delete(e.types, name)
	}
	return val
}

func (e *Environment) SetTyped(name string, val Object, anno *ast.TypeAnnotation) Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.store == nil {
		e.store = make(map[string]Object)
	}
	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = false
	if anno == nil {
		if e.types != nil {
			delete(e.types, name)
		}
		return val
	}
	if e.types == nil {
		e.types = make(map[string]*ast.TypeAnnotation)
	}
	e.types[name] = anno
	return val
}

func (e *Environment) SetTypedConst(name string, val Object, anno *ast.TypeAnnotation) Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.store == nil {
		e.store = make(map[string]Object)
	}
	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = true
	if anno == nil {
		if e.types != nil {
			delete(e.types, name)
		}
		return val
	}
	if e.types == nil {
		e.types = make(map[string]*ast.TypeAnnotation)
	}
	e.types[name] = anno
	return val
}

func (e *Environment) TypeOf(name string) (*ast.TypeAnnotation, bool) {
	for env := e; env != nil; env = env.parent {
		env.mu.RLock()
		if _, ok := env.store[name]; ok {
			anno, typed := env.types[name]
			env.mu.RUnlock()
			return anno, typed && anno != nil
		}
		env.mu.RUnlock()
	}
	return nil, false
}

// SetUp sets a variable in the nearest ancestor scope where it exists,
// creating it in the current scope if not found anywhere.
func (e *Environment) SetUp(name string, val Object) Object {
	for env := e; env != nil; env = env.parent {
		env.mu.Lock()
		if _, ok := env.store[name]; ok {
			if env.consts[name] {
				env.mu.Unlock()
				return nil
			}
			env.store[name] = val
			env.mu.Unlock()
			return val
		}
		if env.parent == nil {
			if env.VM().HasGlobalConst(name) {
				env.mu.Unlock()
				return nil
			}
			if env.store == nil {
				env.store = make(map[string]Object)
			}
			if env.consts == nil {
				env.consts = make(map[string]bool)
			}
			env.store[name] = val
			env.consts[name] = false
			if env.types != nil {
				delete(env.types, name)
			}
			env.mu.Unlock()
			return val
		}
		env.mu.Unlock()
	}
	return val
}

func (e *Environment) Assign(name string, val Object) (Object, bool, bool) {
	for env := e; env != nil; env = env.parent {
		env.mu.Lock()
		if _, ok := env.store[name]; ok {
			if env.consts[name] {
				env.mu.Unlock()
				return nil, true, true
			}
			env.store[name] = val
			env.mu.Unlock()
			return val, true, false
		}
		env.mu.Unlock()
	}
	if e.VM().HasGlobalConst(name) {
		return nil, true, true
	}
	return nil, false, false
}

// SetHere sets only in this environment (not parent).
func (e *Environment) SetHere(name string, val Object) Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.store == nil {
		e.store = make(map[string]Object)
	}
	if e.consts == nil {
		e.consts = make(map[string]bool)
	}
	e.store[name] = val
	e.consts[name] = false
	if e.types != nil {
		delete(e.types, name)
	}
	return val
}

// Has checks if name exists in this environment.
func (e *Environment) Has(name string) bool {
	for env := e; env != nil; env = env.parent {
		env.mu.RLock()
		if _, ok := env.store[name]; ok {
			env.mu.RUnlock()
			return true
		}
		env.mu.RUnlock()
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
	e.mu.RLock()
	defer e.mu.RUnlock()
	keys := make([]string, 0, len(e.store))
	for k := range e.store {
		keys = append(keys, k)
	}
	return keys
}
