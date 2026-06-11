package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/packagefile"
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

func TestResolverCachesSuccessfulResolution(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.gs")
	writeFile(t, lib, "")

	resolver := NewResolver("")
	first, err := resolver.Resolve("./lib", ResolveOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(lib); err != nil {
		t.Fatal(err)
	}
	second, err := resolver.Resolve("./lib", ResolveOptions{BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != second.Path || first.ID != second.ID {
		t.Fatalf("cached resolution changed: first=%#v second=%#v", first, second)
	}
}

func TestResolverCachesFailedResolution(t *testing.T) {
	dir := t.TempDir()
	resolver := NewResolver("")

	if _, err := resolver.Resolve("./missing", ResolveOptions{BaseDir: dir}); err == nil {
		t.Fatal("expected initial missing module error")
	}
	writeFile(t, filepath.Join(dir, "missing.gs"), "")
	if _, err := resolver.Resolve("./missing", ResolveOptions{BaseDir: dir}); err == nil {
		t.Fatal("expected cached missing module error")
	}
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

func TestResolverPackageImportAlias(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "project.toml"), `[package]
name = "app"
main = "src/main.gs"

[imports]
"#util/*" = "src/internal/*.gs"
`)
	writeFile(t, filepath.Join(root, "src", "internal", "format.gs"), "")

	resolved, err := NewResolver("").Resolve("#util/format", ResolveOptions{BaseDir: filepath.Join(root, "src")})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageName != "app" {
		t.Fatalf("want package name app, got %q", resolved.PackageName)
	}
	assertPath(t, resolved.Path, filepath.Join(root, "src", "internal", "format.gs"))
}

func TestResolverPackageImportAliasAbsoluteStyle(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "project.toml"), `[package]
name = "app"
main = "src/main.gs"

[imports]
"@/*" = "src/*.gs"
`)
	writeFile(t, filepath.Join(root, "src", "lib", "format.gs"), "")

	resolved, err := NewResolver("").Resolve("@/lib/format", ResolveOptions{BaseDir: filepath.Join(root, "src")})
	if err != nil {
		t.Fatal(err)
	}
	assertPath(t, resolved.Path, filepath.Join(root, "src", "lib", "format.gs"))
}

func TestResolverPackageDependencyUsesNearestPackageRoot(t *testing.T) {
	root := t.TempDir()
	tools := filepath.Join(root, "vendor", "tools")
	helper := filepath.Join(tools, "vendor", "helper")

	writeFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"

[dependencies]
"tools" = "file:vendor/tools"
`)
	writeFile(t, filepath.Join(tools, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[dependencies]
"helper" = "file:vendor/helper"
`)
	writeFile(t, filepath.Join(helper, "project.toml"), `[package]
name = "helper"
version = "2.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`)
	writeFile(t, filepath.Join(helper, "src", "index.gs"), "")

	resolved, err := NewResolver(root).Resolve("helper", ResolveOptions{
		ProjectRoot: root,
		BaseDir:     filepath.Join(tools, "src"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != "pkg:helper@2.0.0:." {
		t.Fatalf("unexpected package id: %q", resolved.ID)
	}
	assertPath(t, resolved.Path, filepath.Join(helper, "src", "index.gs"))
}

func TestResolverPackageFileExport(t *testing.T) {
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "tools-src")
	pkgPath := filepath.Join(root, "vendor", "tools.gspkg")
	writeFile(t, filepath.Join(root, "project.toml"), `[project]
name = "app"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	writeFile(t, filepath.Join(pkgRoot, "project.toml"), `[package]
name = "tools"
version = "1.2.3"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
"./format" = "src/format.gs"
`)
	writeFile(t, filepath.Join(pkgRoot, "src", "index.gs"), "")
	writeFile(t, filepath.Join(pkgRoot, "src", "format.gs"), "")
	if err := packagefile.PackDirectory(pkgRoot, pkgPath); err != nil {
		t.Fatal(err)
	}

	resolved, err := NewResolver(root).Resolve("tools/format", ResolveOptions{
		ProjectRoot: root,
		BaseDir:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageFile == "" || resolved.ArchivePath != "src/format.gs" {
		t.Fatalf("unexpected package file resolution: %#v", resolved)
	}
	if resolved.ID != "pkg:tools@1.2.3:./format:src/format.gs" {
		t.Fatalf("unexpected package id: %q", resolved.ID)
	}
}

func TestResolverPackageFileNestedDependency(t *testing.T) {
	root := t.TempDir()
	helperRoot := filepath.Join(root, "helper-src")
	writeFile(t, filepath.Join(helperRoot, "project.toml"), `[package]
name = "helper"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`)
	writeFile(t, filepath.Join(helperRoot, "src", "index.gs"), "")
	helperPkg := filepath.Join(root, "helper.gspkg")
	if err := packagefile.PackDirectory(helperRoot, helperPkg); err != nil {
		t.Fatal(err)
	}

	toolsRoot := filepath.Join(root, "tools-src")
	writeFile(t, filepath.Join(toolsRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[dependencies]
"helper" = "file:vendor/helper.gspkg"
`)
	writeFile(t, filepath.Join(toolsRoot, "src", "index.gs"), "")
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
	toolsPkg := filepath.Join(root, "tools.gspkg")
	if err := packagefile.PackDirectory(toolsRoot, toolsPkg); err != nil {
		t.Fatal(err)
	}

	resolved, err := NewResolver(root).Resolve("helper", ResolveOptions{
		ProjectRoot: root,
		BaseDir:     filepath.ToSlash(toolsPkg) + "!src",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageFile == "" || !strings.Contains(resolved.PackageFile, "tools.gspkg!vendor/helper.gspkg") {
		t.Fatalf("unexpected nested package file resolution: %#v", resolved)
	}
	if resolved.ArchivePath != "src/index.gs" {
		t.Fatalf("want nested archive path src/index.gs, got %q", resolved.ArchivePath)
	}
}

func TestResolverPackageFileImportAliasAbsoluteStyle(t *testing.T) {
	root := t.TempDir()
	pkgRoot := filepath.Join(root, "tools-src")
	pkgPath := filepath.Join(root, "tools.gspkg")
	writeFile(t, filepath.Join(pkgRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[imports]
"@/*" = "src/*.gs"
`)
	writeFile(t, filepath.Join(pkgRoot, "src", "internal", "format.gs"), "")
	if err := packagefile.PackDirectory(pkgRoot, pkgPath); err != nil {
		t.Fatal(err)
	}

	resolved, err := NewResolver(root).Resolve("@/internal/format", ResolveOptions{
		ProjectRoot: root,
		BaseDir:     filepath.ToSlash(pkgPath) + "!src",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PackageFile == "" || resolved.ArchivePath != "src/internal/format.gs" {
		t.Fatalf("unexpected package file alias resolution: %#v", resolved)
	}
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
