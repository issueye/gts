package object

import "runtime"

// VirtualMachinePool reuses reset VMs to reduce repeated allocation cost.
type VirtualMachinePool struct {
	idle chan *VirtualMachine
}

func NewVirtualMachinePool(size int) *VirtualMachinePool {
	if size <= 0 {
		size = runtime.NumCPU()
	}
	if size < 1 {
		size = 1
	}
	return &VirtualMachinePool{idle: make(chan *VirtualMachine, size)}
}

func (p *VirtualMachinePool) Get() *VirtualMachine {
	if p == nil {
		return NewVirtualMachine()
	}
	select {
	case vm := <-p.idle:
		return vm
	default:
		return NewVirtualMachine()
	}
}

func (p *VirtualMachinePool) Put(vm *VirtualMachine) {
	if p == nil || vm == nil {
		return
	}
	vm.Reset()
	select {
	case p.idle <- vm:
	default:
	}
}

func (p *VirtualMachinePool) Len() int {
	if p == nil {
		return 0
	}
	return len(p.idle)
}
