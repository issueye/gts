package runtime

import (
	"errors"
	"sync"
)

// InstanceFactory produces a ready-to-serve Session. For @std/web isolated
// mode this means a Session that has already loaded the app bootstrap (routes
// registered, modules cached), so a checked-out instance can run a request
// handler without re-parsing the app.
//
// The factory exists because the bootstrap source and options are known to the
// caller (the web layer), not to this package — the pool is generic over how
// instances are created.
type InstanceFactory func() (*Session, error)

// InstancePool is a bounded pool of warm Sessions. Unlike
// object.VirtualMachinePool (which Reset()s VMs on return, scrubbing all
// script state), InstancePool preserves each Session's loaded state so the
// bootstrap does not re-run per request. This is the performance-critical
// difference between cold and warm isolated mode.
//
// Get blocks (acquiring a semaphore slot) until an instance is available or
// the pool is closed; it then returns either an idle instance or a freshly
// factory-built one. Put returns an instance to the idle set. Discard removes
// an instance from rotation entirely — used when a Session's state is
// suspected poisoned (e.g. a handler mutated global state) and must not be
// reused.
//
// Concurrency: Get/Put/Close/Discard are safe for concurrent use.
type InstancePool struct {
	factory InstanceFactory
	slots   chan struct{} // bounded capacity; acquired in Get, released in Put/Discard
	idle    chan *Session
	once    sync.Once
	closed  chan struct{}
}

// NewInstancePool creates a pool with the given bounded size. The factory is
// called lazily on demand (not pre-warmed) unless Warm is called.
func NewInstancePool(size int, factory InstanceFactory) *InstancePool {
	if size < 1 {
		size = 1
	}
	return &InstancePool{
		factory: factory,
		slots:   make(chan struct{}, size),
		idle:    make(chan *Session, size),
		closed:  make(chan struct{}),
	}
}

// Warm pre-builds up to n instances and parks them in the idle set, each
// counting against the pool's capacity. It blocks until all n are built or an
// error occurs. n must not exceed the pool size.
//
// Note: Warm acquires a slot per instance (matching Get's contract), so the
// slot-release path is consistent whether an instance arrives via Warm or Get.
func (p *InstancePool) Warm(n int) error {
	if p == nil {
		return errors.New("instance pool is nil")
	}
	for i := 0; i < n; i++ {
		// Acquire a capacity slot so Warm respects the pool bound.
		select {
		case p.slots <- struct{}{}:
		case <-p.closed:
			return ErrPoolClosed
		}
		sess, err := p.factory()
		if err != nil {
			<-p.slots
			return err
		}
		// Park in idle. offerIdle never closes the session on the full branch
		// here because we hold exactly one slot per warmed instance and the
		// caller is expected to pass n <= size; but guard defensively.
		select {
		case p.idle <- sess:
		default:
			sess.Close()
			<-p.slots
		}
	}
	return nil
}

// ErrPoolClosed is returned by Get after Close.
var ErrPoolClosed = errors.New("instance pool is closed")

// Get returns a Session for exclusive use. It blocks until one is available.
// Callers must call either Put (reuse) or Discard (do not reuse) exactly once.
func (p *InstancePool) Get() (*Session, error) {
	if p == nil {
		return nil, errors.New("instance pool is nil")
	}
	// Acquire a capacity slot first so the total outstanding count never
	// exceeds size, even when factory builds are in flight.
	select {
	case p.slots <- struct{}{}:
	case <-p.closed:
		return nil, ErrPoolClosed
	}
	for {
		select {
		case sess := <-p.idle:
			return sess, nil
		case <-p.closed:
			p.releaseSlot()
			return nil, ErrPoolClosed
		default:
		}
		// No idle instance; build a fresh one.
		sess, err := p.factory()
		if err != nil {
			p.releaseSlot()
			return nil, err
		}
		return sess, nil
	}
}

// Put returns an instance to the idle set for reuse. If the pool is closed or
// the idle set is full, the instance is closed and its slot released.
func (p *InstancePool) Put(sess *Session) {
	if p == nil || sess == nil {
		return
	}
	select {
	case p.idle <- sess:
		// Parked; slot stays held until a future Get pulls it back out.
		return
	default:
	}
	// Idle set full: close the session and release its capacity slot so a
	// blocked Get can build/accept another.
	sess.Close()
	p.releaseSlot()
}

// Discard removes an instance from rotation and closes it. Use this when the
// instance's state may be corrupted and must not be reused. Its capacity slot
// is released so a blocked Get can proceed.
func (p *InstancePool) Discard(sess *Session) {
	if p == nil || sess == nil {
		return
	}
	sess.Close()
	p.releaseSlot()
}

// Close closes the pool and all idle instances. Outstanding (checked-out)
// instances are the caller's responsibility; Put/Discard on them after Close
// simply closes them and releases their slots. Idempotent.
func (p *InstancePool) Close() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		close(p.closed)
		for {
			select {
			case sess := <-p.idle:
				sess.Close()
				p.releaseSlot()
			default:
				return
			}
		}
	})
}

// releaseSlot returns one capacity slot to the pool, tolerating Close having
// already happened (in which case the slot channel is simply not refilled).
func (p *InstancePool) releaseSlot() {
	select {
	case <-p.slots:
	case <-p.closed:
	}
}
