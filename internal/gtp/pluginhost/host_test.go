package pluginhost

import (
	"path/filepath"
	"testing"

	"github.com/issueye/goscript/internal/gtp"
	"github.com/issueye/goscript/internal/object"
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
		Args:         []string{"run", "."},
		Cwd:          "plugins/scheduler",
		AutoStart:    true,
		Modules:      []string{"@plugin/scheduler"},
		Capabilities: []string{"call", "event"},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPluginEventListenersAreScopedByModule(t *testing.T) {
	plugin := &Plugin{eventListeners: make(map[string][]*pluginEventListener)}
	defer plugin.closeEventListeners()

	env := object.NewEnvironment()
	env.VM().SetEvaluator(func(node interface{}, env *object.Environment) object.Object {
		return object.UNDEFINED
	})
	fn := &object.Function{Env: env}
	plugin.addEventListener("@plugin/a", "trigger", fn, true)
	plugin.addEventListener("@plugin/b", "trigger", fn, true)

	plugin.dispatchEvent(gtp.Frame{
		Version: gtp.Version,
		ID:      "evt-a",
		Type:    "event",
		Module:  "@plugin/a",
		Event:   "trigger",
	})

	if got := plugin.listenerCount("@plugin/a", "trigger"); got != 0 {
		t.Fatalf("module a listener count = %d, want 0", got)
	}
	if got := plugin.listenerCount("@plugin/b", "trigger"); got != 1 {
		t.Fatalf("module b listener count = %d, want 1", got)
	}
}
