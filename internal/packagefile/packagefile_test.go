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

func TestAppendPackageToExecutableAndOpen(t *testing.T) {
	root := t.TempDir()
	writePackageFile(t, filepath.Join(root, "project.toml"), `[package]
name = "tools"
main = "src/index.gs"
`)
	writePackageFile(t, filepath.Join(root, "src", "index.gs"), `exports.value = 42;`)
	pkgPath := filepath.Join(t.TempDir(), "tools.gspkg")
	if err := PackDirectory(root, pkgPath); err != nil {
		t.Fatal(err)
	}

	stub := filepath.Join(t.TempDir(), "stub.exe")
	if err := os.WriteFile(stub, []byte("stub-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "app.exe")
	if err := AppendPackageToExecutable(stub, pkgPath, out); err != nil {
		t.Fatal(err)
	}

	data, err := ReadAppendedPackage(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected appended package bytes")
	}
	opened, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	text, err := opened.ReadText("src/index.gs")
	if err != nil {
		t.Fatal(err)
	}
	if text != `exports.value = 42;` {
		t.Fatalf("unexpected source: %q", text)
	}
}

func TestOpenAppendedExecutableSubpackage(t *testing.T) {
	root := t.TempDir()
	writePackageFile(t, filepath.Join(root, "project.toml"), `[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools"
`)
	writePackageFile(t, filepath.Join(root, "app.gs"), `exports.value = "app";`)
	writePackageFile(t, filepath.Join(root, "vendor", "tools", "project.toml"), `[package]
name = "tools"
main = "src/index.gs"
`)
	writePackageFile(t, filepath.Join(root, "vendor", "tools", "src", "index.gs"), `exports.value = "tools";`)
	pkgPath := filepath.Join(t.TempDir(), "app.gspkg")
	if err := PackDirectory(root, pkgPath); err != nil {
		t.Fatal(err)
	}

	stub := filepath.Join(t.TempDir(), "stub.exe")
	if err := os.WriteFile(stub, []byte("stub-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "app.exe")
	if err := AppendPackageToExecutable(stub, pkgPath, out); err != nil {
		t.Fatal(err)
	}
	subpkg, err := Open(out + "!vendor/tools")
	if err != nil {
		t.Fatal(err)
	}
	defer subpkg.Close()
	src, err := subpkg.ReadText("src/index.gs")
	if err != nil {
		t.Fatal(err)
	}
	if src != `exports.value = "tools";` {
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
