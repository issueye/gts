package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	cfg := Load(filepath.Join(t.TempDir(), "config.toml"))

	if cfg.Plugins != nil {
		t.Fatalf("Plugins = %#v, want nil", cfg.Plugins)
	}
}

func TestLoadPluginConfig(t *testing.T) {
	cfg := loadToml(t, `
[plugins.scheduler]
command = "go"
args = ["run", "."]
cwd = "../../plugins/scheduler"
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
	if !reflect.DeepEqual(scheduler.Args, []string{"run", "."}) {
		t.Fatalf("Args = %#v", scheduler.Args)
	}
	if scheduler.Cwd != "../../plugins/scheduler" {
		t.Fatalf("Cwd = %q", scheduler.Cwd)
	}
	if !reflect.DeepEqual(scheduler.Modules, []string{"@plugin/scheduler"}) {
		t.Fatalf("Modules = %#v", scheduler.Modules)
	}
	if !reflect.DeepEqual(scheduler.Capabilities, []string{"call", "event"}) {
		t.Fatalf("Capabilities = %#v", scheduler.Capabilities)
	}

	disabled := cfg.Plugins["disabled"]
	if disabled.AutoStart {
		t.Fatal("disabled AutoStart = true, want false")
	}
}

func loadToml(t *testing.T, content string) *Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}
