package async

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Pool manages a bounded set of goroutines to prevent leaks.
type Pool struct {
	sem     chan struct{}
	wg      sync.WaitGroup
	active  int64
	running int32 // non-zero if pool is active
}

// NewPool creates a goroutine pool with max concurrent workers.
// If maxWorkers <= 0, uses runtime.NumCPU().
func NewPool(maxWorkers int) *Pool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	return &Pool{sem: make(chan struct{}, maxWorkers)}
}

// Go runs fn in the pool. If the pool is at capacity, the call blocks
// until a worker slot is available. The WaitGroup is incremented before
// the goroutine starts and decremented when fn returns.
func (p *Pool) Go(fn func()) {
	p.sem <- struct{}{} // acquire slot
	p.wg.Add(1)
	atomic.AddInt64(&p.active, 1)
	atomic.StoreInt32(&p.running, 1)
	go func() {
		defer func() {
			p.wg.Done()
			atomic.AddInt64(&p.active, -1)
			<-p.sem // release slot
		}()
		defer RecoverPanic("worker pool task")
		fn()
	}()
}

// TryGo runs fn if a worker slot is available immediately.
// Returns false if the pool is full.
func (p *Pool) TryGo(fn func()) bool {
	select {
	case p.sem <- struct{}{}:
		p.wg.Add(1)
		atomic.AddInt64(&p.active, 1)
		go func() {
			defer func() {
				p.wg.Done()
				atomic.AddInt64(&p.active, -1)
				<-p.sem
			}()
			defer RecoverPanic("worker pool task")
			fn()
		}()
		return true
	default:
		return false
	}
}

// Wait blocks until all submitted tasks have completed.
func (p *Pool) Wait() {
	p.wg.Wait()
	atomic.StoreInt32(&p.running, 0)
}

// Active returns the number of currently executing tasks.
func (p *Pool) Active() int64 {
	return atomic.LoadInt64(&p.active)
}

// Size returns the maximum number of concurrent workers.
func (p *Pool) Size() int {
	return cap(p.sem)
}

// QueueLen returns the number of waiting tasks.
func (p *Pool) QueueLen() int {
	return len(p.sem)
}
