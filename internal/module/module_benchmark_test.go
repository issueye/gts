package module

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/issueye/goscript/internal/object"
)

func BenchmarkModuleCacheHit(b *testing.B) {
	vm := object.NewVirtualMachine()
	cache := NewCacheWithVM(vm)
	path := filepath.Join(b.TempDir(), "lib.gs")
	cache.GetOrCreate(path)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if env := cache.Get(path); env == nil {
			b.Fatal("cache miss")
		}
	}
}

func BenchmarkResolverRelativeSource(b *testing.B) {
	dir := b.TempDir()
	writeBenchmarkFile(b, filepath.Join(dir, "lib.gs"), "")
	resolver := NewResolver("")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved, err := resolver.Resolve("./lib", ResolveOptions{BaseDir: dir})
		if err != nil {
			b.Fatal(err)
		}
		if resolved.Kind != ModuleKindSource {
			b.Fatalf("want source module, got %s", resolved.Kind)
		}
	}
}

func BenchmarkResolverPackageExport(b *testing.B) {
	root := b.TempDir()
	dep := filepath.Join(root, "vendor", "tools")
	writeBenchmarkFile(b, filepath.Join(root, "project.toml"), `[project]
name = "app"

[dependencies]
"tools" = "file:vendor/tools"
`)
	writeBenchmarkFile(b, filepath.Join(dep, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
"./feature" = "src/feature.gs"
`)
	writeBenchmarkFile(b, filepath.Join(dep, "src", "index.gs"), "")
	writeBenchmarkFile(b, filepath.Join(dep, "src", "feature.gs"), "")
	resolver := NewResolver(root)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolved, err := resolver.Resolve("tools/feature", ResolveOptions{ProjectRoot: root, BaseDir: root})
		if err != nil {
			b.Fatal(err)
		}
		if resolved.Kind != ModuleKindPackage {
			b.Fatalf("want package module, got %s", resolved.Kind)
		}
	}
}

func writeBenchmarkFile(b *testing.B, path, contents string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		b.Fatal(err)
	}
}
