package module

import (
	"os"
	"path/filepath"

	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/safemap"
)

// Cache stores loaded modules to avoid re-execution.
type Cache struct {
	vm      *object.VirtualMachine
	modules safemap.SafeSortedMap[string, *object.Environment] // absolute path → module env
}

func NewCache() *Cache {
	return NewCacheWithVM(nil)
}

func NewCacheWithVM(vm *object.VirtualMachine) *Cache {
	if vm == nil {
		vm = object.NewVirtualMachine()
	}
	cache := &Cache{vm: vm}
	cache.modules.SetLess(func(a, b string) bool { return a < b })
	return cache
}

// GetOrCreate returns a cached module env, or nil if not yet loaded.
func (c *Cache) GetOrCreate(absPath string) *object.Environment {
	env, _ := c.modules.GetOrSetFunc(absPath, func() *object.Environment {
		return object.NewEnvironmentWithVM(c.vm)
	})
	return env
}

// Get returns a cached module env or nil.
func (c *Cache) Get(absPath string) *object.Environment {
	mod, _ := c.modules.Get(absPath)
	return mod
}

// ResolvePath resolves a module path relative to the base directory.
func ResolvePath(path, baseDir string) string {
	resolved, err := NewResolver("").Resolve(path, ResolveOptions{BaseDir: baseDir})
	if err != nil {
		if filepath.IsAbs(path) {
			return withDefaultExt(path)
		}
		if baseDir == "" {
			baseDir, _ = os.Getwd()
		}
		return withDefaultExt(filepath.Join(baseDir, path))
	}
	if resolved.Path != "" {
		return resolved.Path
	}
	return path
}

func withDefaultExt(path string) string {
	if filepath.Ext(path) == "" {
		return path + ".gs"
	}
	return path
}

// FindProjectRoot walks upward looking for a GoScript project root.
func FindProjectRoot(startDir string) string {
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return startDir
	}
	for {
		if fileExists(filepath.Join(dir, "project.toml")) || fileExists(filepath.Join(dir, ".git")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Clean(startDir)
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetupExports initializes module.exports on an environment.
func SetupExports(env *object.Environment) {
	exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	env.ObjectManager().Register(exports)
	env.Set("exports", exports)
	mod := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	env.ObjectManager().Register(mod)
	mod.SetMember(&object.String{Value: "exports"}, exports)
	env.Set("module", mod)
}

// GetExports returns the exports object from a module env.
func GetExports(env *object.Environment) object.Object {
	if modObj, ok := env.Get("module"); ok {
		if mod, ok := modObj.(*object.Hash); ok {
			if pair, ok := mod.Pairs[hashKey(&object.String{Value: "exports"})]; ok {
				return pair.Value
			}
		}
	}
	if v, ok := env.Get("exports"); ok {
		return v
	}
	return object.UNDEFINED
}

func hashKey(o object.Object) object.HashKey {
	switch o := o.(type) {
	case *object.String:
		return object.HashKey{Type: o.Type(), Value: o.Value}
	default:
		return object.HashKey{Type: o.Type(), Value: o.Inspect()}
	}
}
