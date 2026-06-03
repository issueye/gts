package packagefile

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/proj"
)

const Extension = ".gspkg"

var ErrNoAppendedPackage = errors.New("no appended GoScript package")

var appendedPackageMagic = []byte("GOSCRIPT-PKG-v1")

type Package struct {
	Path     string
	Root     string
	Manifest *proj.Config
	reader   *zip.ReadCloser
	zip      *zip.Reader
	files    map[string]*zip.File
}

func Open(path string) (*Package, error) {
	if strings.Contains(path, "!") {
		return openPackageRef(path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	reader, err := zip.OpenReader(abs)
	if err != nil {
		data, appendedErr := ReadAppendedPackage(abs)
		if appendedErr != nil {
			return nil, err
		}
		return OpenBytes(abs, data)
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
	cleaned := cleanArchivePath(path)
	if p.Root != "" && cleaned != p.Root && !strings.HasPrefix(cleaned, p.Root+"/") {
		cleaned = cleanArchivePath(filepath.ToSlash(filepath.Join(p.Root, cleaned)))
	}
	file, ok := p.files[cleaned]
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
		Path:   p.Path + "!" + root,
		Root:   root,
		reader: p.reader,
		zip:    p.zip,
		files:  p.files,
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
	pkg, err := Open(packagePath)
	if err != nil {
		return "", err
	}
	defer pkg.Close()
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

func openPackageRef(path string) (*Package, error) {
	parts := strings.Split(path, "!")
	if len(parts) == 0 || parts[0] == "" {
		return nil, os.ErrNotExist
	}
	pkg, err := Open(parts[0])
	if err != nil {
		return nil, err
	}
	for _, part := range parts[1:] {
		part = cleanArchivePath(part)
		if part == "" {
			continue
		}
		var next *Package
		if strings.EqualFold(filepath.Ext(part), Extension) {
			if pkg.Root != "" && !strings.HasPrefix(part, pkg.Root+"/") {
				part = cleanArchivePath(filepath.ToSlash(filepath.Join(pkg.Root, part)))
			}
			next, err = pkg.OpenNested(part)
			_ = pkg.Close()
		} else {
			next, err = pkg.Subpackage(part)
		}
		if err != nil {
			return nil, err
		}
		pkg = next
	}
	return pkg, nil
}

func AppendPackageToExecutable(stubPath, packagePath, out string) error {
	if out == "" {
		return fmt.Errorf("output executable path is required")
	}
	absStub, err := filepath.Abs(stubPath)
	if err != nil {
		return err
	}
	absPackage, err := filepath.Abs(packagePath)
	if err != nil {
		return err
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	if filepath.Clean(absStub) == filepath.Clean(absOut) {
		return fmt.Errorf("output executable must differ from the stub executable")
	}
	if filepath.Clean(absPackage) == filepath.Clean(absOut) {
		return fmt.Errorf("output executable must differ from the package file")
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0755); err != nil {
		return err
	}

	stub, err := os.Open(absStub)
	if err != nil {
		return err
	}
	defer stub.Close()
	stubInfo, err := stub.Stat()
	if err != nil {
		return err
	}
	stubSize := stubInfo.Size()
	if offset, _, err := appendedPackageOffset(stub, stubSize); err == nil {
		stubSize = offset
	} else if !errors.Is(err, ErrNoAppendedPackage) {
		return err
	}
	if _, err := stub.Seek(0, io.SeekStart); err != nil {
		return err
	}

	pkg, err := os.Open(absPackage)
	if err != nil {
		return err
	}
	defer pkg.Close()
	pkgInfo, err := pkg.Stat()
	if err != nil {
		return err
	}

	tmp := absOut + ".tmp"
	outFile, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyStubErr := io.CopyN(outFile, stub, stubSize)
	_, copyPkgErr := io.Copy(outFile, pkg)
	trailerErr := writeAppendedPackageTrailer(outFile, uint64(pkgInfo.Size()))
	closeErr := outFile.Close()
	if copyStubErr != nil || copyPkgErr != nil || trailerErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyStubErr != nil {
			return copyStubErr
		}
		if copyPkgErr != nil {
			return copyPkgErr
		}
		if trailerErr != nil {
			return trailerErr
		}
		return closeErr
	}
	if err := os.Chmod(tmp, stubInfo.Mode().Perm()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Remove(absOut); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, absOut)
}

func ReadAppendedPackage(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	offset, size, err := appendedPackageOffset(file, info.Size())
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, err
	}
	return data, nil
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

func appendedPackageOffset(file *os.File, fileSize int64) (int64, uint64, error) {
	trailerSize := int64(8 + len(appendedPackageMagic))
	if fileSize < trailerSize {
		return 0, 0, ErrNoAppendedPackage
	}
	trailer := make([]byte, trailerSize)
	if _, err := file.Seek(fileSize-trailerSize, io.SeekStart); err != nil {
		return 0, 0, err
	}
	if _, err := io.ReadFull(file, trailer); err != nil {
		return 0, 0, err
	}
	if !bytes.Equal(trailer[8:], appendedPackageMagic) {
		return 0, 0, ErrNoAppendedPackage
	}
	size := binary.LittleEndian.Uint64(trailer[:8])
	if size > uint64(fileSize-trailerSize) {
		return 0, 0, fmt.Errorf("invalid appended GoScript package size %d", size)
	}
	return fileSize - trailerSize - int64(size), size, nil
}

func writeAppendedPackageTrailer(w io.Writer, packageSize uint64) error {
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], packageSize)
	if _, err := w.Write(size[:]); err != nil {
		return err
	}
	_, err := w.Write(appendedPackageMagic)
	return err
}
