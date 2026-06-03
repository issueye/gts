package object

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/issueye/goscript/internal/ast"
)

// ObjectID is the runtime identity assigned by ObjectManager.
type ObjectID uint64

// ObjectRecord describes one object currently known to the runtime manager.
type ObjectRecord struct {
	ID        ObjectID
	Type      ObjectType
	Object    Object
	CreatedAt time.Time
	Tags      map[string]string
}

// ObjectStats is a compact view of managed runtime objects.
type ObjectStats struct {
	TotalAllocated uint64
	Active         int
	ByType         map[ObjectType]int
}

// ObjectManager centralizes runtime object allocation and tracking.
//
// It is intentionally lightweight today: it gives the interpreter one place to
// create and observe runtime objects. The same boundary can later grow VM-level
// concerns such as handles, roots, tracing, or custom memory policies.
type ObjectManager struct {
	mu             sync.RWMutex
	tracking       atomic.Bool
	nextID         ObjectID
	totalAllocated uint64
	records        map[ObjectID]ObjectRecord
	ids            map[Object]ObjectID
}

// NewObjectManager creates an empty runtime object manager.
func NewObjectManager() *ObjectManager {
	return newObjectManager(true)
}

func newObjectManager(track bool) *ObjectManager {
	manager := &ObjectManager{
		records: make(map[ObjectID]ObjectRecord),
		ids:     make(map[Object]ObjectID),
	}
	manager.tracking.Store(track)
	return manager
}

func (m *ObjectManager) SetTracking(enabled bool) {
	if m == nil {
		return
	}
	m.tracking.Store(enabled)
}

func (m *ObjectManager) Tracking() bool {
	return m != nil && m.tracking.Load()
}

// Register records an object that was created outside the manager factory
// helpers. Registering the same object twice returns the original id.
func (m *ObjectManager) Register(obj Object) ObjectID {
	if obj == nil || !m.Tracking() {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.ids[obj]; ok {
		return id
	}
	m.nextID++
	id := m.nextID
	m.totalAllocated++
	m.ids[obj] = id
	m.records[id] = ObjectRecord{
		ID:        id,
		Type:      obj.Type(),
		Object:    obj,
		CreatedAt: time.Now(),
	}
	return id
}

// Unregister removes an object from active tracking.
func (m *ObjectManager) Unregister(obj Object) bool {
	if obj == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.ids[obj]
	if !ok {
		return false
	}
	delete(m.ids, obj)
	delete(m.records, id)
	return true
}

// IDOf returns the runtime id assigned to obj.
func (m *ObjectManager) IDOf(obj Object) (ObjectID, bool) {
	if obj == nil {
		return 0, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.ids[obj]
	return id, ok
}

// Get returns a record by id.
func (m *ObjectManager) Get(id ObjectID) (ObjectRecord, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.records[id]
	return record, ok
}

// Snapshot returns a stable copy of active object records.
func (m *ObjectManager) Snapshot() []ObjectRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ObjectRecord, 0, len(m.records))
	for _, record := range m.records {
		out = append(out, record)
	}
	return out
}

// Stats summarizes active and historical allocations.
func (m *ObjectManager) Stats() ObjectStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := ObjectStats{
		TotalAllocated: m.totalAllocated,
		Active:         len(m.records),
		ByType:         make(map[ObjectType]int),
	}
	for _, record := range m.records {
		stats.ByType[record.Type]++
	}
	return stats
}

// Allocate registers obj and returns it for inline construction.
func (m *ObjectManager) Allocate(obj Object) Object {
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewArray(elements []Object) *Array {
	obj := &Array{Elements: elements}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewArrayAt(elements []Object, pos ast.Position) *Array {
	obj := &Array{Elements: elements, Pos: pos}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewHash() *Hash {
	obj := &Hash{Pairs: make(map[HashKey]HashPair)}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewHashAt(pos ast.Position) *Hash {
	obj := &Hash{Pairs: make(map[HashKey]HashPair), Pos: pos}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewMap() *Map {
	obj := &Map{Entries: make(map[HashKey]HashPair)}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewMapAt(pos ast.Position) *Map {
	obj := &Map{Entries: make(map[HashKey]HashPair), Pos: pos}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewSet() *Set {
	obj := &Set{Values: make(map[HashKey]Object)}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewSetAt(pos ast.Position) *Set {
	obj := &Set{Values: make(map[HashKey]Object), Pos: pos}
	m.Register(obj)
	return obj
}

func (m *ObjectManager) NewPromise() *Promise {
	obj := NewPromise()
	m.Register(obj)
	return obj
}
