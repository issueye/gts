package stdlib

import (
	"os"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"gopkg.in/yaml.v3"
)

func init() {
	module.RegisterNative("@std/yaml", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initYAMLModule(exports)
		return exports, nil
	})
}

func initYAMLModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "yaml.parse", Fn: yamlParse})
	setHashMember(exports, "stringify", &object.Builtin{Name: "yaml.stringify", Fn: yamlStringify})
	setHashMember(exports, "readFileSync", &object.Builtin{Name: "yaml.readFileSync", Fn: yamlReadFileSync})
	setHashMember(exports, "writeFileSync", &object.Builtin{Name: "yaml.writeFileSync", Fn: yamlWriteFileSync})
}

func yamlParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	text, errObj := requiredString(pos, "yaml.parse", args, 0, "text")
	if errObj != nil {
		return errObj
	}
	var decoded interface{}
	if err := yaml.Unmarshal([]byte(text), &decoded); err != nil {
		return object.NewError(pos, "yaml.parse: %v", err)
	}
	return goValueToObject(normalizeYAMLValue(decoded))
}

func yamlStringify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "yaml.stringify requires a value")
	}
	encoded, err := yaml.Marshal(objectToGoValue(args[0]))
	if err != nil {
		return object.NewError(pos, "yaml.stringify: %v", err)
	}
	return &object.String{Value: string(encoded)}
}

func yamlReadFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "yaml.readFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return object.NewError(pos, "yaml.readFileSync: %v", err)
	}
	return yamlParse(env, pos, &object.String{Value: string(data)})
}

func yamlWriteFileSync(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "yaml.writeFileSync", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "yaml.writeFileSync requires value")
	}
	encoded := yamlStringify(env, pos, args[1])
	if err, ok := encoded.(*object.Error); ok {
		return err
	}
	text, ok := encoded.(*object.String)
	if !ok {
		return object.NewError(pos, "yaml.writeFileSync: stringify did not return text")
	}
	if err := os.WriteFile(path, []byte(text.Value), 0644); err != nil {
		return object.NewError(pos, "yaml.writeFileSync: %v", err)
	}
	return object.UNDEFINED
}

func normalizeYAMLValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, item := range v {
			out[key] = normalizeYAMLValue(item)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, item := range v {
			out[objectToMapKey(goValueToObject(key))] = normalizeYAMLValue(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = normalizeYAMLValue(item)
		}
		return out
	default:
		return v
	}
}
