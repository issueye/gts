package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/packagefile"
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

func TestBundlePackageFileDependency(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.gs")
	pkgRoot := filepath.Join(root, "tools-src")
	pkgPath := filepath.Join(root, "vendor", "tools.gspkg")

	writeBundleFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"
entry = "main.gs"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	writeBundleFile(t, entry, `let feature = require("tools/feature");
exports.value = feature.value;
`)
	writeBundleFile(t, filepath.Join(pkgRoot, "project.toml"), `[package]
name = "tools"
version = "1.2.3"

[exports]
"." = "src/index.gs"
"./feature" = "src/feature.gs"
`)
	writeBundleFile(t, filepath.Join(pkgRoot, "src", "index.gs"), `exports.value = 1;
`)
	writeBundleFile(t, filepath.Join(pkgRoot, "src", "feature.gs"), `let fmt = require("./format");
exports.value = fmt.wrap("pkgfile");
`)
	writeBundleFile(t, filepath.Join(pkgRoot, "src", "format.gs"), `exports.wrap = function(x) { return "[" + x + "]"; };
`)
	if err := packagefile.PackDirectory(pkgRoot, pkgPath); err != nil {
		t.Fatal(err)
	}

	out, err := Bundle(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `require("tools/feature")`) || strings.Contains(out, `require("./format")`) {
		t.Fatalf("package file requires were not bundled:\n%s", out)
	}
	if !strings.Contains(out, "tools.gspkg!src/feature.gs") || !strings.Contains(out, "tools.gspkg!src/format.gs") {
		t.Fatalf("expected package file modules in bundle:\n%s", out)
	}
	if !strings.Contains(out, "pkgfile") {
		t.Fatalf("expected package file source in bundle:\n%s", out)
	}
}

func TestBundlePackageFileNestedDependency(t *testing.T) {
	root := t.TempDir()
	entry := filepath.Join(root, "main.gs")
	helperRoot := filepath.Join(root, "helper-src")
	toolsRoot := filepath.Join(root, "tools-src")

	writeBundleFile(t, filepath.Join(helperRoot, "project.toml"), `[package]
name = "helper"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`)
	writeBundleFile(t, filepath.Join(helperRoot, "src", "index.gs"), `exports.label = "nested-bundle";
`)
	helperPkg := filepath.Join(root, "helper.gspkg")
	if err := packagefile.PackDirectory(helperRoot, helperPkg); err != nil {
		t.Fatal(err)
	}

	writeBundleFile(t, filepath.Join(toolsRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[dependencies]
"helper" = "file:vendor/helper.gspkg"
`)
	writeBundleFile(t, filepath.Join(toolsRoot, "src", "index.gs"), `let helper = require("helper");
exports.value = "tools:" + helper.label;
`)
	helperBytes, err := os.ReadFile(helperPkg)
	if err != nil {
		t.Fatal(err)
	}
	helperInTools := filepath.Join(toolsRoot, "vendor", "helper.gspkg")
	if err := os.MkdirAll(filepath.Dir(helperInTools), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helperInTools, helperBytes, 0644); err != nil {
		t.Fatal(err)
	}
	toolsPkg := filepath.Join(root, "vendor", "tools.gspkg")
	if err := packagefile.PackDirectory(toolsRoot, toolsPkg); err != nil {
		t.Fatal(err)
	}

	writeBundleFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"
entry = "main.gs"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	writeBundleFile(t, entry, `let tools = require("tools");
exports.value = tools.value;
`)

	out, err := Bundle(entry)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `require("tools")`) || strings.Contains(out, `require("helper")`) {
		t.Fatalf("nested package requires were not bundled:\n%s", out)
	}
	if !strings.Contains(out, "helper.gspkg!src/index.gs") || !strings.Contains(out, "nested-bundle") {
		t.Fatalf("expected nested package module in bundle:\n%s", out)
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
