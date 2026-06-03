package packagefile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/proj"
)

const Extension = ".gspkg"

type Package struct {
	Path     string
	Root     string
	Manifest *proj.Config
	reader   *zip.ReadCloser
	zip      *zip.Reader
	files    map[string]*zip.File
}

func Open(path string) (*Package, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	reader, err := zip.OpenReader(abs)
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		Path:   abs,
		reader: reader,
		zip:    &reader.Reader,
		files:  make(map[string]*zip.File),
	}
	if err := pkg.indexFiles(); err != nil {
		_ = reader.Close()
		return nil, err
	}
	if err := pkg.loadManifest(); err != nil {
		_ = reader.Close()
		return nil, err
	}
	return pkg, nil
}

func OpenBytes(name string, data []byte) (*Package, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		Path:  name,
		zip:   reader,
		files: make(map[string]*zip.File),
	}
	if err := pkg.indexFiles(); err != nil {
		return nil, err
	}
	if err := pkg.loadManifest(); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *Package) Close() error {
	if p == nil || p.reader == nil {
		return nil
	}
	return p.reader.Close()
}

func (p *Package) ReadText(path string) (string, error) {
	data, err := p.ReadBytes(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *Package) ReadBytes(path string) ([]byte, error) {
	file, ok := p.files[cleanArchivePath(path)]
	if !ok {
		return nil, os.ErrNotExist
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *Package) HasFile(path string) bool {
	_, ok := p.files[cleanArchivePath(path)]
	return ok
}

func (p *Package) Subpackage(root string) (*Package, error) {
	root = cleanArchivePath(root)
	if root == "" {
		return p, nil
	}
	pkg := &Package{
		Path:  p.Path + "!" + root,
		Root:  root,
		zip:   p.zip,
		files: p.files,
	}
	if err := pkg.loadManifest(); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (p *Package) OpenNested(path string) (*Package, error) {
	data, err := p.ReadBytes(path)
	if err != nil {
		return nil, err
	}
	return OpenBytes(p.Path+"!"+cleanArchivePath(path), data)
}

func ReadNestedText(packagePath, archivePath string) (string, error) {
	parts := strings.Split(packagePath, "!")
	if len(parts) == 0 {
		return "", os.ErrNotExist
	}
	pkg, err := Open(parts[0])
	if err != nil {
		return "", err
	}
	defer pkg.Close()
	for _, nestedPath := range parts[1:] {
		next, err := pkg.OpenNested(nestedPath)
		if err != nil {
			return "", err
		}
		defer next.Close()
		pkg = next
	}
	return pkg.ReadText(archivePath)
}

func (p *Package) indexFiles() error {
	if p.zip == nil {
		return fmt.Errorf("package %q has no zip reader", p.Path)
	}
	for _, file := range p.zip.File {
		name := cleanArchivePath(file.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		p.files[name] = file
	}
	return nil
}

func (p *Package) loadManifest() error {
	manifestPath := cleanArchivePath(filepath.ToSlash(filepath.Join(p.Root, "project.toml")))
	manifestSrc, err := p.ReadText(manifestPath)
	if err != nil {
		return fmt.Errorf("package %q is missing project.toml: %w", p.Path, err)
	}
	manifest, err := proj.Parse(manifestSrc, manifestPath)
	if err != nil {
		return fmt.Errorf("package %q has invalid manifest: %w", p.Path, err)
	}
	p.Manifest = manifest
	return nil
}

func PackDirectory(root, out string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(absRoot, "project.toml")); err != nil {
		return fmt.Errorf("package root %q must contain project.toml: %w", absRoot, err)
	}
	if out == "" {
		out = filepath.Base(absRoot) + Extension
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		return err
	}
	tmp := absOut + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zipper := zip.NewWriter(file)
	walkErr := filepath.WalkDir(absRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Clean(path) == filepath.Clean(absOut) || filepath.Clean(path) == filepath.Clean(tmp) {
			return nil
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		return addFile(zipper, path, filepath.ToSlash(rel))
	})
	closeErr := zipper.Close()
	fileErr := file.Close()
	if walkErr != nil {
		_ = os.Remove(tmp)
		return walkErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if fileErr != nil {
		_ = os.Remove(tmp)
		return fileErr
	}
	return os.Rename(tmp, absOut)
}

func addFile(zipper *zip.Writer, path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = cleanArchivePath(name)
	header.Method = zip.Deflate
	writer, err := zipper.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}

func cleanArchivePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." {
		return ""
	}
	return cleaned
}
