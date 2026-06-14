package runtime

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/issueye/goscript/internal/object"
)

// TestTwoSessionsIsolateModuleTopLevelState is the P1 acceptance test called
// out in docs/plans/vm-environment-isolation-development-plan.md: two sessions
// loading the same module file must observe independent top-level state.
// If this regresses, per-request VM isolation in @std/web is broken.
func TestTwoSessionsIsolateModuleTopLevelState(t *testing.T) {
	dir := t.TempDir()
	// counter.gs exports inc()/get(). Each session that requires it gets its
	// own *object.Environment and its own `value` binding.
	counterSrc := `let value = 0;
export function inc() { value = value + 1; return value; }
export function get() { return value; }
`
	if err := os.WriteFile(filepath.Join(dir, "counter.gs"), []byte(counterSrc), 0644); err != nil {
		t.Fatal(err)
	}
	appSrc := `let counter = require("./counter.gs");
counter.inc();
counter.inc();
// Expose the value via a top-level thrown value is awkward; instead we stash it
// on a global-ish field by returning from main. The test reads it via callMain.
function main() { return counter.get(); }
`
	if err := os.WriteFile(filepath.Join(dir, "app.gs"), []byte(appSrc), 0644); err != nil {
		t.Fatal(err)
	}

	run := func() int {
		sess := NewSession(Options{WorkingDir: dir, RootDir: dir, Timeout: 0})
		defer sess.Close()
		result, err := sess.LoadEntry(appSrc, filepath.Join(dir, "app.gs"), true)
		if err != nil {
			t.Fatalf("LoadEntry: %v", err)
		}
		got, ok := asInt(result)
		if !ok {
			t.Fatalf("unexpected main result: %v", result)
		}
		return got
	}

	// First session incs twice -> 2. Second session starts fresh -> also 2.
	// If state leaked across sessions, the second would report 4.
	if got := run(); got != 2 {
		t.Fatalf("first session get = %d, want 2", got)
	}
	if got := run(); got != 2 {
		t.Fatalf("second session get = %d, want 2 (state leaked between sessions)", got)
	}
}

// TestInstancePoolReusesSessionsAndBoundsConcurrency verifies the warm pool's
// core invariants: Get/Put cycle reuses instances, and a pool at capacity
// blocks further Get calls (rather than over-allocating).
func TestInstancePoolReusesSessionsAndBoundsConcurrency(t *testing.T) {
	var created int
	var mu sync.Mutex
	factory := func() (*Session, error) {
		mu.Lock()
		created++
		mu.Unlock()
		return NewSession(Options{}), nil
	}
	pool := NewInstancePool(2, factory)

	s1, err := pool.Get()
	if err != nil {
		t.Fatal(err)
	}
	s2, err := pool.Get()
	if err != nil {
		t.Fatal(err)
	}
	_ = s1
	_ = s2 // held only to saturate the pool; released implicitly by Close

	// Pool is now saturated (size 2). A third Get must block. We prove this by
	// closing the pool: a blocked Get unblocks with ErrPoolClosed. If Get had
	// already returned a session (over-allocation), closeTimeout would catch
	// nothing and the goroutine would have handed us a session — which we
	// detect below.
	gotSess := make(chan *Session, 1)
	go func() {
		s, err := pool.Get()
		if err == nil {
			gotSess <- s
		}
	}()

	select {
	case leaked := <-gotSess:
		t.Fatalf("third Get returned %v despite full pool; want it to block", leaked)
	case <-afterDuration(50 * ms):
		// Still blocked, as expected.
	}

	// Verify reuse on the return path before closing. The pending third Get is
	// still blocked, so Put(s1) must NOT be consumed by it (it is waiting on a
	// slot, and s1 returning frees exactly one slot — actually it WOULD be
	// consumed). To keep this assertion sound, unblock the waiter first.
	pool.Close() // unblocks the third Get with ErrPoolClosed; it sends nothing.

	// Re-create a fresh pool for the reuse check to avoid the waiter's shadow.
	pool2 := NewInstancePool(2, factory)
	a, err := pool2.Get()
	if err != nil {
		t.Fatal(err)
	}
	pool2.Put(a)
	b, err := pool2.Get()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("Get did not reuse returned session")
	}
	pool2.Put(b)
	pool2.Close()

	mu.Lock()
	totalCreated := created
	mu.Unlock()
	// pool built 2; pool2 built 1. Total = 3. The point of the check is that
	// reuse happened within each pool (pool's s1/s2 were not rebuilt, pool2's
	// single instance was returned and reused, not rebuilt).
	if totalCreated != 3 {
		t.Fatalf("factory invoked %d times, want 3 (2 for pool + 1 for pool2)", totalCreated)
	}
}

// TestInstancePoolDiscardDropsInstance verifies that a discarded session is not
// reused, forcing the next Get to build a fresh one.
func TestInstancePoolDiscardDropsInstance(t *testing.T) {
	var created int
	var mu sync.Mutex
	factory := func() (*Session, error) {
		mu.Lock()
		created++
		mu.Unlock()
		return NewSession(Options{}), nil
	}
	pool := NewInstancePool(1, factory)
	defer pool.Close()

	s1, err := pool.Get()
	if err != nil {
		t.Fatal(err)
	}
	pool.Discard(s1) // poisoned; do not reuse

	s2, err := pool.Get()
	if err != nil {
		t.Fatal(err)
	}
	if s2 == s1 {
		t.Fatalf("Get returned discarded session")
	}
	pool.Put(s2)

	mu.Lock()
	defer mu.Unlock()
	if created != 2 {
		t.Fatalf("factory invoked %d times, want 2 after discard", created)
	}
}

// afterDuration returns a channel that fires after d, mirroring time.After but
// kept here to keep test helpers local.
func afterDuration(d msDuration) <-chan time.Time { return time.After(time.Duration(d)) }

type msDuration time.Duration

const ms = msDuration(time.Millisecond)

func asInt(obj interface{}) (int, bool) {
	n, ok := obj.(*object.Number)
	if !ok {
		return 0, false
	}
	return int(n.Value), true
}
