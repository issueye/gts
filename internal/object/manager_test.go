package object

import "testing"

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
