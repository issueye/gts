package pluginhost

import (
	"path/filepath"
	"testing"

	"github.com/issueye/goscript/internal/proj"
)

func TestStartConfiguredSkipsDisabled(t *testing.T) {
	host := New(t.TempDir())
	defer host.Close()

	err := host.StartConfigured(map[string]proj.PluginConfig{
		"disabled": {
			Command:   "definitely-not-a-real-command",
			AutoStart: false,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartSchedulerPlugin(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	host := New(root)
	defer host.Close()

	err = host.Start("scheduler", proj.PluginConfig{
		Command:      "go",
		Args:         []string{"run", "./cmd/gtp-scheduler"},
		Cwd:          ".",
		AutoStart:    true,
		Modules:      []string{"@plugin/scheduler"},
		Capabilities: []string{"call", "event"},
	})
	if err != nil {
		t.Fatal(err)
	}
}
