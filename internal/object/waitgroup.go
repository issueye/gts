package object

import (
	"sync"
)

// WaitGroup is a GoScript wait group for synchronizing goroutines
type WaitGroup struct {
	wg sync.WaitGroup
}

func (w *WaitGroup) Type() ObjectType { return WAITGROUP_OBJ }
func (w *WaitGroup) Inspect() string  { return "<waitgroup>" }

func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

func (w *WaitGroup) Add(delta int) {
	w.wg.Add(delta)
}

func (w *WaitGroup) Done() {
	w.wg.Done()
}

func (w *WaitGroup) Wait() {
	w.wg.Wait()
}
