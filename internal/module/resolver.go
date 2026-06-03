package module

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ModuleKind classifies how a module specifier was resolved.
type ModuleKind string

const (
	ModuleKindNative   ModuleKind = "native"
	ModuleKindSource   ModuleKind = "source"
	ModuleKindPackage  ModuleKind = "package"
	ModuleKindExternal ModuleKind = "external"
)

// ResolvedModule describes the resolved identity and backing location of a
// module specifier.
type ResolvedModule struct {
	ID          string
	Kind        ModuleKind
	Specifier   string
	Path        string
	PackageRoot string
	PackageName string
	External    bool
}

// ResolveOptions configures module resolution.
type ResolveOptions struct {
	ProjectRoot string
	BaseDir     string
	Referrer    string
}

// Resolver resolves GoScript module specifiers.
type Resolver struct {
	ProjectRoot string
}

// NewResolver creates a resolver with default options.
func NewResolver(projectRoot string) *Resolver {
	return &Resolver{ProjectRoot: projectRoot}
}

// Resolve resolves a module specifier from a referrer or base directory.
func (r *Resolver) Resolve(specifier string, opts ResolveOptions) (ResolvedModule, error) {
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = r.baseDirFromReferrer(opts.Referrer)
	}
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return ResolvedModule{}, err
		}
	}

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = r.ProjectRoot
	}
	if projectRoot == "" {
		projectRoot = FindProjectRoot(baseDir)
	}

	if strings.HasPrefix(specifier, "@std/") {
		return ResolvedModule{
			ID:        "native:" + specifier,
			Kind:      ModuleKindNative,
			Specifier: specifier,
			External:  true,
		}, nil
	}

	if isPathSpecifier(specifier) {
		base := specifier
		if !filepath.IsAbs(base) {
			base = filepath.Join(baseDir, base)
		}
		path, err := resolveSourcePath(base)
		if err != nil {
			return ResolvedModule{}, fmt.Errorf("module not found %q from %q: %w", specifier, baseDir, err)
		}
		return sourceModule(specifier, path, ModuleKindSource, "", ""), nil
	}

	if strings.HasPrefix(specifier, "@agent/") {
		path, err := resolveSourcePath(filepath.Join(projectRoot, "scripts", "agent", strings.TrimPrefix(specifier, "@agent/")))
		if err != nil {
			return ResolvedModule{}, fmt.Errorf("module not found %q from @agent alias: %w", specifier, err)
		}
		return sourceModule(specifier, path, ModuleKindSource, "", ""), nil
	}

	return r.resolvePackage(specifier, baseDir, projectRoot)
}

func (r *Resolver) baseDirFromReferrer(referrer string) string {
	if referrer == "" {
		return ""
	}
	if info, err := os.Stat(referrer); err == nil && info.IsDir() {
		return referrer
	}
	return filepath.Dir(referrer)
}

func (r *Resolver) resolvePackage(specifier, baseDir, projectRoot string) (ResolvedModule, error) {
	currentRoot := projectRoot
	if currentRoot == "" {
		currentRoot = FindProjectRoot(baseDir)
	}
	manifest, err := loadManifest(filepath.Join(currentRoot, "project.toml"))
	if err != nil {
		return ResolvedModule{}, err
	}

	packageName, exportName := splitPackageSpecifier(specifier)
	source, ok := manifest.Dependencies[packageName]
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q is not listed in dependencies", packageName)
	}
	depRoot, err := dependencyRoot(currentRoot, source)
	if err != nil {
		return ResolvedModule{}, err
	}

	depManifest, err := loadManifest(filepath.Join(depRoot, "project.toml"))
	if err != nil {
		return ResolvedModule{}, err
	}
	target, ok := depManifest.exportTarget(exportName)
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q has no export %q", packageName, exportName)
	}
	path, err := resolveSourcePath(filepath.Join(depRoot, target))
	if err != nil {
		return ResolvedModule{}, fmt.Errorf("package %q export %q not found: %w", packageName, exportName, err)
	}

	resolved := sourceModule(specifier, path, ModuleKindPackage, depRoot, depManifest.packageName(packageName))
	resolved.ID = packageID(resolved.PackageName, depManifest.Package.Version, exportName, path)
	return resolved, nil
}

func isPathSpecifier(specifier string) bool {
	return specifier == "." ||
		specifier == ".." ||
		strings.HasPrefix(specifier, "./") ||
		strings.HasPrefix(specifier, "../") ||
		strings.HasPrefix(specifier, ".\\") ||
		strings.HasPrefix(specifier, "..\\") ||
		filepath.IsAbs(specifier) ||
		(runtime.GOOS != "windows" && len(specifier) >= 3 && specifier[1] == ':' && (specifier[2] == '\\' || specifier[2] == '/'))
}

func resolveSourcePath(path string) (string, error) {
	path = filepath.Clean(path)
	candidates := []string{path}
	if filepath.Ext(path) == "" {
		candidates = append(candidates, path+".gs")
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return filepath.Abs(candidate)
		}
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		manifest, err := loadManifest(filepath.Join(path, "project.toml"))
		if err == nil {
			if main := manifest.packageMain(); main != "" {
				if resolved, err := resolveSourcePath(filepath.Join(path, main)); err == nil {
					return resolved, nil
				}
			}
		}
		if resolved, err := resolveSourcePath(filepath.Join(path, "index.gs")); err == nil {
			return resolved, nil
		}
	}
	return "", os.ErrNotExist
}

func sourceModule(specifier, path string, kind ModuleKind, packageRoot, packageName string) ResolvedModule {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	return ResolvedModule{
		ID:          "file:" + filepath.ToSlash(path),
		Kind:        kind,
		Specifier:   specifier,
		Path:        path,
		PackageRoot: packageRoot,
		PackageName: packageName,
	}
}

func splitPackageSpecifier(specifier string) (name, exportName string) {
	exportName = "."
	parts := strings.Split(specifier, "/")
	if strings.HasPrefix(specifier, "@") && len(parts) >= 2 {
		name = strings.Join(parts[:2], "/")
		if len(parts) > 2 {
			exportName = "./" + strings.Join(parts[2:], "/")
		}
		return name, exportName
	}
	name = parts[0]
	if len(parts) > 1 {
		exportName = "./" + strings.Join(parts[1:], "/")
	}
	return name, exportName
}

func dependencyRoot(projectRoot, source string) (string, error) {
	var rel string
	switch {
	case strings.HasPrefix(source, "file:"):
		rel = strings.TrimPrefix(source, "file:")
	case strings.HasPrefix(source, "workspace:"):
		rel = strings.TrimPrefix(source, "workspace:")
	default:
		return "", fmt.Errorf("unsupported dependency source %q", source)
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), nil
	}
	return filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(rel))), nil
}

func packageID(name, version, exportName, path string) string {
	if name == "" {
		return "file:" + filepath.ToSlash(path)
	}
	if version == "" {
		return "pkg:" + name + ":" + exportName
	}
	return "pkg:" + name + "@" + version + ":" + exportName
}

type manifestFile struct {
	Project      projectSection    `toml:"project"`
	Package      packageSection    `toml:"package"`
	Exports      map[string]string `toml:"exports"`
	Dependencies map[string]string `toml:"dependencies"`
}

type projectSection struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Entry   string `toml:"entry"`
}

type packageSection struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Main    string `toml:"main"`
}

func loadManifest(path string) (manifestFile, error) {
	var manifest manifestFile
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("invalid manifest %q: %w", path, err)
	}
	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}
	return manifest, nil
}

func (m manifestFile) packageMain() string {
	if m.Package.Main != "" {
		return m.Package.Main
	}
	if m.Project.Entry != "" {
		return m.Project.Entry
	}
	return ""
}

func (m manifestFile) packageName(fallback string) string {
	if m.Package.Name != "" {
		return m.Package.Name
	}
	if m.Project.Name != "" {
		return m.Project.Name
	}
	return fallback
}

func (m manifestFile) exportTarget(exportName string) (string, bool) {
	if len(m.Exports) == 0 {
		main := m.packageMain()
		if main == "" {
			main = "index.gs"
		}
		return main, exportName == "."
	}
	if target, ok := m.Exports[exportName]; ok {
		return target, true
	}
	for pattern, target := range m.Exports {
		if !strings.Contains(pattern, "*") {
			continue
		}
		prefix, suffix, _ := strings.Cut(pattern, "*")
		if !strings.HasPrefix(exportName, prefix) || !strings.HasSuffix(exportName, suffix) {
			continue
		}
		matched := strings.TrimSuffix(strings.TrimPrefix(exportName, prefix), suffix)
		return strings.Replace(target, "*", matched, 1), true
	}
	return "", false
}
