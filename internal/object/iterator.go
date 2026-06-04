package object

import "fmt"

// IterationKind selects whether an iterator yields keys or values.
type IterationKind int

const (
	IterateKeys IterationKind = iota
	IterateValues
)

// Iterator is the common runtime contract for for-in and for-of loops.
type Iterator interface {
	Next() (Object, bool)
}

type sliceIterator struct {
	values []Object
	index  int
}

func (it *sliceIterator) Next() (Object, bool) {
	if it.index >= len(it.values) {
		return nil, false
	}
	value := it.values[it.index]
	it.index++
	return value, true
}

// NewIterator returns the iterator used by loop evaluation.
func NewIterator(obj Object, kind IterationKind) (Iterator, bool) {
	switch o := obj.(type) {
	case *Array:
		return newArrayIterator(o, kind), true
	case *Hash:
		return newHashIterator(o, kind), true
	case *String:
		return newStringIterator(o, kind), true
	case *Map:
		return newMapIterator(o, kind), true
	case *Set:
		return newSetIterator(o), true
	default:
		return nil, false
	}
}

func newArrayIterator(arr *Array, kind IterationKind) Iterator {
	values := make([]Object, len(arr.Elements))
	for i, elem := range arr.Elements {
		if kind == IterateKeys {
			values[i] = &String{Value: fmt.Sprintf("%d", i)}
		} else {
			values[i] = elem
		}
	}
	return &sliceIterator{values: values}
}

func newHashIterator(hash *Hash, kind IterationKind) Iterator {
	values := make([]Object, 0, len(hash.Pairs))
	for _, pair := range hash.OrderedPairs() {
		if kind == IterateKeys {
			values = append(values, pair.Key)
		} else {
			values = append(values, pair.Value)
		}
	}
	return &sliceIterator{values: values}
}

func newStringIterator(str *String, kind IterationKind) Iterator {
	if kind == IterateKeys {
		values := make([]Object, 0, len(str.Value))
		for i := 0; i < len(str.Value); i++ {
			values = append(values, &Number{Value: float64(i)})
		}
		return &sliceIterator{values: values}
	}

	values := make([]Object, 0, len(str.Value))
	for _, ch := range str.Value {
		values = append(values, &String{Value: string(ch)})
	}
	return &sliceIterator{values: values}
}

func newMapIterator(m *Map, kind IterationKind) Iterator {
	values := make([]Object, 0, len(m.Entries))
	for _, pair := range m.Entries {
		if kind == IterateKeys {
			values = append(values, pair.Key)
		} else {
			values = append(values, pair.Value)
		}
	}
	return &sliceIterator{values: values}
}

func newSetIterator(s *Set) Iterator {
	values := make([]Object, 0, len(s.Values))
	for _, value := range s.Values {
		values = append(values, value)
	}
	return &sliceIterator{values: values}
}
