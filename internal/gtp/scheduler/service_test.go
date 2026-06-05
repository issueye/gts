package scheduler

import (
	"testing"
	"time"

	"github.com/issueye/goscript/internal/gtp"
)

func TestScheduleTriggersEvent(t *testing.T) {
	svc := NewService()
	defer svc.Clear()

	result := svc.Handle(gtp.Frame{
		Version: gtp.Version,
		ID:      "1",
		Type:    "call",
		Module:  ModuleName,
		Method:  "schedule",
		Args: []gtp.Value{
			gtp.Object(map[string]gtp.Value{
				"name":    gtp.String("once"),
				"delayMs": gtp.Number(10),
				"payload": gtp.Object(map[string]gtp.Value{
					"kind": gtp.String("test"),
				}),
			}),
		},
	})
	if result.OK == nil || !*result.OK || result.Result == nil {
		t.Fatalf("result = %#v", result)
	}

	select {
	case event := <-svc.Events():
		if event.Type != "event" || event.Event != "trigger" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestCancelStopsTask(t *testing.T) {
	svc := NewService()
	defer svc.Clear()

	result := svc.Handle(gtp.Frame{
		Version: gtp.Version,
		ID:      "1",
		Type:    "call",
		Module:  ModuleName,
		Method:  "schedule",
		Args: []gtp.Value{
			gtp.Object(map[string]gtp.Value{"delayMs": gtp.Number(200)}),
		},
	})
	taskID, _ := gtp.StringField(*result.Result, "id")
	cancel := svc.Handle(gtp.Frame{
		Version: gtp.Version,
		ID:      "2",
		Type:    "call",
		Module:  ModuleName,
		Method:  "cancel",
		Args:    []gtp.Value{gtp.String(taskID)},
	})
	cancelled, _ := gtp.Field(*cancel.Result, "cancelled")
	if b, _ := cancelled.BoolValue(); !b {
		t.Fatal("task was not cancelled")
	}

	select {
	case event := <-svc.Events():
		t.Fatalf("unexpected event %#v", event)
	case <-time.After(250 * time.Millisecond):
	}
}

func TestListAndClear(t *testing.T) {
	svc := NewService()
	defer svc.Clear()

	svc.Handle(gtp.Frame{ID: "1", Type: "call", Module: ModuleName, Method: "schedule", Args: []gtp.Value{gtp.Object(map[string]gtp.Value{"delayMs": gtp.Number(1000)})}})
	list := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "list"})
	if list.Result == nil || len(list.Result.Items) != 1 {
		t.Fatalf("list = %#v", list)
	}
	clear := svc.Handle(gtp.Frame{ID: "3", Type: "call", Module: ModuleName, Method: "clear"})
	count, _ := gtp.Field(*clear.Result, "count")
	if n, _ := count.NumberValue(); n != 1 {
		t.Fatalf("clear count = %v", n)
	}
}
