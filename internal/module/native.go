package module

import (
	"sort"
	"sync"

	"github.com/issueye/goscript/internal/object"
)

// NativeModuleFactory creates a new module instance.
type NativeModuleFactory func(env *object.Environment) (object.Object, error)

var (
	nativeMu      sync.RWMutex
	nativeModules = make(map[string]NativeModuleFactory)
	nativeAPIDocs = make(map[string][]string)
)

// RegisterNative registers a native Go module under a given import path.
// Import paths use the @std/ prefix convention (e.g. "@std/exec", "@std/net/http/client").
func RegisterNative(path string, factory NativeModuleFactory) {
	nativeMu.Lock()
	nativeModules[path] = factory
	nativeMu.Unlock()
}

// RegisterNativeAPIDoc registers printable API signatures for a native module.
func RegisterNativeAPIDoc(path string, signatures []string) {
	nativeMu.Lock()
	nativeAPIDocs[path] = append([]string{}, signatures...)
	nativeMu.Unlock()
}

// GetNative returns a native module by its import path.
// Returns nil, false if no native module is registered for the path.
func GetNative(path string, env *object.Environment) (object.Object, bool) {
	nativeMu.RLock()
	factory, ok := nativeModules[path]
	nativeMu.RUnlock()
	if !ok {
		return nil, false
	}
	obj, err := factory(env)
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

// ListNative returns all registered native module paths in sorted order.
func ListNative() []string {
	nativeMu.RLock()
	paths := make([]string, 0, len(nativeModules))
	for path := range nativeModules {
		paths = append(paths, path)
	}
	nativeMu.RUnlock()
	sort.Strings(paths)
	return paths
}

// GetNativeAPIDoc returns printable API signatures for a native module.
func GetNativeAPIDoc(path string) ([]string, bool) {
	nativeMu.RLock()
	signatures, ok := nativeAPIDocs[path]
	nativeMu.RUnlock()
	if !ok {
		return nil, false
	}
	out := append([]string{}, signatures...)
	sort.Strings(out)
	return out, true
}
