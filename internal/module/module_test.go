package module

import (
	"os"
	"path/filepath"
	"testing"
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
