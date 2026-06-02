package module

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/issueye/goscript/internal/object"
)

// Cache stores loaded modules to avoid re-execution.
type Cache struct {
	mu      sync.Mutex
	modules map[string]*object.Environment // absolute path → module env
}

func NewCache() *Cache {
	return &Cache{modules: make(map[string]*object.Environment)}
}

// GetOrCreate returns a cached module env, or nil if not yet loaded.
func (c *Cache) GetOrCreate(absPath string) *object.Environment {
	c.mu.Lock()
	defer c.mu.Unlock()
	if mod, ok := c.modules[absPath]; ok {
		return mod
	}
	env := object.NewEnvironment()
	c.modules[absPath] = env
	return env
}

// Get returns a cached module env or nil.
func (c *Cache) Get(absPath string) *object.Environment {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.modules[absPath]
}

// ResolvePath resolves a module path relative to the base directory.
func ResolvePath(path, baseDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if baseDir == "" {
		baseDir, _ = os.Getwd()
	}
	resolved := filepath.Join(baseDir, path)
	if filepath.Ext(resolved) == "" {
		resolved += ".gs"
	}
	return resolved
}

// SetupExports initializes module.exports on an environment.
func SetupExports(env *object.Environment) {
	exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	env.Set("exports", exports)
	mod := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	mod.Pairs[hashKey(&object.String{Value: "exports"})] = object.HashPair{
		Key: &object.String{Value: "exports"}, Value: exports,
	}
	env.Set("module", mod)
}

// GetExports returns the exports object from a module env.
func GetExports(env *object.Environment) object.Object {
	v, _ := env.Get("exports")
	return v
}

func hashKey(o object.Object) object.HashKey {
	switch o := o.(type) {
	case *object.String:
		return object.HashKey{Type: o.Type(), Value: o.Value}
	default:
		return object.HashKey{Type: o.Type(), Value: o.Inspect()}
	}
}
