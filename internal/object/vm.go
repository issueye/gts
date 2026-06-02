package object

import (
	"sync"
	"sync/atomic"
)

// VirtualMachine owns the runtime object space for one script execution.
//
// Environments inside the same VM share this manager. Separate VM instances get
// separate managers, so their runtime objects, ids, and stats stay independent.
type VirtualMachine struct {
	manager   *ObjectManager
	asyncWG   sync.WaitGroup
	nextTimer int64
	spawn     func(func())
	importer  func(env *Environment, path string) (Object, error)
	evaluator func(node interface{}, env *Environment) Object
}

func NewVirtualMachine() *VirtualMachine {
	return &VirtualMachine{manager: NewObjectManager()}
}

func (vm *VirtualMachine) ObjectManager() *ObjectManager {
	if vm.manager == nil {
		vm.manager = NewObjectManager()
	}
	return vm.manager
}

func (vm *VirtualMachine) NewEnvironment() *Environment {
	return NewEnvironmentWithVM(vm)
}

func (vm *VirtualMachine) SetSpawner(spawn func(func())) {
	vm.spawn = spawn
}

func (vm *VirtualMachine) Go(fn func()) {
	if vm.spawn != nil {
		vm.spawn(fn)
		return
	}
	go fn()
}

func (vm *VirtualMachine) AsyncAdd(delta int) {
	vm.asyncWG.Add(delta)
}

func (vm *VirtualMachine) AsyncDone() {
	vm.asyncWG.Done()
}

func (vm *VirtualMachine) WaitAsync() {
	vm.asyncWG.Wait()
}

func (vm *VirtualMachine) NextTimerID() int64 {
	return atomic.AddInt64(&vm.nextTimer, 1)
}

func (vm *VirtualMachine) SetImportFunc(fn func(env *Environment, path string) (Object, error)) {
	vm.importer = fn
}

func (vm *VirtualMachine) Import(env *Environment, path string) (Object, error) {
	if vm.importer == nil {
		return nil, nil
	}
	return vm.importer(env, path)
}

func (vm *VirtualMachine) SetEvaluator(fn func(node interface{}, env *Environment) Object) {
	vm.evaluator = fn
}

func (vm *VirtualMachine) EvalNode(node interface{}, env *Environment) Object {
	if vm.evaluator == nil {
		return NewError(env.Pos, "RuntimeError: evaluator is not configured")
	}
	return vm.evaluator(node, env)
}
