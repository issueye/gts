package proj

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	cfg := Load(filepath.Join(t.TempDir(), "project.toml"))

	if cfg.Name != "project" {
		t.Fatalf("Name = %q, want project", cfg.Name)
	}
	if cfg.Version != "" {
		t.Fatalf("Version = %q, want empty", cfg.Version)
	}
	if cfg.Entry != "main.gs" {
		t.Fatalf("Entry = %q, want main.gs", cfg.Entry)
	}
}

func TestLoadProjectFields(t *testing.T) {
	cfg := loadToml(t, `
[project]
name = "demo"
version = "1.2.3"
entry = "src/main.gs"
`)

	if cfg.Name != "demo" {
		t.Fatalf("Name = %q, want demo", cfg.Name)
	}
	if cfg.Version != "1.2.3" {
		t.Fatalf("Version = %q, want 1.2.3", cfg.Version)
	}
	if cfg.Entry != "src/main.gs" {
		t.Fatalf("Entry = %q, want src/main.gs", cfg.Entry)
	}
}

func TestLoadPackageManifestFields(t *testing.T) {
	cfg := loadToml(t, `
[project]
name = "app"
version = "0.1.0"
entry = "main.gs"

[package]
name = "@scope/pkg"
version = "2.0.0"
type = "module"
main = "dist/index.gs"

[exports]
"." = "./dist/index.gs"
"./cli" = "./dist/cli.gs"

[imports]
"#util" = "./src/util.gs"

[dependencies]
"std/http" = ">=1.0.0"
"left-pad" = "1.3.0"

[bundle]
target = "browser"
format = "esm"
includeStd = true
external = ["std/http", "left-pad"]
`)

	if cfg.Package != (PackageConfig{Name: "@scope/pkg", Version: "2.0.0", Type: "module", Main: "dist/index.gs"}) {
		t.Fatalf("Package = %#v", cfg.Package)
	}

	wantExports := map[string]string{
		".":     "./dist/index.gs",
		"./cli": "./dist/cli.gs",
	}
	if !reflect.DeepEqual(cfg.Exports, wantExports) {
		t.Fatalf("Exports = %#v, want %#v", cfg.Exports, wantExports)
	}

	wantImports := map[string]string{
		"#util": "./src/util.gs",
	}
	if !reflect.DeepEqual(cfg.Imports, wantImports) {
		t.Fatalf("Imports = %#v, want %#v", cfg.Imports, wantImports)
	}

	wantDependencies := map[string]string{
		"std/http": ">=1.0.0",
		"left-pad": "1.3.0",
	}
	if !reflect.DeepEqual(cfg.Dependencies, wantDependencies) {
		t.Fatalf("Dependencies = %#v, want %#v", cfg.Dependencies, wantDependencies)
	}

	if cfg.Bundle.Target != "browser" {
		t.Fatalf("Bundle.Target = %q, want browser", cfg.Bundle.Target)
	}
	if cfg.Bundle.Format != "esm" {
		t.Fatalf("Bundle.Format = %q, want esm", cfg.Bundle.Format)
	}
	if !cfg.Bundle.IncludeStd {
		t.Fatal("Bundle.IncludeStd = false, want true")
	}
	if !reflect.DeepEqual(cfg.Bundle.External, []string{"std/http", "left-pad"}) {
		t.Fatalf("Bundle.External = %#v", cfg.Bundle.External)
	}
}

func TestLoadPluginConfig(t *testing.T) {
	cfg := loadToml(t, `
[plugins.scheduler]
command = "go"
args = ["run", "./cmd/gtp-scheduler"]
cwd = "."
modules = ["@plugin/scheduler"]
capabilities = ["call", "event"]

[plugins.disabled]
command = "noop"
autoStart = false
`)

	scheduler, ok := cfg.Plugins["scheduler"]
	if !ok {
		t.Fatal("scheduler plugin was not loaded")
	}
	if scheduler.Command != "go" {
		t.Fatalf("Command = %q, want go", scheduler.Command)
	}
	if !scheduler.AutoStart {
		t.Fatal("AutoStart = false, want true by default")
	}
	if !reflect.DeepEqual(scheduler.Args, []string{"run", "./cmd/gtp-scheduler"}) {
		t.Fatalf("Args = %#v", scheduler.Args)
	}
	if !reflect.DeepEqual(scheduler.Modules, []string{"@plugin/scheduler"}) {
		t.Fatalf("Modules = %#v", scheduler.Modules)
	}

	disabled := cfg.Plugins["disabled"]
	if disabled.AutoStart {
		t.Fatal("disabled AutoStart = true, want false")
	}
}

func loadToml(t *testing.T, content string) *Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "project.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}
