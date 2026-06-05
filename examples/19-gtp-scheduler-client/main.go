package main

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/issueye/goscript/internal/gtp"
	"github.com/issueye/goscript/internal/gtp/scheduler"
)

func main() {
	cmd := exec.Command("go", "run", "./cmd/gtp-scheduler")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	enc := gtp.NewEncoder(stdin)
	dec := gtp.NewDecoder(stdout)

	if err := enc.Encode(gtp.Frame{
		Version:      gtp.Version,
		ID:           "hello-1",
		Type:         "hello",
		Runtime:      "example",
		Protocol:     "gtp",
		Capabilities: []string{"call", "event"},
		Modules:      []string{scheduler.ModuleName},
	}); err != nil {
		panic(err)
	}
	ready, err := dec.Decode()
	if err != nil {
		panic(err)
	}
	fmt.Printf("ready: service=%s modules=%v\n", ready.Service, ready.Modules)

	if err := enc.Encode(gtp.Frame{
		Version: gtp.Version,
		ID:      "schedule-1",
		Type:    "call",
		Module:  scheduler.ModuleName,
		Method:  "schedule",
		Args: []gtp.Value{
			gtp.Object(map[string]gtp.Value{
				"name":    gtp.String("demo"),
				"delayMs": gtp.Number(100),
				"payload": gtp.Object(map[string]gtp.Value{
					"message": gtp.String("hello from timer"),
				}),
			}),
		},
	}); err != nil {
		panic(err)
	}
	scheduled, err := dec.Decode()
	if err != nil {
		panic(err)
	}
	if scheduled.Result != nil {
		fmt.Printf("scheduled: %v\n", gtp.Plain(*scheduled.Result))
	}

	start := time.Now()
	event, err := dec.Decode()
	if err != nil {
		panic(err)
	}
	if event.Data != nil {
		fmt.Printf("event after %s: %s %v\n", time.Since(start).Round(time.Millisecond), event.Event, gtp.Plain(*event.Data))
	}
}
