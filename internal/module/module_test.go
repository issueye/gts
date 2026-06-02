package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/issueye/goscript/internal/object"
)

func TestResolvePathAddsDefaultExtension(t *testing.T) {
	got := ResolvePath("./lib", "/project/src")
	want := filepath.Join("/project/src", "lib.gs")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestResolveAgentAliasFromProjectRoot(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "examples", "nested")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte("[project]\nentry = \"main.gs\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := ResolvePath("@agent/core/agent", nested)
	want := filepath.Join(dir, "scripts", "agent", "core", "agent.gs")
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestCacheUsesConfiguredVirtualMachine(t *testing.T) {
	vm := object.NewVirtualMachine()
	cache := NewCacheWithVM(vm)

	env := cache.GetOrCreate(filepath.Join(t.TempDir(), "lib.gs"))
	if env.VM() != vm {
		t.Fatal("module cache should create environments inside the configured vm")
	}
	if env.ObjectManager() != vm.ObjectManager() {
		t.Fatal("module cache should create environments inside the configured vm")
	}

	otherCache := NewCache()
	otherEnv := otherCache.GetOrCreate(filepath.Join(t.TempDir(), "lib.gs"))
	if otherEnv.VM() == vm {
		t.Fatal("separate module cache should not share the vm by default")
	}
	if otherEnv.ObjectManager() == vm.ObjectManager() {
		t.Fatal("separate module cache should not share the vm manager by default")
	}
}
