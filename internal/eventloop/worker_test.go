package eventloop

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/issueye/goscript/internal/object"
)

func TestWorkerRunsTasksFIFOOnOneOwnerGoroutine(t *testing.T) {
	worker := NewWorker(object.NewVirtualMachine(), 2)
	defer worker.Stop()
	go worker.Run()

	var mu sync.Mutex
	order := []int{}
	owners := map[uint64]bool{}
	for i := 0; i < 8; i++ {
		i := i
		if err := worker.RunSync(Task{Name: "task", Run: func(vm *object.VirtualMachine) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, i)
			owners[currentGoroutineID()] = true
		}}); err != nil {
			t.Fatal(err)
		}
	}

	for i, got := range order {
		if got != i {
			t.Fatalf("task order[%d] = %d, want %d; full order=%v", i, got, i, order)
		}
	}
	if len(owners) != 1 {
		t.Fatalf("tasks should run on one owner goroutine, got %d goroutines", len(owners))
	}
}

func TestWorkerAsyncOpResumesOnOwnerGoroutine(t *testing.T) {
	worker := NewWorker(object.NewVirtualMachine(), 2)
	defer worker.Stop()
	go worker.Run()

	ownerReady := make(chan uint64, 1)
	if err := worker.RunSync(Task{Name: "owner", Run: func(vm *object.VirtualMachine) {
		ownerReady <- currentGoroutineID()
	}}); err != nil {
		t.Fatal(err)
	}
	ownerID := <-ownerReady

	backendReady := make(chan uint64, 1)
	resumeReady := make(chan struct{})
	if err := worker.StartAsync(AsyncOp{
		Name: "io",
		Run: func(ctx context.Context) (AsyncResult, error) {
			backendReady <- currentGoroutineID()
			return AsyncResult{Value: "ok"}, nil
		},
		Resume: func(vm *object.VirtualMachine, result AsyncResult, err error) {
			defer close(resumeReady)
			if err != nil {
				t.Errorf("unexpected async error: %v", err)
			}
			if result.Value != "ok" {
				t.Errorf("result = %v, want ok", result.Value)
			}
			if got := currentGoroutineID(); got != ownerID {
				t.Errorf("resume goroutine = %d, want owner %d", got, ownerID)
			}
		},
	}); err != nil {
		t.Fatal(err)
	}
	backendID := <-backendReady
	if backendID == ownerID {
		t.Fatalf("async backend should not run on owner goroutine")
	}
	<-resumeReady
}

func currentGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 {
		return 0
	}
	id, _ := strconv.ParseUint(fields[1], 10, 64)
	return id
}
