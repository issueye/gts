package module

import (
	"os"
	"path/filepath"

	"github.com/issueye/goscript/internal/packagefile"
)

// ReadResolvedSource reads the source text of a resolved module, transparently
// handling std-source modules, modules nested inside a package archive, and
// regular files on disk. It is the single source of truth for "give me the
// source of this resolved module" used by the runtime Session, the SDK, and
// the CLI — previously this logic was duplicated in sdk/runtime_helpers.go and
// cmd/gs/main.go.
func ReadResolvedSource(resolved ResolvedModule) (string, error) {
	if resolved.Kind == ModuleKindStdSource {
		return ReadStdSource(resolved.Specifier)
	}
	if resolved.PackageFile != "" {
		return packagefile.ReadNestedText(resolved.PackageFile, resolved.ArchivePath)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ResolvedModuleDir returns the directory that relative imports inside a
// resolved module should resolve against: the std-source dir for std modules,
// an archive-scoped dir for packaged modules, or the on-disk parent directory
// otherwise.
func ResolvedModuleDir(resolved ResolvedModule) string {
	if resolved.Kind == ModuleKindStdSource {
		return StdSourceDir(resolved.Specifier)
	}
	if resolved.PackageFile != "" {
		return filepath.ToSlash(resolved.PackageFile) + "!" + filepath.ToSlash(filepath.Dir(resolved.ArchivePath))
	}
	return filepath.Dir(resolved.Path)
}
