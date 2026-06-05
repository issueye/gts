package module

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/issueye/goscript/internal/packagefile"
	"github.com/issueye/goscript/internal/proj"
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
	PackageFile string
	ArchivePath string
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
	mu          sync.RWMutex
	projectRoot map[string]string
	packageRoot map[string]string
	manifests   map[string]*proj.Config
	resolutions map[resolveCacheKey]resolveCacheEntry
}

type resolveCacheKey struct {
	Specifier   string
	ProjectRoot string
	BaseDir     string
	Referrer    string
}

type resolveCacheEntry struct {
	Resolved ResolvedModule
	Err      error
}

// NewResolver creates a resolver with default options.
func NewResolver(projectRoot string) *Resolver {
	return &Resolver{
		ProjectRoot: projectRoot,
		projectRoot: make(map[string]string),
		packageRoot: make(map[string]string),
		manifests:   make(map[string]*proj.Config),
		resolutions: make(map[resolveCacheKey]resolveCacheEntry),
	}
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
	baseDir = filepath.Clean(baseDir)

	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = r.ProjectRoot
	}
	if projectRoot == "" {
		projectRoot = r.findProjectRoot(baseDir)
	}
	if projectRoot != "" {
		projectRoot = filepath.Clean(projectRoot)
	}

	key := resolveCacheKey{
		Specifier:   specifier,
		ProjectRoot: projectRoot,
		BaseDir:     baseDir,
		Referrer:    opts.Referrer,
	}
	if resolved, ok, err := r.cachedResolution(key); ok {
		return resolved, err
	}

	resolved, err := r.resolveUncached(specifier, baseDir, projectRoot)
	r.storeResolution(key, resolved, err)
	return resolved, err
}

func (r *Resolver) resolveUncached(specifier, baseDir, projectRoot string) (ResolvedModule, error) {
	if pkgPath, archiveBase, ok := splitArchiveBaseDir(baseDir); ok {
		if isPathSpecifier(specifier) {
			return resolvePackageFileRelative(specifier, pkgPath, archiveBase)
		}
		if resolved, ok, err := tryResolvePackageFileImportAlias(specifier, pkgPath); ok || err != nil {
			return resolved, err
		}
		if !IsNativeSpecifier(specifier) && !strings.HasPrefix(specifier, "@agent/") {
			return r.resolvePackageFromPackageFile(specifier, pkgPath, projectRoot)
		}
	}

	if IsNativeSpecifier(specifier) {
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
		path, err := r.resolveSourcePath(base)
		if err != nil {
			return ResolvedModule{}, fmt.Errorf("module not found %q from %q: %w", specifier, baseDir, err)
		}
		return sourceModule(specifier, path, ModuleKindSource, "", ""), nil
	}

	if strings.HasPrefix(specifier, "@agent/") {
		agentRoot := r.findAgentRoot(baseDir)
		if agentRoot == "" {
			agentRoot = projectRoot
		}
		path, err := r.resolveSourcePath(filepath.Join(agentRoot, "scripts", "agent", strings.TrimPrefix(specifier, "@agent/")))
		if err != nil {
			return ResolvedModule{}, fmt.Errorf("module not found %q from @agent alias: %w", specifier, err)
		}
		return sourceModule(specifier, path, ModuleKindSource, "", ""), nil
	}

	if resolved, ok, err := r.tryResolveImportAlias(specifier, baseDir, projectRoot); ok || err != nil {
		return resolved, err
	}

	return r.resolvePackage(specifier, baseDir, projectRoot)
}

// IsNativeSpecifier reports whether a module specifier is reserved for a Go
// native module registered by the runtime or host application.
func IsNativeSpecifier(specifier string) bool {
	return strings.HasPrefix(specifier, "@std/") ||
		strings.HasPrefix(specifier, "@go/") ||
		strings.HasPrefix(specifier, "@host/") ||
		strings.HasPrefix(specifier, "@plugin/")
}

func (r *Resolver) cachedResolution(key resolveCacheKey) (ResolvedModule, bool, error) {
	r.mu.RLock()
	entry, ok := r.resolutions[key]
	r.mu.RUnlock()
	if !ok {
		return ResolvedModule{}, false, nil
	}
	return entry.Resolved, true, entry.Err
}

func (r *Resolver) storeResolution(key resolveCacheKey, resolved ResolvedModule, err error) {
	r.mu.Lock()
	r.resolutions[key] = resolveCacheEntry{Resolved: resolved, Err: err}
	r.mu.Unlock()
}

func resolvePackageFileRelative(specifier, pkgPath, archiveBase string) (ResolvedModule, error) {
	pkg, err := packagefile.Open(pkgPath)
	if err != nil {
		return ResolvedModule{}, err
	}
	defer pkg.Close()
	base := specifier
	if !strings.HasPrefix(base, "/") {
		base = filepath.ToSlash(filepath.Join(archiveBase, base))
	}
	path, err := resolveArchiveSourcePath(pkg, base)
	if err != nil {
		return ResolvedModule{}, fmt.Errorf("module not found %q from %q: %w", specifier, archiveBase, err)
	}
	name := packageNameFromManifest(pkg.Manifest, "")
	version := pkg.Manifest.Package.Version
	absPkg, _ := filepath.Abs(pkgPath)
	return ResolvedModule{
		ID:          packageFileID(name, version, "./"+path, absPkg, path),
		Kind:        ModuleKindPackage,
		Specifier:   specifier,
		Path:        archiveModulePath(absPkg, path),
		PackageRoot: absPkg,
		PackageFile: absPkg,
		ArchivePath: path,
		PackageName: name,
	}, nil
}

func tryResolvePackageFileImportAlias(specifier, pkgPath string) (ResolvedModule, bool, error) {
	pkg, err := openPackageFileRef(pkgPath)
	if err != nil {
		return ResolvedModule{}, false, err
	}
	defer pkg.Close()
	target, ok := matchPatternMap(pkg.Manifest.Imports, specifier)
	if !ok {
		return ResolvedModule{}, false, nil
	}
	path, err := resolveArchiveSourcePath(pkg, filepath.ToSlash(filepath.Join(pkg.Root, target)))
	if err != nil {
		return ResolvedModule{}, true, fmt.Errorf("package import %q not found: %w", specifier, err)
	}
	name := packageNameFromManifest(pkg.Manifest, "")
	version := pkg.Manifest.Package.Version
	absPkg := pkg.Path
	if !strings.Contains(absPkg, "!") {
		absPkg, _ = filepath.Abs(absPkg)
	}
	return ResolvedModule{
		ID:          packageFileID(name, version, specifier, absPkg, path),
		Kind:        ModuleKindPackage,
		Specifier:   specifier,
		Path:        archiveModulePath(absPkg, path),
		PackageRoot: absPkg,
		PackageFile: absPkg,
		ArchivePath: path,
		PackageName: name,
	}, true, nil
}

func openPackageFileRef(pkgPath string) (*packagefile.Package, error) {
	if !strings.Contains(pkgPath, "!") {
		return packagefile.Open(pkgPath)
	}
	parts := strings.Split(pkgPath, "!")
	pkg, err := packagefile.Open(parts[0])
	if err != nil {
		return nil, err
	}
	for _, nested := range parts[1:] {
		next, err := pkg.OpenNested(nested)
		if err != nil {
			_ = pkg.Close()
			return nil, err
		}
		_ = pkg.Close()
		pkg = next
	}
	return pkg, nil
}

func (r *Resolver) resolvePackageFromPackageFile(specifier, pkgPath, projectRoot string) (ResolvedModule, error) {
	pkg, err := packagefile.Open(pkgPath)
	if err != nil {
		return ResolvedModule{}, err
	}
	defer pkg.Close()

	packageName, exportName := splitPackageSpecifier(specifier)
	source, ok := pkg.Manifest.Dependencies[packageName]
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q is not listed in dependencies", packageName)
	}
	depPath, depArchivePath, depInArchive, err := packageFileDependencyPath(pkgPath, source)
	if err != nil {
		return ResolvedModule{}, err
	}
	if depInArchive {
		if isPackageFile(depArchivePath) {
			nested, err := pkg.OpenNested(depArchivePath)
			if err != nil {
				return ResolvedModule{}, err
			}
			pkg.Close()
			defer nested.Close()
			return resolveOpenedPackageFile(specifier, nested, packageName, exportName)
		}
		subpkg, err := pkg.Subpackage(depArchivePath)
		if err != nil {
			return ResolvedModule{}, err
		}
		return resolveOpenedPackageFile(specifier, subpkg, packageName, exportName)
	}
	if isPackageFile(depPath) {
		return resolvePackageFile(specifier, depPath, packageName, exportName)
	}
	depManifest, err := loadManifest(depPath)
	if err != nil && projectRoot != "" {
		depPath = filepath.Clean(filepath.Join(projectRoot, filepath.FromSlash(strings.TrimPrefix(strings.TrimPrefix(source, "file:"), "workspace:"))))
		depManifest, err = loadManifest(depPath)
	}
	if err != nil {
		return ResolvedModule{}, err
	}
	target, ok := exportTarget(depManifest, exportName)
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q has no export %q", packageName, exportName)
	}
	path, err := resolveSourcePath(filepath.Join(depPath, target))
	if err != nil {
		return ResolvedModule{}, fmt.Errorf("package %q export %q not found: %w", packageName, exportName, err)
	}
	resolved := sourceModule(specifier, path, ModuleKindPackage, depPath, packageNameFromManifest(depManifest, packageName))
	resolved.ID = packageID(resolved.PackageName, depManifest.Package.Version, exportName, path)
	return resolved, nil
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
	currentRoot := r.findPackageRoot(baseDir)
	if currentRoot == "" {
		currentRoot = projectRoot
	}
	if currentRoot == "" {
		currentRoot = r.findProjectRoot(baseDir)
	}
	manifest, err := r.loadManifest(currentRoot)
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
	if isPackageFile(depRoot) {
		return resolvePackageFile(specifier, depRoot, packageName, exportName)
	}

	depManifest, err := r.loadManifest(depRoot)
	if err != nil {
		return ResolvedModule{}, err
	}
	target, ok := exportTarget(depManifest, exportName)
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q has no export %q", packageName, exportName)
	}
	path, err := r.resolveSourcePath(filepath.Join(depRoot, target))
	if err != nil {
		return ResolvedModule{}, fmt.Errorf("package %q export %q not found: %w", packageName, exportName, err)
	}

	resolved := sourceModule(specifier, path, ModuleKindPackage, depRoot, packageNameFromManifest(depManifest, packageName))
	resolved.ID = packageID(resolved.PackageName, depManifest.Package.Version, exportName, path)
	return resolved, nil
}

func resolvePackageFile(specifier, pkgPath, packageName, exportName string) (ResolvedModule, error) {
	pkg, err := packagefile.Open(pkgPath)
	if err != nil {
		return ResolvedModule{}, err
	}
	defer pkg.Close()
	return resolveOpenedPackageFile(specifier, pkg, packageName, exportName)
}

func resolveOpenedPackageFile(specifier string, pkg *packagefile.Package, packageName, exportName string) (ResolvedModule, error) {
	target, ok := exportTarget(pkg.Manifest, exportName)
	if !ok {
		return ResolvedModule{}, fmt.Errorf("package %q has no export %q", packageName, exportName)
	}
	path, err := resolveArchiveSourcePath(pkg, filepath.ToSlash(filepath.Join(pkg.Root, target)))
	if err != nil {
		return ResolvedModule{}, fmt.Errorf("package %q export %q not found: %w", packageName, exportName, err)
	}
	name := packageNameFromManifest(pkg.Manifest, packageName)
	version := pkg.Manifest.Package.Version
	pkgPath := pkg.Path
	if !strings.Contains(pkgPath, "!") {
		pkgPath, _ = filepath.Abs(pkgPath)
	}
	if pkg.Root != "" && !strings.Contains(pkgPath, "!") {
		pkgPath = filepath.ToSlash(pkgPath) + "!" + filepath.ToSlash(pkg.Root)
	}
	return ResolvedModule{
		ID:          packageFileID(name, version, exportName, pkgPath, path),
		Kind:        ModuleKindPackage,
		Specifier:   specifier,
		Path:        archiveModulePath(pkgPath, path),
		PackageRoot: pkgPath,
		PackageFile: pkgPath,
		ArchivePath: path,
		PackageName: name,
	}, nil
}

func (r *Resolver) tryResolveImportAlias(specifier, baseDir, projectRoot string) (ResolvedModule, bool, error) {
	currentRoot := r.findPackageRoot(baseDir)
	if currentRoot == "" {
		currentRoot = projectRoot
	}
	if currentRoot == "" {
		currentRoot = r.findProjectRoot(baseDir)
	}
	manifest, err := r.loadManifest(currentRoot)
	if err != nil {
		return ResolvedModule{}, false, err
	}
	target, ok := matchPatternMap(manifest.Imports, specifier)
	if !ok {
		return ResolvedModule{}, false, nil
	}
	path, err := r.resolveSourcePath(filepath.Join(currentRoot, target))
	if err != nil {
		return ResolvedModule{}, true, fmt.Errorf("package import %q not found: %w", specifier, err)
	}
	return sourceModule(specifier, path, ModuleKindSource, currentRoot, packageNameFromManifest(manifest, "")), true, nil
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

func (r *Resolver) resolveSourcePath(path string) (string, error) {
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
		manifest, err := r.loadManifest(path)
		if err == nil {
			if main := packageMainFromManifest(manifest); main != "" {
				if resolved, err := r.resolveSourcePath(filepath.Join(path, main)); err == nil {
					return resolved, nil
				}
			}
		}
		if resolved, err := r.resolveSourcePath(filepath.Join(path, "index.gs")); err == nil {
			return resolved, nil
		}
	}
	return "", os.ErrNotExist
}

func resolveSourcePath(path string) (string, error) {
	return NewResolver("").resolveSourcePath(path)
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

func packageFileDependencyPath(pkgPath, source string) (path, archivePath string, inArchive bool, err error) {
	var rel string
	switch {
	case strings.HasPrefix(source, "file:"):
		rel = strings.TrimPrefix(source, "file:")
	case strings.HasPrefix(source, "workspace:"):
		rel = strings.TrimPrefix(source, "workspace:")
	default:
		return "", "", false, fmt.Errorf("unsupported dependency source %q", source)
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel), "", false, nil
	}
	if strings.Contains(pkgPath, "!") {
		rootPkg, rootArchive, _ := strings.Cut(pkgPath, "!")
		archiveRel := cleanArchiveSpecifier(filepath.ToSlash(filepath.Join(filepath.Dir(rootArchive), rel)))
		return filepath.Clean(filepath.FromSlash(rootPkg)), archiveRel, true, nil
	}
	archiveRel := cleanArchiveSpecifier(rel)
	return filepath.Clean(filepath.Join(filepath.Dir(pkgPath), filepath.FromSlash(rel))), archiveRel, true, nil
}

func isPackageFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), packagefile.Extension)
}

// FindPackageRoot walks upward looking for the nearest package manifest.
func FindPackageRoot(startDir string) string {
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for {
		if manifest, err := loadExistingManifest(dir); err == nil && isPackageManifest(manifest) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// FindAgentRoot walks upward looking for the repository-level script agent
// library. This intentionally differs from FindProjectRoot because examples and
// package projects can have their own project.toml files below the repository.
func FindAgentRoot(startDir string) string {
	if startDir == "" {
		startDir, _ = os.Getwd()
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for {
		if fileExists(filepath.Join(dir, "scripts", "agent")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func loadExistingManifest(root string) (*proj.Config, error) {
	path := filepath.Join(root, "project.toml")
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return loadManifest(root)
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

func packageFileID(name, version, exportName, pkgPath, archivePath string) string {
	prefix := "pkgfile:" + filepath.ToSlash(pkgPath)
	if name != "" {
		prefix = "pkg:" + name
		if version != "" {
			prefix += "@" + version
		}
	}
	return prefix + ":" + exportName + ":" + filepath.ToSlash(archivePath)
}

func archiveModulePath(pkgPath, archivePath string) string {
	return filepath.ToSlash(pkgPath) + "!" + filepath.ToSlash(archivePath)
}

func resolveArchiveSourcePath(pkg *packagefile.Package, path string) (string, error) {
	path = cleanArchiveSpecifier(path)
	candidates := []string{path}
	if filepath.Ext(path) == "" {
		candidates = append(candidates, path+".gs")
	}
	for _, candidate := range candidates {
		if pkg.HasFile(candidate) {
			return cleanArchiveSpecifier(candidate), nil
		}
	}
	if pkg.HasFile(cleanArchiveSpecifier(filepath.ToSlash(filepath.Join(path, "project.toml")))) {
		// Nested package manifests inside .gspkg are intentionally not followed in
		// the MVP; package archives expose one package root.
		return "", os.ErrNotExist
	}
	index := cleanArchiveSpecifier(filepath.ToSlash(filepath.Join(path, "index.gs")))
	if pkg.HasFile(index) {
		return index, nil
	}
	return "", os.ErrNotExist
}

func cleanArchiveSpecifier(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func (r *Resolver) findProjectRoot(startDir string) string {
	key := filepath.Clean(startDir)
	r.mu.RLock()
	root, ok := r.projectRoot[key]
	r.mu.RUnlock()
	if ok {
		return root
	}

	root = FindProjectRoot(startDir)
	r.mu.Lock()
	r.projectRoot[key] = root
	r.mu.Unlock()
	return root
}

func (r *Resolver) findPackageRoot(startDir string) string {
	key := filepath.Clean(startDir)
	r.mu.RLock()
	root, ok := r.packageRoot[key]
	r.mu.RUnlock()
	if ok {
		return root
	}

	root = FindPackageRoot(startDir)
	r.mu.Lock()
	r.packageRoot[key] = root
	r.mu.Unlock()
	return root
}

func (r *Resolver) findAgentRoot(startDir string) string {
	key := "agent:" + filepath.Clean(startDir)
	r.mu.RLock()
	root, ok := r.projectRoot[key]
	r.mu.RUnlock()
	if ok {
		return root
	}

	root = FindAgentRoot(startDir)
	r.mu.Lock()
	r.projectRoot[key] = root
	r.mu.Unlock()
	return root
}

func (r *Resolver) loadManifest(root string) (*proj.Config, error) {
	root = filepath.Clean(root)
	r.mu.RLock()
	manifest, ok := r.manifests[root]
	r.mu.RUnlock()
	if ok {
		return manifest, nil
	}

	manifest, err := loadManifest(root)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.manifests[root] = manifest
	r.mu.Unlock()
	return manifest, nil
}

func splitArchiveBaseDir(baseDir string) (pkgPath, archiveDir string, ok bool) {
	idx := strings.LastIndex(baseDir, "!")
	if idx < 0 {
		return "", "", false
	}
	pkgPath = filepath.FromSlash(baseDir[:idx])
	archiveDir = cleanArchiveSpecifier(baseDir[idx+1:])
	return pkgPath, archiveDir, pkgPath != ""
}

func loadManifest(root string) (*proj.Config, error) {
	manifest, err := proj.LoadStrict(filepath.Join(root, "project.toml"))
	if err != nil {
		return nil, err
	}
	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}
	if manifest.Imports == nil {
		manifest.Imports = make(map[string]string)
	}
	return manifest, nil
}

func isPackageManifest(m *proj.Config) bool {
	return m.Package.Name != "" ||
		m.Package.Main != "" ||
		len(m.Exports) > 0 ||
		len(m.Imports) > 0 ||
		len(m.Dependencies) > 0 ||
		m.Entry != ""
}

func packageMainFromManifest(m *proj.Config) string {
	if m.Package.Main != "" {
		return m.Package.Main
	}
	if m.Entry != "" {
		return m.Entry
	}
	return ""
}

func packageNameFromManifest(m *proj.Config, fallback string) string {
	if m.Package.Name != "" {
		return m.Package.Name
	}
	if m.Name != "" {
		return m.Name
	}
	return fallback
}

func exportTarget(m *proj.Config, exportName string) (string, bool) {
	if len(m.Exports) == 0 {
		main := packageMainFromManifest(m)
		if main == "" {
			main = "index.gs"
		}
		return main, exportName == "."
	}
	return matchPatternMap(m.Exports, exportName)
}

func matchPatternMap(mapping map[string]string, name string) (string, bool) {
	if target, ok := mapping[name]; ok {
		return target, true
	}
	for pattern, target := range mapping {
		if !strings.Contains(pattern, "*") {
			continue
		}
		prefix, suffix, _ := strings.Cut(pattern, "*")
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
			continue
		}
		matched := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
		return strings.Replace(target, "*", matched, 1), true
	}
	return "", false
}
