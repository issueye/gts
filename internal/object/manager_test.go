package object

import (
	"testing"
	"time"
)

func TestObjectManagerRegisterAndStats(t *testing.T) {
	manager := NewObjectManager()
	arr := manager.NewArray([]Object{&Number{Value: 1}})
	hash := manager.NewHash()

	arrID, ok := manager.IDOf(arr)
	if !ok {
		t.Fatal("array should have a manager id")
	}
	if duplicate := manager.Register(arr); duplicate != arrID {
		t.Fatalf("duplicate registration changed id: want %d, got %d", arrID, duplicate)
	}
	if _, ok := manager.IDOf(hash); !ok {
		t.Fatal("hash should have a manager id")
	}

	stats := manager.Stats()
	if stats.TotalAllocated != 2 {
		t.Fatalf("want 2 total allocations, got %d", stats.TotalAllocated)
	}
	if stats.Active != 2 {
		t.Fatalf("want 2 active objects, got %d", stats.Active)
	}
	if stats.ByType[ARRAY_OBJ] != 1 {
		t.Fatalf("want 1 array, got %d", stats.ByType[ARRAY_OBJ])
	}
	if stats.ByType[OBJECT_OBJ] != 1 {
		t.Fatalf("want 1 object, got %d", stats.ByType[OBJECT_OBJ])
	}
}

func TestObjectManagerUnregister(t *testing.T) {
	manager := NewObjectManager()
	arr := manager.NewArray(nil)

	if !manager.Unregister(arr) {
		t.Fatal("unregister should remove a known object")
	}
	if _, ok := manager.IDOf(arr); ok {
		t.Fatal("unregistered object should not have an active id")
	}

	stats := manager.Stats()
	if stats.TotalAllocated != 1 {
		t.Fatalf("want historical allocation count to remain 1, got %d", stats.TotalAllocated)
	}
	if stats.Active != 0 {
		t.Fatalf("want 0 active objects, got %d", stats.Active)
	}
}

func TestEnvironmentScopesShareObjectManager(t *testing.T) {
	env := NewEnvironment()
	child := env.NewScope()

	if env.ObjectManager() != child.ObjectManager() {
		t.Fatal("child scope should share parent object manager")
	}

	arr := child.ObjectManager().NewArray(nil)
	if _, ok := env.ObjectManager().IDOf(arr); !ok {
		t.Fatal("parent manager should see child scope allocations")
	}
}

func TestNewEnvironmentsCreateIndependentVirtualMachines(t *testing.T) {
	envA := NewEnvironment()
	envB := NewEnvironment()

	if envA.VM() == envB.VM() {
		t.Fatal("separate root environments should not share a vm")
	}
	if envA.ObjectManager() == envB.ObjectManager() {
		t.Fatal("separate root environments should not share object managers")
	}

	objA := envA.ObjectManager().NewHash()
	if _, ok := envB.ObjectManager().IDOf(objA); ok {
		t.Fatal("env b should not know objects allocated in env a")
	}
}

func TestVirtualMachinesHaveIndependentObjectManagers(t *testing.T) {
	vmA := NewVirtualMachine()
	vmB := NewVirtualMachine()

	if vmA.ObjectManager() == vmB.ObjectManager() {
		t.Fatal("different virtual machines should not share object managers")
	}

	objA := vmA.ObjectManager().NewHash()
	if _, ok := vmB.ObjectManager().IDOf(objA); ok {
		t.Fatal("vm b should not know objects allocated in vm a")
	}

	objB := vmB.ObjectManager().NewHash()
	idA, okA := vmA.ObjectManager().IDOf(objA)
	idB, okB := vmB.ObjectManager().IDOf(objB)
	if !okA || !okB {
		t.Fatal("allocated objects should have ids in their own vm")
	}
	if idA != 1 || idB != 1 {
		t.Fatalf("each vm should start object ids independently, got %d and %d", idA, idB)
	}
}

func TestVirtualMachinesHaveIndependentRuntimeState(t *testing.T) {
	vmA := NewVirtualMachine()
	vmB := NewVirtualMachine()

	if got := vmA.NextTimerID(); got != 1 {
		t.Fatalf("vm a first timer id: want 1, got %d", got)
	}
	if got := vmA.NextTimerID(); got != 2 {
		t.Fatalf("vm a second timer id: want 2, got %d", got)
	}
	if got := vmB.NextTimerID(); got != 1 {
		t.Fatalf("vm b first timer id should be independent: want 1, got %d", got)
	}
}

func TestVirtualMachinesWaitForAsyncTasksIndependently(t *testing.T) {
	vmA := NewVirtualMachine()
	vmB := NewVirtualMachine()

	releaseA := make(chan struct{})
	vmA.AsyncAdd(1)
	vmA.Go(func() {
		defer vmA.AsyncDone()
		<-releaseA
	})

	doneB := make(chan struct{})
	go func() {
		vmB.WaitAsync()
		close(doneB)
	}()

	select {
	case <-doneB:
	case <-time.After(100 * time.Millisecond):
		close(releaseA)
		t.Fatal("vm b should not wait for async tasks registered in vm a")
	}

	close(releaseA)
	vmA.WaitAsync()
}

func TestVirtualMachineHooksAreIndependent(t *testing.T) {
	vmA := NewVirtualMachine()
	vmB := NewVirtualMachine()
	envA := vmA.NewEnvironment()
	envB := vmB.NewEnvironment()

	vmA.SetImportFunc(func(env *Environment, path string) (Object, error) {
		if env.VM() != vmA {
			t.Fatal("vm a importer received an environment from another vm")
		}
		return &String{Value: "a:" + path}, nil
	})
	vmB.SetImportFunc(func(env *Environment, path string) (Object, error) {
		if env.VM() != vmB {
			t.Fatal("vm b importer received an environment from another vm")
		}
		return &String{Value: "b:" + path}, nil
	})

	importA, err := vmA.Import(envA, "mod")
	if err != nil {
		t.Fatal(err)
	}
	importB, err := vmB.Import(envB, "mod")
	if err != nil {
		t.Fatal(err)
	}
	if importA.Inspect() != "a:mod" || importB.Inspect() != "b:mod" {
		t.Fatalf("import hooks should stay isolated, got %q and %q", importA.Inspect(), importB.Inspect())
	}

	vmA.SetEvaluator(func(node interface{}, env *Environment) Object {
		if env.VM() != vmA {
			t.Fatal("vm a evaluator received an environment from another vm")
		}
		return &String{Value: "eval-a"}
	})
	vmB.SetEvaluator(func(node interface{}, env *Environment) Object {
		if env.VM() != vmB {
			t.Fatal("vm b evaluator received an environment from another vm")
		}
		return &String{Value: "eval-b"}
	})

	evalA := vmA.EvalNode(nil, envA)
	evalB := vmB.EvalNode(nil, envB)
	if evalA.Inspect() != "eval-a" || evalB.Inspect() != "eval-b" {
		t.Fatalf("evaluator hooks should stay isolated, got %q and %q", evalA.Inspect(), evalB.Inspect())
	}
}

func TestVirtualMachineSpawnersAreIndependent(t *testing.T) {
	vmA := NewVirtualMachine()
	vmB := NewVirtualMachine()

	calledA := false
	calledB := false
	vmA.SetSpawner(func(fn func()) {
		calledA = true
		fn()
	})
	vmB.SetSpawner(func(fn func()) {
		calledB = true
		fn()
	})

	ranA := false
	vmA.Go(func() { ranA = true })
	if !calledA || !ranA {
		t.Fatal("vm a should use its own spawner")
	}
	if calledB {
		t.Fatal("vm a should not call vm b spawner")
	}

	ranB := false
	vmB.Go(func() { ranB = true })
	if !calledB || !ranB {
		t.Fatal("vm b should use its own spawner")
	}
}
