package object

import "testing"

func TestVirtualMachineResetClearsRuntimeState(t *testing.T) {
	vm := NewVirtualMachine()
	vm.SetTypeCheck(true)
	vm.SetObjectTracking(true)
	vm.SetGlobalConst("answer", &Number{Value: 42})
	vm.SetImportFunc(func(env *Environment, path string) (Object, error) {
		return &String{Value: path}, nil
	})
	vm.SetEvaluator(func(node interface{}, env *Environment) Object {
		return &String{Value: "eval"}
	})
	if got := vm.NextTimerID(); got != 1 {
		t.Fatalf("want first timer id 1, got %d", got)
	}
	tracked := vm.ObjectManager().NewArray(nil)
	if _, ok := vm.ObjectManager().IDOf(tracked); !ok {
		t.Fatal("expected tracked object before reset")
	}

	vm.Reset()

	if vm.TypeCheck() {
		t.Fatal("reset should clear type checking")
	}
	if !vm.ObjectTracking() {
		t.Fatal("reset should preserve object tracking preference")
	}
	if _, ok := vm.GetGlobalConst("answer"); ok {
		t.Fatal("reset should clear global constants")
	}
	if got := vm.NextTimerID(); got != 1 {
		t.Fatalf("reset should restart timer ids, got %d", got)
	}
	if result, err := vm.Import(vm.NewEnvironment(), "mod"); err != nil || result != nil {
		t.Fatalf("reset should clear importer, got result=%v err=%v", result, err)
	}
	if result := vm.EvalNode(nil, vm.NewEnvironment()); result == nil || result.Type() != ERROR_OBJ {
		t.Fatalf("reset should clear evaluator, got %T", result)
	}
	if stats := vm.ObjectManager().Stats(); stats.Active != 0 || stats.TotalAllocated != 0 {
		t.Fatalf("reset should replace tracked object state, got %+v", stats)
	}
}

func TestVirtualMachinePoolReusesResetVMs(t *testing.T) {
	pool := NewVirtualMachinePool(1)
	vm := pool.Get()
	vm.SetTypeCheck(true)
	vm.SetGlobalConst("value", &Number{Value: 1})
	pool.Put(vm)

	if pool.Len() != 1 {
		t.Fatalf("want one idle vm, got %d", pool.Len())
	}
	reused := pool.Get()
	if reused != vm {
		t.Fatal("pool should reuse idle VM")
	}
	if reused.TypeCheck() {
		t.Fatal("pooled VM should be reset before reuse")
	}
	if _, ok := reused.GetGlobalConst("value"); ok {
		t.Fatal("pooled VM should not retain global constants")
	}
}
