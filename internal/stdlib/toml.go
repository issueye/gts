package stdlib

import (
	"os"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/pelletier/go-toml/v2"
)

func init() {
	module.RegisterNative("@std/toml", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTOMLModule(exports)
		return exports, nil
	})
}

func initTOMLModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "toml.parse", Fn: tomlParse})
	setHashMember(exports, "stringify", &object.Builtin{Name: "toml.stringify", Fn: tomlStringify})
	setHashMember(exports, "readFileSync", &object.Builtin{Name: "toml.readFileSync", Fn: tomlReadFileSync})
	setHashMember(exports, "writeFileSync", &object.Builtin{Name: "toml.writeFileSync", Fn: tomlWriteFileSync})
}

func tomlParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "toml.parse", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	var decoded map[string]interface{}
	if err := toml.Unmarshal([]byte(text), &decoded); err != nil {
		return object.NewError(pos, "toml.parse: %v", err)
	}
	return goValueToObject(decoded)
}

func tomlStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "toml.stringify requires a value")
	}
	encoded, err := toml.Marshal(objectToGoValue(args[0]))
	if err != nil {
		return object.NewError(pos, "toml.stringify: %v", err)
	}
	return &object.String{Value: string(encoded)}
}

func tomlReadFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "toml.readFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "toml.readFileSync: %v", err)
	}
	return tomlParse(env, pos, &object.String{Value: string(data)})
}

func tomlWriteFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "toml.writeFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "toml.writeFileSync requires value")
	}
	encoded := tomlStringify(env, pos, args[1])
	if err, ok := encoded.(*object.Error); ok {
		return err
	}
	text, ok := encoded.(*object.String)
	if !ok {
		return object.NewError(pos, "toml.writeFileSync: stringify did not return text")
	}
	if err := os.WriteFile(path, []byte(text.Value), 0644); err != nil {
		return object.NewError(pos, "toml.writeFileSync: %v", err)
	}
	return object.UNDEFINED
}
