package object

// VirtualMachine owns the runtime object space for one script execution.
//
// Environments inside the same VM share this manager. Separate VM instances get
// separate managers, so their runtime objects, ids, and stats stay independent.
type VirtualMachine struct {
	manager *ObjectManager
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
	return NewEnvironmentWithManager(vm.ObjectManager())
}
