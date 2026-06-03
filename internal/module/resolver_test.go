package module

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolverNative(t *testing.T) {
	resolved, err := NewResolver("").Resolve("@std/fs", ResolveOptions{BaseDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != ModuleKindNative || resolved.ID != "native:@std/fs" || !resolved.External {
		t.Fatalf("unexpected native resolution: %#v", resolved)
	}
}

func TestResolverRelativeDefaultExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "lib.gs"), "")

	resolved, err := NewResolver("").Resolve("./lib", ResolveOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	assertPath(t, resolved.Path, filepath.Join(dir, "lib.gs"))
	if resolved.Kind != ModuleKindSource {
		t.Fatalf("want source kind, got %s", resolved.Kind)
	}
}

func TestResolverAgentAlias(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "examples", "nested")
	agent := filepath.Join(root, "scripts", "agent", "core", "agent.gs")
	writeFile(t, filepath.Join(root, "project.toml"), "[project]\nentry = \"main.gs\"\n")
	writeFile(t, agent, "")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	resolved, err := NewResolver("").Resolve("@agent/core/agent", ResolveOptions{BaseDir: nested})
	if err != nil {
		t.Fatal(err)
	}
	assertPath(t, resolved.Path, agent)
}

func TestResolverDirectoryMainAndIndex(t *testing.T) {
	dir := t.TempDir()
	withMain := filepath.Join(dir, "with-main")
	withIndex := filepath.Join(dir, "with-index")
	writeFile(t, filepath.Join(withMain, "project.toml"), "[package]\nmain = \"src/start.gs\"\n")
	writeFile(t, filepath.Join(withMain, "src", "start.gs"), "")
	writeFile(t, filepath.Join(withIndex, "index.gs"), "")

	resolvedMain, err := NewResolver("").Resolve("./with-main", ResolveOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	assertPath(t, resolvedMain.Path, filepath.Join(withMain, "src", "start.gs"))

	resolvedIndex, err := NewResolver("").Resolve("./with-index", ResolveOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	assertPath(t, resolvedIndex.Path, filepath.Join(withIndex, "index.gs"))
}

func TestResolverPackageExport(t *testing.T) {
	root := t.TempDir()
	dep := filepath.Join(root, "vendor", "tools")
	writeFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"

[dependencies]
"tools" = "file:vendor/tools"
`)
	writeFile(t, filepath.Join(dep, "project.toml"), `[package]
name = "tools"
version = "1.2.3"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
"./feature" = "src/feature.gs"
`)
	writeFile(t, filepath.Join(dep, "src", "index.gs"), "")
	writeFile(t, filepath.Join(dep, "src", "feature.gs"), "")

	resolved, err := NewResolver("").Resolve("tools/feature", ResolveOptions{BaseDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != ModuleKindPackage {
		t.Fatalf("want package kind, got %s", resolved.Kind)
	}
	if resolved.PackageName != "tools" {
		t.Fatalf("want package name tools, got %q", resolved.PackageName)
	}
	if resolved.ID != "pkg:tools@1.2.3:./feature" {
		t.Fatalf("unexpected package id: %q", resolved.ID)
	}
	assertPath(t, resolved.Path, filepath.Join(dep, "src", "feature.gs"))
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertPath(t *testing.T, got, want string) {
	t.Helper()
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("want %q, got %q", want, got)
	}
}
