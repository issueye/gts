package stdlib

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/archive/zip", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initArchiveZipModule(exports)
		return exports, nil
	})
}

func initArchiveZipModule(exports *object.Hash) {
	setHashMember(exports, "list", &object.Builtin{Name: "zip.list", Fn: zipList})
	setHashMember(exports, "extract", &object.Builtin{Name: "zip.extract", Fn: zipExtract})
	setHashMember(exports, "create", &object.Builtin{Name: "zip.create", Fn: zipCreate})
}

func zipList(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "zip.list", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return object.NewError(pos, "zip.list: %v", err)
	}
	defer reader.Close()
	items := make([]object.Object, 0, len(reader.File))
	for _, file := range reader.File {
		info := file.FileInfo()
		item := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(item, "name", &object.String{Value: file.Name})
		setHashMember(item, "size", &object.Number{Value: float64(file.UncompressedSize64)})
		setHashMember(item, "compressedSize", &object.Number{Value: float64(file.CompressedSize64)})
		setHashMember(item, "isDir", object.NativeBool(info.IsDir()))
		setHashMember(item, "modified", &object.String{Value: file.Modified.Format("2006-01-02T15:04:05Z07:00")})
		items = append(items, item)
	}
	return &object.Array{Elements: items}
}

func zipExtract(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	archivePath, errObj := requiredString(pos, "zip.extract", args, 0, "archive path")
	if errObj != nil {
		return errObj
	}
	dest, errObj := requiredString(pos, "zip.extract", args, 1, "destination path")
	if errObj != nil {
		return errObj
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return object.NewError(pos, "zip.extract: %v", err)
	}
	defer reader.Close()
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return object.NewError(pos, "zip.extract: %v", err)
	}
	for _, file := range reader.File {
		target, err := safeZipTarget(cleanDest, file.Name)
		if err != nil {
			return object.NewError(pos, "zip.extract: %v", err)
		}
		info := file.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode()); err != nil {
				return object.NewError(pos, "zip.extract: %v", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return object.NewError(pos, "zip.extract: %v", err)
		}
		src, err := file.Open()
		if err != nil {
			return object.NewError(pos, "zip.extract: %v", err)
		}
		err = writeZipExtractedFile(target, src, info.Mode())
		_ = src.Close()
		if err != nil {
			return object.NewError(pos, "zip.extract: %v", err)
		}
	}
	return object.UNDEFINED
}

func zipCreate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "zip.create requires files array")
	}
	files, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "zip.create: files must be an array")
	}
	outPath, errObj := requiredString(pos, "zip.create", args, 1, "output path")
	if errObj != nil {
		return errObj
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return object.NewError(pos, "zip.create: %v", err)
	}
	out, err := os.Create(outPath)
	if err != nil {
		return object.NewError(pos, "zip.create: %v", err)
	}
	okWrite := false
	defer func() {
		_ = out.Close()
		if !okWrite {
			_ = os.Remove(outPath)
		}
	}()
	writer := zip.NewWriter(out)
	for i, item := range files.Elements {
		spec, errObj := zipFileSpec(pos, item, i)
		if errObj != nil {
			_ = writer.Close()
			return errObj
		}
		if err := addZipFile(writer, spec.path, spec.name); err != nil {
			_ = writer.Close()
			return object.NewError(pos, "zip.create: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		return object.NewError(pos, "zip.create: %v", err)
	}
	if err := out.Close(); err != nil {
		return object.NewError(pos, "zip.create: %v", err)
	}
	okWrite = true
	return object.UNDEFINED
}

type zipSpec struct {
	path string
	name string
}

func zipFileSpec(pos ast.Position, item object.Object, index int) (zipSpec, *object.Error) {
	hash, ok := item.(*object.Hash)
	if !ok {
		return zipSpec{}, object.NewError(pos, "zip.create: files[%d] must be an object", index)
	}
	pathObj, ok := hashValue(hash, "path")
	if !ok {
		return zipSpec{}, object.NewError(pos, "zip.create: files[%d].path is required", index)
	}
	path, ok := pathObj.(*object.String)
	if !ok {
		return zipSpec{}, object.NewError(pos, "zip.create: files[%d].path must be a string", index)
	}
	nameObj, ok := hashValue(hash, "name")
	if !ok {
		return zipSpec{path: path.Value, name: filepath.ToSlash(filepath.Base(path.Value))}, nil
	}
	name, ok := nameObj.(*object.String)
	if !ok {
		return zipSpec{}, object.NewError(pos, "zip.create: files[%d].name must be a string", index)
	}
	cleanName, err := cleanZipName(name.Value)
	if err != nil {
		return zipSpec{}, object.NewError(pos, "zip.create: files[%d].name %v", index, err)
	}
	return zipSpec{path: path.Value, name: cleanName}, nil
}

func addZipFile(writer *zip.Writer, path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(path, func(next string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(path, next)
			if err != nil {
				return err
			}
			return addZipFile(writer, next, filepath.ToSlash(filepath.Join(name, rel)))
		})
	}
	cleanName, err := cleanZipName(name)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = cleanName
	header.Method = zip.Deflate
	dst, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	_, err = io.Copy(dst, src)
	return err
}

func writeZipExtractedFile(path string, src io.Reader, mode os.FileMode) error {
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = dst.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	ok = true
	return nil
}

func cleanZipName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimLeft(name, "/")
	clean := filepath.ToSlash(filepath.Clean(name))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", os.ErrInvalid
	}
	return clean, nil
}

func safeZipTarget(dest, name string) (string, error) {
	cleanName, err := cleanZipName(name)
	if err != nil {
		return "", err
	}
	target := filepath.Join(dest, filepath.FromSlash(cleanName))
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != dest && !strings.HasPrefix(absTarget, dest+string(filepath.Separator)) {
		return "", os.ErrInvalid
	}
	return absTarget, nil
}
