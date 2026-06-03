package packagefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackDirectoryAndOpen(t *testing.T) {
	root := t.TempDir()
	writePackageFile(t, filepath.Join(root, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"
`)
	writePackageFile(t, filepath.Join(root, "src", "index.gs"), `export const value = 42;`)
	out := filepath.Join(t.TempDir(), "tools.gspkg")

	if err := PackDirectory(root, out); err != nil {
		t.Fatal(err)
	}

	pkg, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Close()
	if pkg.Manifest.Package.Name != "tools" {
		t.Fatalf("want package tools, got %q", pkg.Manifest.Package.Name)
	}
	src, err := pkg.ReadText("src/index.gs")
	if err != nil {
		t.Fatal(err)
	}
	if src != `export const value = 42;` {
		t.Fatalf("unexpected source: %q", src)
	}
}

func writePackageFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
