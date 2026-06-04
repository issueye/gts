package safemap

import (
	"sort"
	"sync"
)

// Less orders keys for SortedKeys and SortedItems.
type Less[K comparable] func(a, b K) bool

// Item is a key/value pair snapshot from SafeSortedMap.
type Item[K comparable, V any] struct {
	Key   K
	Value V
}

// SafeSortedMap is a concurrency-safe map with deterministic sorted snapshots.
//
// The zero value is ready to use. Sorting requires a Less function; without one,
// SortedKeys and SortedItems return the same keys as insertion-neutral snapshots.
type SafeSortedMap[K comparable, V any] struct {
	mu     sync.RWMutex
	values map[K]V
	less   Less[K]
}

func New[K comparable, V any](less Less[K]) *SafeSortedMap[K, V] {
	return &SafeSortedMap[K, V]{less: less}
}

func (m *SafeSortedMap[K, V]) SetLess(less Less[K]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.less = less
}

func (m *SafeSortedMap[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = make(map[K]V)
	}
	m.values[key] = value
}

func (m *SafeSortedMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.values[key]
	return value, ok
}

func (m *SafeSortedMap[K, V]) GetOrSetFunc(key K, makeValue func() V) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = make(map[K]V)
	}
	if value, ok := m.values[key]; ok {
		return value, true
	}
	value := makeValue()
	m.values[key] = value
	return value, false
}

func (m *SafeSortedMap[K, V]) Has(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.values[key]
	return ok
}

func (m *SafeSortedMap[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
}

func (m *SafeSortedMap[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.values)
}

func (m *SafeSortedMap[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.values))
	for key := range m.values {
		keys = append(keys, key)
	}
	return keys
}

func (m *SafeSortedMap[K, V]) SortedKeys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.values))
	for key := range m.values {
		keys = append(keys, key)
	}
	if m.less != nil {
		sort.Slice(keys, func(i, j int) bool {
			return m.less(keys[i], keys[j])
		})
	}
	return keys
}

func (m *SafeSortedMap[K, V]) Snapshot() map[K]V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[K]V, len(m.values))
	for key, value := range m.values {
		out[key] = value
	}
	return out
}

func (m *SafeSortedMap[K, V]) Items() []Item[K, V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]Item[K, V], 0, len(m.values))
	for key, value := range m.values {
		items = append(items, Item[K, V]{Key: key, Value: value})
	}
	return items
}

func (m *SafeSortedMap[K, V]) SortedItems() []Item[K, V] {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]Item[K, V], 0, len(m.values))
	for key, value := range m.values {
		items = append(items, Item[K, V]{Key: key, Value: value})
	}
	if m.less != nil {
		sort.Slice(items, func(i, j int) bool {
			return m.less(items[i].Key, items[j].Key)
		})
	}
	return items
}
