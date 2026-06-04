package safemap

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSafeSortedMapSortedItems(t *testing.T) {
	m := New[string, int](func(a, b string) bool { return a < b })
	m.Set("b", 2)
	m.Set("a", 1)
	m.Set("c", 3)

	items := m.SortedItems()
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	got := items[0].Key + items[1].Key + items[2].Key
	if got != "abc" {
		t.Fatalf("want sorted keys abc, got %q", got)
	}
}

func TestSafeSortedMapConcurrentAccess(t *testing.T) {
	m := New[string, int](func(a, b string) bool { return a < b })
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := strconv.Itoa(i % 10)
			m.Set(key, i)
			m.Get(key)
			m.SortedKeys()
		}()
	}
	wg.Wait()
	if m.Len() == 0 {
		t.Fatal("expected concurrent writes to populate map")
	}
}

func TestSafeSortedMapGetOrSetFuncCreatesOnce(t *testing.T) {
	m := New[string, int](func(a, b string) bool { return a < b })
	var calls atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, _ := m.GetOrSetFunc("shared", func() int {
				calls.Add(1)
				return 42
			})
			if value != 42 {
				t.Errorf("want 42, got %d", value)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("factory should be called once, got %d", calls.Load())
	}
}

func BenchmarkNativeMapGet(b *testing.B) {
	m := map[string]int{"value": 1}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := m["value"]; !ok {
			b.Fatal("value not found")
		}
	}
}

func BenchmarkSafeSortedMapGet(b *testing.B) {
	m := New[string, int](func(a, b string) bool { return a < b })
	m.Set("value", 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := m.Get("value"); !ok {
			b.Fatal("value not found")
		}
	}
}

func BenchmarkSafeSortedMapSetExisting(b *testing.B) {
	m := New[string, int](func(a, b string) bool { return a < b })
	m.Set("value", 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Set("value", i)
	}
}

func BenchmarkSafeSortedMapGetOrSetExisting(b *testing.B) {
	m := New[string, int](func(a, b string) bool { return a < b })
	m.Set("value", 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if value, existed := m.GetOrSetFunc("value", func() int { return 2 }); !existed || value != 1 {
			b.Fatalf("unexpected result value=%d existed=%v", value, existed)
		}
	}
}

func BenchmarkSafeSortedMapSortedKeys100(b *testing.B) {
	m := New[string, int](func(a, b string) bool { return a < b })
	for i := 99; i >= 0; i-- {
		m.Set(strconv.Itoa(i), i)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		keys := m.SortedKeys()
		if len(keys) != 100 {
			b.Fatalf("want 100 keys, got %d", len(keys))
		}
	}
}
