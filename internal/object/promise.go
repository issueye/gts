package object

import "sync"

// PromiseState represents the state of a Promise.
type PromiseState int

const (
	PROMISE_PENDING   PromiseState = 0
	PROMISE_FULFILLED PromiseState = 1
	PROMISE_REJECTED  PromiseState = 2
)

// Promise represents an async value.
type Promise struct {
	mu      sync.Mutex
	state   PromiseState
	value   Object
	reason  Object
	resolve chan struct{} // closed when resolved
}

func NewPromise() *Promise {
	return &Promise{
		state:   PROMISE_PENDING,
		resolve: make(chan struct{}),
	}
}

func (p *Promise) Type() ObjectType { return "PROMISE" }
func (p *Promise) Inspect() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == PROMISE_PENDING {
		return "<promise pending>"
	}
	if p.state == PROMISE_FULFILLED {
		return "<promise resolved: " + p.value.Inspect() + ">"
	}
	return "<promise rejected: " + p.reason.Inspect() + ">"
}

func (p *Promise) Resolve(val Object) {
	p.mu.Lock()
	if p.state != PROMISE_PENDING {
		p.mu.Unlock()
		return
	}
	p.state = PROMISE_FULFILLED
	p.value = val
	p.mu.Unlock()
	close(p.resolve)
}

func (p *Promise) Reject(reason Object) {
	p.mu.Lock()
	if p.state != PROMISE_PENDING {
		p.mu.Unlock()
		return
	}
	p.state = PROMISE_REJECTED
	p.reason = reason
	p.mu.Unlock()
	close(p.resolve)
}

// Wait blocks until the promise is settled, then returns the value/reason.
func (p *Promise) Wait() Object {
	<-p.resolve
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == PROMISE_FULFILLED {
		return p.value
	}
	return p.reason
}

func (p *Promise) Then(onFulfilled *Function) *Promise {
	next := NewPromise()
	Spawn(func() {
		result := p.Wait()
		if p.State() == PROMISE_FULFILLED && onFulfilled != nil {
			scope := onFulfilled.Env.NewScope()
			scope.Set(onFulfilled.Parameters[0].Name, result)
			EvalPromiseFn(onFulfilled.Body, scope, next)
		} else {
			next.Resolve(result)
		}
	})
	return next
}

// Spawn is set by the evaluator to use the goroutine pool.
var Spawn func(fn func()) = func(fn func()) { go fn() }

var EvalPromiseFn func(body interface{}, env *Environment, promise *Promise)

// EvalFn is set by the evaluator to allow stdlib modules to evaluate AST nodes.
var EvalFn func(node interface{}, env *Environment) Object

func (p *Promise) State() PromiseState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}
