package async

import (
	"sync/atomic"
	"testing"
)

func TestPool_Go(t *testing.T) {
	p := NewPool(4)
	var count int64
	for i := 0; i < 10; i++ {
		p.Go(func() {
			atomic.AddInt64(&count, 1)
		})
	}
	p.Wait()
	if count != 10 {
		t.Fatalf("want 10, got %d", count)
	}
	if p.Active() != 0 {
		t.Fatalf("want 0 active, got %d", p.Active())
	}
}

func TestPool_Size(t *testing.T) {
	p := NewPool(8)
	if p.Size() != 8 {
		t.Fatalf("want 8, got %d", p.Size())
	}
}

func TestPool_Default(t *testing.T) {
	p := NewPool(0)
	if p.Size() < 1 {
		t.Fatalf("default pool size should be >= 1, got %d", p.Size())
	}
}

func TestPool_MultipleWait(t *testing.T) {
	p := NewPool(2)
	p.Go(func() {})
	p.Wait()
	p.Go(func() {})
	p.Wait()
	if p.Active() != 0 {
		t.Fatalf("active should be 0 after Wait")
	}
}
