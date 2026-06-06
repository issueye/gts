package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/issueye/goscript/internal/gtp"
	"github.com/issueye/goscript/internal/gtp/pluginhost"
	"github.com/issueye/goscript/internal/gtp/scheduler"
	"github.com/issueye/goscript/internal/proj"
)

func main() {
	root, err := filepath.Abs(".")
	if err != nil {
		fatal(err)
	}

	host := pluginhost.New(root)
	defer host.Close()

	err = host.Start("scheduler", proj.PluginConfig{
		Command:      "go",
		Args:         []string{"run", "./cmd/gtp-scheduler"},
		Cwd:          ".",
		AutoStart:    true,
		Modules:      []string{scheduler.ModuleName},
		Capabilities: []string{"call", "event"},
	})
	if err != nil {
		fatal(err)
	}

	plugin, ok := host.Plugin("scheduler")
	if !ok {
		fatal(fmt.Errorf("scheduler plugin was not started"))
	}

	task, err := plugin.Call(scheduler.ModuleName, "schedule", []gtp.Value{
		gtp.Object(map[string]gtp.Value{
			"name":    gtp.String("host-handled-demo"),
			"delayMs": gtp.Number(150),
			"payload": gtp.Object(map[string]gtp.Value{
				"kind":    gtp.String("print"),
				"message": gtp.String("run this payload in the host"),
				"source":  gtp.String("scheduler"),
			}),
		}),
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("scheduled task: %v\n", gtp.Plain(task))

	select {
	case event, ok := <-plugin.Events():
		if !ok {
			fatal(fmt.Errorf("scheduler plugin event stream closed"))
		}
		if err := handleSchedulerEvent(event); err != nil {
			fatal(err)
		}
	case <-time.After(3 * time.Second):
		fatal(fmt.Errorf("timeout waiting for scheduler trigger event"))
	}
}

func handleSchedulerEvent(frame gtp.Frame) error {
	if frame.Type != "event" {
		return fmt.Errorf("expected event frame, got %q", frame.Type)
	}
	if frame.Module != scheduler.ModuleName || frame.Event != "trigger" {
		return fmt.Errorf("unexpected event %s/%s", frame.Module, frame.Event)
	}
	if frame.Data == nil {
		return fmt.Errorf("scheduler trigger event is missing data")
	}

	data := *frame.Data
	taskID, _ := gtp.StringField(data, "id")
	name, _ := gtp.StringField(data, "name")
	fired, _ := gtp.NumberField(data, "fired")
	payload, ok := gtp.Field(data, "payload")
	if !ok {
		payload = gtp.Null()
	}

	fmt.Printf("received scheduler event: task=%s name=%s fired=%.0f payload=%v\n", taskID, name, fired, gtp.Plain(payload))
	return executePayload(payload)
}

func executePayload(payload gtp.Value) error {
	kind, _ := gtp.StringField(payload, "kind")
	switch kind {
	case "print":
		message, _ := gtp.StringField(payload, "message")
		fmt.Printf("host executor: %s\n", message)
		return nil
	case "":
		return fmt.Errorf("payload.kind is required")
	default:
		return fmt.Errorf("unsupported payload.kind %q", kind)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
