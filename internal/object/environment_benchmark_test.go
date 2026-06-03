package object

import "testing"

func BenchmarkEnvironmentGetLocal(b *testing.B) {
	env := NewEnvironment()
	env.Set("value", &Number{Value: 1})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := env.Get("value"); !ok {
			b.Fatal("value not found")
		}
	}
}

func BenchmarkEnvironmentGetParentDepth16(b *testing.B) {
	env := NewEnvironment()
	env.Set("value", &Number{Value: 1})
	for i := 0; i < 16; i++ {
		env = env.NewScope()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := env.Get("value"); !ok {
			b.Fatal("value not found")
		}
	}
}

func BenchmarkEnvironmentGetGlobalConstDepth16(b *testing.B) {
	vm := NewVirtualMachine()
	env := vm.NewEnvironment()
	vm.SetGlobalConst("value", &Number{Value: 1})
	for i := 0; i < 16; i++ {
		env = env.NewScope()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := env.Get("value"); !ok {
			b.Fatal("value not found")
		}
	}
}

func BenchmarkEnvironmentAssignParentDepth16(b *testing.B) {
	env := NewEnvironment()
	env.Set("value", &Number{Value: 1})
	for i := 0; i < 16; i++ {
		env = env.NewScope()
	}
	next := &Number{Value: 2}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok, isConst := env.Assign("value", next); !ok || isConst {
			b.Fatalf("assign failed: ok=%v const=%v", ok, isConst)
		}
	}
}
