package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleRelativeDependency(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.gs")
	writeBundleFile(t, entry, `let lib = require("./lib");
exports.value = lib.value;
`)
	writeBundleFile(t, filepath.Join(dir, "lib.gs"), `exports.value = 42;
`)

	out, err := Bundle(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `require("./lib")`) {
		t.Fatalf("relative require was not bundled:\n%s", out)
	}
	if !strings.Contains(out, "lib = __mod_") || !strings.Contains(out, "_lib_exports") {
		t.Fatalf("expected bundled dependency export reference:\n%s", out)
	}
}

func TestBundlePackageExportDependency(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.gs")
	dep := filepath.Join(root, "vendor", "tools")

	writeBundleFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"
entry = "main.gs"

[dependencies]
"tools" = "file:vendor/tools"
`)
	writeBundleFile(t, entry, `let feature = require("tools/feature");
exports.value = feature.value;
`)
	writeBundleFile(t, filepath.Join(dep, "project.toml"), `[package]
name = "tools"
version = "1.2.3"

[exports]
"." = "src/index.gs"
"./feature" = "src/feature.gs"
`)
	writeBundleFile(t, filepath.Join(dep, "src", "index.gs"), `exports.value = 1;
`)
	writeBundleFile(t, filepath.Join(dep, "src", "feature.gs"), `exports.value = 2;
`)

	out, err := Bundle(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `require("tools/feature")`) {
		t.Fatalf("package export require was not bundled:\n%s", out)
	}
	if !strings.Contains(out, "feature = __mod_") || !strings.Contains(out, "_feature_exports") {
		t.Fatalf("expected package export reference:\n%s", out)
	}
}

func writeBundleFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
