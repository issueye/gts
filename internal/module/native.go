package module

import (
	"sync"

	"github.com/issueye/goscript/internal/object"
)

// NativeModuleFactory creates a new module instance.
type NativeModuleFactory func() (object.Object, error)

var (
	nativeMu      sync.RWMutex
	nativeModules = make(map[string]NativeModuleFactory)
)

// RegisterNative registers a native Go module under a given import path.
// Import paths use the @std/ prefix convention (e.g. "@std/exec", "@std/net/http/client").
func RegisterNative(path string, factory NativeModuleFactory) {
	nativeMu.Lock()
	nativeModules[path] = factory
	nativeMu.Unlock()
}

// GetNative returns a native module by its import path.
// Returns nil, false if no native module is registered for the path.
func GetNative(path string) (object.Object, bool) {
	nativeMu.RLock()
	factory, ok := nativeModules[path]
	nativeMu.RUnlock()
	if !ok {
		return nil, false
	}
	obj, err := factory()
	if err != nil {
		return nil, false
	}
	return obj, true
}

// HasNative checks whether a native module is registered for the given path.
func HasNative(path string) bool {
	nativeMu.RLock()
	_, ok := nativeModules[path]
	nativeMu.RUnlock()
	return ok
}
