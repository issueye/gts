package stdlib

import (
	"github.com/issueye/goscript/internal/object"
)

func hashKey(o object.Object) object.HashKey {
	switch o := o.(type) {
	case *object.String:
		return object.HashKey{Type: o.Type(), Value: o.Value}
	default:
		return object.HashKey{Type: o.Type(), Value: o.Inspect()}
	}
}

func setHashMember(hash *object.Hash, key string, value object.Object) {
	hash.Pairs[hashKey(&object.String{Value: key})] = object.HashPair{
		Key: &object.String{Value: key}, Value: value,
	}
}

func hashValue(hash *object.Hash, key string) (object.Object, bool) {
	hk := hashKey(&object.String{Value: key})
	v, ok := hash.Pairs[hk]
	return v.Value, ok
}
