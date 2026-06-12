package stdlib

import (
	"github.com/issueye/goscript/internal/object"
)

func hashKey(o object.Object) object.HashKey {
	return object.HashKeyFor(o)
}

func setHashMember(hash *object.Hash, key string, value object.Object) {
	hash.SetMember(&object.String{Value: key}, value)
}

func hashValue(hash *object.Hash, key string) (object.Object, bool) {
	hk := hashKey(&object.String{Value: key})
	v, ok := hash.Pairs[hk]
	return v.Value, ok
}

func getHashValue(hash *object.Hash, key string) object.Object {
	val, _ := hashValue(hash, key)
	return val
}
