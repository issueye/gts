package object

import (
	"sync"
	"sync/atomic"

	"github.com/issueye/goscript/internal/async"
)

// VirtualMachine owns the runtime object space for one script execution.
//
// Environments inside the same VM share this manager. Separate VM instances get
// separate managers, so their runtime objects, ids, and stats stay independent.
type VirtualMachine struct {
	manager         *ObjectManager
	asyncWG         sync.WaitGroup
	nextTimer       int64
	globalConstants atomic.Value // stores map[string]Object
	globalMu        sync.Mutex
	typeCheck       atomic.Bool
	spawn           func(func())
	importer        func(env *Environment, path string) (Object, error)
	evaluator       func(node interface{}, env *Environment) Object
	argv            atomic.Value // stores []string
}

func NewVirtualMachine() *VirtualMachine {
	vm := &VirtualMachine{}
	vm.Reset()
	return vm
}

func (vm *VirtualMachine) ObjectManager() *ObjectManager {
	if vm.manager == nil {
		vm.manager = newObjectManager(false)
	}
	return vm.manager
}

func (vm *VirtualMachine) Reset() {
	if vm == nil {
		return
	}
	vm.WaitAsync()
	tracking := vm.ObjectTracking()
	vm.manager = newObjectManager(tracking)
	atomic.StoreInt64(&vm.nextTimer, 0)
	vm.globalMu.Lock()
	vm.globalConstants.Store(map[string]Object{})
	vm.globalMu.Unlock()
	vm.spawn = nil
	vm.importer = nil
	vm.evaluator = nil
	vm.typeCheck.Store(false)
	vm.argv.Store([]string{})
}

func (vm *VirtualMachine) SetObjectTracking(enabled bool) {
	if vm == nil {
		return
	}
	vm.ObjectManager().SetTracking(enabled)
}

func (vm *VirtualMachine) ObjectTracking() bool {
	return vm != nil && vm.ObjectManager().Tracking()
}

func (vm *VirtualMachine) SetTypeCheck(enabled bool) {
	if vm == nil {
		return
	}
	vm.typeCheck.Store(enabled)
}

func (vm *VirtualMachine) TypeCheck() bool {
	return vm != nil && vm.typeCheck.Load()
}

func (vm *VirtualMachine) globalConstantMap() map[string]Object {
	if vm == nil {
		return nil
	}
	if constants, ok := vm.globalConstants.Load().(map[string]Object); ok {
		return constants
	}
	empty := map[string]Object{}
	vm.globalConstants.Store(empty)
	return empty
}

// SetGlobalConst registers a VM-level read-only binding.
//
// Global constants are visible to every environment in the VM after local and
// parent scopes are checked. They are copy-on-write so reads stay lock-free on
// the identifier hot path.
func (vm *VirtualMachine) SetGlobalConst(name string, val Object) Object {
	if vm == nil || name == "" {
		return val
	}
	vm.globalMu.Lock()
	defer vm.globalMu.Unlock()

	current := vm.globalConstantMap()
	next := make(map[string]Object, len(current)+1)
	for key, value := range current {
		next[key] = value
	}
	next[name] = val
	vm.globalConstants.Store(next)
	return val
}

func (vm *VirtualMachine) SetGlobalConsts(values map[string]Object) {
	if vm == nil || len(values) == 0 {
		return
	}
	vm.globalMu.Lock()
	defer vm.globalMu.Unlock()

	current := vm.globalConstantMap()
	next := make(map[string]Object, len(current)+len(values))
	for key, value := range current {
		next[key] = value
	}
	for key, value := range values {
		if key != "" {
			next[key] = value
		}
	}
	vm.globalConstants.Store(next)
}

func (vm *VirtualMachine) GetGlobalConst(name string) (Object, bool) {
	constants := vm.globalConstantMap()
	obj, ok := constants[name]
	return obj, ok
}

func (vm *VirtualMachine) HasGlobalConst(name string) bool {
	_, ok := vm.GetGlobalConst(name)
	return ok
}

// GlobalConstants returns a stable snapshot of VM-level read-only bindings.
func (vm *VirtualMachine) GlobalConstants() map[string]Object {
	current := vm.globalConstantMap()
	out := make(map[string]Object, len(current))
	for key, value := range current {
		out[key] = value
	}
	return out
}

func (vm *VirtualMachine) NewEnvironment() *Environment {
	return NewEnvironmentWithVM(vm)
}

func (vm *VirtualMachine) SetArgv(argv []string) {
	if vm == nil {
		return
	}
	vm.argv.Store(append([]string{}, argv...))
}

func (vm *VirtualMachine) Argv() []string {
	if vm == nil {
		return nil
	}
	if argv, ok := vm.argv.Load().([]string); ok {
		return append([]string{}, argv...)
	}
	return nil
}

func (vm *VirtualMachine) SetSpawner(spawn func(func())) {
	vm.spawn = spawn
}

func (vm *VirtualMachine) Go(fn func()) {
	if vm.spawn != nil {
		vm.spawn(fn)
		return
	}
	go func() {
		defer async.RecoverPanic("virtual machine task")
		fn()
	}()
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

func (vm *VirtualMachine) HasEvaluator() bool {
	return vm != nil && vm.evaluator != nil
}

func (vm *VirtualMachine) EvalNode(node interface{}, env *Environment) Object {
	if vm.evaluator == nil {
		return NewError(env.Pos, "RuntimeError: evaluator is not configured")
	}
	return vm.evaluator(node, env)
}
