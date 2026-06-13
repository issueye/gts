package eventloop

import (
	"context"
	"errors"
	"sync"

	"github.com/issueye/goscript/internal/async"
	"github.com/issueye/goscript/internal/object"
)

var ErrStopped = errors.New("event loop worker stopped")

type Task struct {
	Name string
	Run  func(*object.VirtualMachine)
}

type AsyncResult struct {
	Value any
}

type AsyncOp struct {
	Name   string
	Run    func(context.Context) (AsyncResult, error)
	Resume func(*object.VirtualMachine, AsyncResult, error)
}

type Worker struct {
	vm      *object.VirtualMachine
	tasks   chan Task
	done    chan struct{}
	once    sync.Once
	ioPool  *async.Pool
	context context.Context
	cancel  context.CancelFunc
}

func NewWorker(vm *object.VirtualMachine, ioWorkers int) *Worker {
	if vm == nil {
		vm = object.NewVirtualMachine()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		vm:      vm,
		tasks:   make(chan Task, 1024),
		done:    make(chan struct{}),
		ioPool:  async.NewPool(ioWorkers),
		context: ctx,
		cancel:  cancel,
	}
}

func (w *Worker) VM() *object.VirtualMachine {
	if w == nil {
		return nil
	}
	return w.vm
}

func (w *Worker) Post(task Task) error {
	if w == nil {
		return ErrStopped
	}
	select {
	case <-w.done:
		return ErrStopped
	case w.tasks <- task:
		return nil
	}
}

func (w *Worker) PostFunc(name string, run func(*object.VirtualMachine)) error {
	return w.Post(Task{Name: name, Run: run})
}

func (w *Worker) Run() {
	if w == nil {
		return
	}
	for {
		select {
		case <-w.done:
			return
		case task := <-w.tasks:
			if task.Run != nil {
				task.Run(w.vm)
			}
		}
	}
}

func (w *Worker) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.cancel()
		close(w.done)
		if w.ioPool != nil {
			w.ioPool.Wait()
		}
	})
}

func (w *Worker) RunSync(task Task) error {
	done := make(chan struct{})
	err := w.Post(Task{
		Name: task.Name,
		Run: func(vm *object.VirtualMachine) {
			defer close(done)
			if task.Run != nil {
				task.Run(vm)
			}
		},
	})
	if err != nil {
		return err
	}
	select {
	case <-done:
		return nil
	case <-w.done:
		return ErrStopped
	}
}

func (w *Worker) StartAsync(op AsyncOp) error {
	if w == nil || w.ioPool == nil {
		return ErrStopped
	}
	select {
	case <-w.done:
		return ErrStopped
	default:
	}
	w.ioPool.Go(func() {
		var result AsyncResult
		var err error
		if op.Run != nil {
			result, err = op.Run(w.context)
		}
		_ = w.Post(Task{
			Name: op.Name + ".resume",
			Run: func(vm *object.VirtualMachine) {
				if op.Resume != nil {
					op.Resume(vm, result, err)
				}
			},
		})
	})
	return nil
}
