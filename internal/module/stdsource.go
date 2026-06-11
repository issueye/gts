package module

import (
	"embed"
	"fmt"
	"path"
	"strings"
)

//go:embed std/*.gs
var stdSourceFS embed.FS

var stdSourceModules = map[string]string{
	"@std/orm": "std/orm.gs",
}

// IsStdSourceSpecifier reports whether a standard-library module is
// implemented in GoScript source rather than as a native Go module.
func IsStdSourceSpecifier(specifier string) bool {
	_, ok := stdSourceModules[specifier]
	return ok
}

// IsStdSourceDir reports whether a module base directory belongs to an
// embedded source-backed standard-library module.
func IsStdSourceDir(dir string) bool {
	return strings.HasPrefix(dir, "stdsrc/")
}

// ReadStdSource returns the embedded GoScript source for a source-backed
// standard-library module.
func ReadStdSource(specifier string) (string, error) {
	file, ok := stdSourceModules[specifier]
	if !ok {
		return "", fmt.Errorf("standard source module %s is not registered", specifier)
	}
	data, err := stdSourceFS.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// StdSourceFile returns a stable display name for diagnostics.
func StdSourceFile(specifier string) string {
	file, ok := stdSourceModules[specifier]
	if !ok {
		return specifier
	}
	return "std:" + file
}

// StdSourceDir returns a stable module directory for source-backed standard
// modules. These modules currently import native @std modules only, but a
// directory keeps relative imports well-defined if they are added later.
func StdSourceDir(specifier string) string {
	file, ok := stdSourceModules[specifier]
	if !ok {
		return "stdsrc"
	}
	dir := path.Dir(file)
	if dir == "." {
		return "stdsrc"
	}
	return "stdsrc/" + strings.TrimPrefix(strings.TrimSuffix(dir, "."), "std/")
}
