package stdlib

import (
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/fs", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initFSModule(exports)
		return exports, nil
	})
}

func initFSModule(exports *object.Hash) {
	setHashMember(exports, "readFileSync", &object.Builtin{Name: "fs.readFileSync", Fn: fsReadFileSync})
	setHashMember(exports, "readTextSync", &object.Builtin{Name: "fs.readTextSync", Fn: fsReadFileSync})
	setHashMember(exports, "writeFileSync", &object.Builtin{Name: "fs.writeFileSync", Fn: fsWriteFileSync})
	setHashMember(exports, "writeTextSync", &object.Builtin{Name: "fs.writeTextSync", Fn: fsWriteFileSync})
	setHashMember(exports, "appendFileSync", &object.Builtin{Name: "fs.appendFileSync", Fn: fsAppendFileSync})
	setHashMember(exports, "appendTextSync", &object.Builtin{Name: "fs.appendTextSync", Fn: fsAppendFileSync})
	setHashMember(exports, "writeFileAtomicSync", &object.Builtin{Name: "fs.writeFileAtomicSync", Fn: fsWriteFileAtomicSync})
	setHashMember(exports, "existsSync", &object.Builtin{Name: "fs.existsSync", Fn: fsExistsSync})
	setHashMember(exports, "readdirSync", &object.Builtin{Name: "fs.readdirSync", Fn: fsReaddirSync})
	setHashMember(exports, "walkSync", &object.Builtin{Name: "fs.walkSync", Fn: fsWalkSync})
	setHashMember(exports, "globSync", &object.Builtin{Name: "fs.globSync", Fn: fsGlobSync})
	setHashMember(exports, "copyFileSync", &object.Builtin{Name: "fs.copyFileSync", Fn: fsCopyFileSync})
	setHashMember(exports, "rmSync", &object.Builtin{Name: "fs.rmSync", Fn: fsRmSync})
	setHashMember(exports, "mkdtempSync", &object.Builtin{Name: "fs.mkdtempSync", Fn: fsMkdtempSync})
	setHashMember(exports, "realpathSync", &object.Builtin{Name: "fs.realpathSync", Fn: fsRealpathSync})
	setHashMember(exports, "lstatSync", &object.Builtin{Name: "fs.lstatSync", Fn: fsLstatSync})
	setHashMember(exports, "mkdirSync", &object.Builtin{Name: "fs.mkdirSync", Fn: fsMkdirSync})
	setHashMember(exports, "statSync", &object.Builtin{Name: "fs.statSync", Fn: fsStatSync})
	setHashMember(exports, "renameSync", &object.Builtin{Name: "fs.renameSync", Fn: fsRenameSync})
	setHashMember(exports, "unlinkSync", &object.Builtin{Name: "fs.unlinkSync", Fn: fsUnlinkSync})
}

func fsReadFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.readFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "fs.readFileSync: %v", err)
	}
	return &object.String{Value: string(data)}
}

func fsWriteFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.writeFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "fs.writeFileSync requires data")
	}
	data := args[1].Inspect()
	if s, ok := args[1].(*object.String); ok {
		data = s.Value
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return object.NewError(pos, "fs.writeFileSync: %v", err)
	}
	return object.UNDEFINED
}

func fsAppendFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.appendFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "fs.appendFileSync requires data")
	}
	data := objectToText(args[1])
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return object.NewError(pos, "fs.appendFileSync: %v", err)
	}
	defer file.Close()
	if _, err := file.WriteString(data); err != nil {
		return object.NewError(pos, "fs.appendFileSync: %v", err)
	}
	return object.UNDEFINED
}

func fsWriteFileAtomicSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.writeFileAtomicSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "fs.writeFileAtomicSync requires data")
	}
	data := objectToText(args[1])
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return object.NewError(pos, "fs.writeFileAtomicSync: %v", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return object.NewError(pos, "fs.writeFileAtomicSync: %v", err)
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.WriteString(data); err != nil {
		_ = tmp.Close()
		return object.NewError(pos, "fs.writeFileAtomicSync: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return object.NewError(pos, "fs.writeFileAtomicSync: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return object.NewError(pos, "fs.writeFileAtomicSync: %v", err)
	}
	ok = true
	return object.UNDEFINED
}

func fsExistsSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.existsSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	_, err := os.Stat(path)
	return object.NativeBool(err == nil)
}

func fsReaddirSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.readdirSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return object.NewError(pos, "fs.readdirSync: %v", err)
	}
	withFileTypes := false
	if len(args) >= 2 {
		if opts, ok := args[1].(*object.Hash); ok {
			if v, ok := hashValue(opts, "withFileTypes"); ok {
				if b, ok := v.(*object.Boolean); ok {
					withFileTypes = b.Value
				}
			}
		}
	}
	elements := make([]object.Object, len(entries))
	for i, entry := range entries {
		if withFileTypes {
			info, err := entry.Info()
			if err != nil {
				return object.NewError(pos, "fs.readdirSync: %v", err)
			}
			item := statObject(filepath.Join(path, entry.Name()), info)
			setHashMember(item, "name", &object.String{Value: entry.Name()})
			elements[i] = item
		} else {
			elements[i] = &object.String{Value: entry.Name()}
		}
	}
	return &object.Array{Elements: elements}
}

func fsWalkSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	root, errObj := requiredString(pos, "fs.walkSync", args, 0, "root")
	if errObj != nil {
		return errObj
	}
	includeDirs := true
	if len(args) >= 2 {
		if opts, ok := args[1].(*object.Hash); ok {
			if v, ok := hashValue(opts, "includeDirs"); ok {
				if b, ok := v.(*object.Boolean); ok {
					includeDirs = b.Value
				}
			}
		}
	}
	entries := []object.Object{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() && !includeDirs {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		item := statObject(path, info)
		setHashMember(item, "relativePath", &object.String{Value: rel})
		entries = append(entries, item)
		return nil
	})
	if err != nil {
		return object.NewError(pos, "fs.walkSync: %v", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Inspect() < entries[j].Inspect()
	})
	return &object.Array{Elements: entries}
}

func fsGlobSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	pattern, errObj := requiredString(pos, "fs.globSync", args, 0, "pattern")
	if errObj != nil {
		return errObj
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return object.NewError(pos, "fs.globSync: %v", err)
	}
	sort.Strings(matches)
	return strSliceToArray(matches)
}

func fsCopyFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	from, errObj := requiredString(pos, "fs.copyFileSync", args, 0, "from path")
	if errObj != nil {
		return errObj
	}
	to, errObj := requiredString(pos, "fs.copyFileSync", args, 1, "to path")
	if errObj != nil {
		return errObj
	}
	src, err := os.Open(from)
	if err != nil {
		return object.NewError(pos, "fs.copyFileSync: %v", err)
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		return object.NewError(pos, "fs.copyFileSync: %v", err)
	}
	dst, err := os.Create(to)
	if err != nil {
		return object.NewError(pos, "fs.copyFileSync: %v", err)
	}
	ok := false
	defer func() {
		_ = dst.Close()
		if !ok {
			_ = os.Remove(to)
		}
	}()
	if _, err := io.Copy(dst, src); err != nil {
		return object.NewError(pos, "fs.copyFileSync: %v", err)
	}
	if err := dst.Close(); err != nil {
		return object.NewError(pos, "fs.copyFileSync: %v", err)
	}
	ok = true
	return object.UNDEFINED
}

func fsRmSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.rmSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	recursive := false
	force := false
	if len(args) >= 2 {
		if opts, ok := args[1].(*object.Hash); ok {
			if v, ok := hashValue(opts, "recursive"); ok {
				if b, ok := v.(*object.Boolean); ok {
					recursive = b.Value
				}
			}
			if v, ok := hashValue(opts, "force"); ok {
				if b, ok := v.(*object.Boolean); ok {
					force = b.Value
				}
			}
		}
	}
	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		if force && os.IsNotExist(err) {
			return object.UNDEFINED
		}
		return object.NewError(pos, "fs.rmSync: %v", err)
	}
	return object.UNDEFINED
}

func fsMkdtempSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	prefix, errObj := requiredString(pos, "fs.mkdtempSync", args, 0, "prefix")
	if errObj != nil {
		return errObj
	}
	dir := filepath.Dir(prefix)
	base := filepath.Base(prefix)
	if dir == "" {
		dir = "."
	}
	path, err := os.MkdirTemp(dir, base)
	if err != nil {
		return object.NewError(pos, "fs.mkdtempSync: %v", err)
	}
	return &object.String{Value: path}
}

func fsRealpathSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.realpathSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return object.NewError(pos, "fs.realpathSync: %v", err)
	}
	abs, err := filepath.Abs(real)
	if err != nil {
		return object.NewError(pos, "fs.realpathSync: %v", err)
	}
	return &object.String{Value: abs}
}

func fsLstatSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.lstatSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	info, err := os.Lstat(path)
	if err != nil {
		return object.NewError(pos, "fs.lstatSync: %v", err)
	}
	return statObject(path, info)
}

func fsMkdirSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.mkdirSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	recursive := false
	if len(args) >= 2 {
		switch v := args[1].(type) {
		case *object.Boolean:
			recursive = v.Value
		case *object.Hash:
			if opt, ok := hashValue(v, "recursive"); ok {
				if b, ok := opt.(*object.Boolean); ok {
					recursive = b.Value
				}
			}
		}
	}
	var err error
	if recursive {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}
	if err != nil {
		return object.NewError(pos, "fs.mkdirSync: %v", err)
	}
	return object.UNDEFINED
}

func fsStatSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.statSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	info, err := os.Stat(path)
	if err != nil {
		return object.NewError(pos, "fs.statSync: %v", err)
	}
	return statObject(path, info)
}

func fsRenameSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	from, errObj := requiredString(pos, "fs.renameSync", args, 0, "from path")
	if errObj != nil {
		return errObj
	}
	to, errObj := requiredString(pos, "fs.renameSync", args, 1, "to path")
	if errObj != nil {
		return errObj
	}
	if err := os.Rename(from, to); err != nil {
		return object.NewError(pos, "fs.renameSync: %v", err)
	}
	return object.UNDEFINED
}

func fsUnlinkSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "fs.unlinkSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if err := os.Remove(path); err != nil {
		return object.NewError(pos, "fs.unlinkSync: %v", err)
	}
	return object.UNDEFINED
}

func statObject(path string, info os.FileInfo) *object.Hash {
	stat := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(stat, "path", &object.String{Value: path})
	setHashMember(stat, "name", &object.String{Value: filepath.Base(path)})
	setHashMember(stat, "size", &object.Number{Value: float64(info.Size())})
	setHashMember(stat, "mode", &object.String{Value: info.Mode().String()})
	setHashMember(stat, "mtimeMs", &object.Number{Value: float64(info.ModTime().UnixMilli())})
	setHashMember(stat, "isFileValue", object.NativeBool(info.Mode().IsRegular()))
	setHashMember(stat, "isDirectoryValue", object.NativeBool(info.IsDir()))
	setHashMember(stat, "isSymlinkValue", object.NativeBool(info.Mode()&os.ModeSymlink != 0))
	setHashMember(stat, "isFile", &object.Builtin{Name: "fs.stat.isFile", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.NativeBool(info.Mode().IsRegular())
	}})
	setHashMember(stat, "isDirectory", &object.Builtin{Name: "fs.stat.isDirectory", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.NativeBool(info.IsDir())
	}})
	setHashMember(stat, "isSymlink", &object.Builtin{Name: "fs.stat.isSymlink", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.NativeBool(info.Mode()&os.ModeSymlink != 0)
	}})
	return stat
}

func objectToText(obj object.Object) string {
	if s, ok := obj.(*object.String); ok {
		return s.Value
	}
	return obj.Inspect()
}

func requiredString(pos ast.Position, name string, args []object.Object, index int, label string) (string, *object.Error) {
	if len(args) <= index {
		return "", object.NewError(pos, "%s requires %s", name, label)
	}
	s, ok := args[index].(*object.String)
	if !ok {
		return "", object.NewError(pos, "%s: %s must be a string", name, label)
	}
	return s.Value, nil
}
