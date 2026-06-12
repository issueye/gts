package stdlib

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/env", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initEnvModule(exports)
		return exports, nil
	})
}

func initEnvModule(exports *object.Hash) {
	setHashMember(exports, "load", &object.Builtin{Name: "env.load", Fn: envLoad})
	setHashMember(exports, "loadMultiple", &object.Builtin{Name: "env.loadMultiple", Fn: envLoadMultiple})
	setHashMember(exports, "get", &object.Builtin{Name: "env.get", Fn: envGet})
	setHashMember(exports, "getString", &object.Builtin{Name: "env.getString", Fn: envGet})
	setHashMember(exports, "getInt", &object.Builtin{Name: "env.getInt", Fn: envGetInt})
	setHashMember(exports, "getFloat", &object.Builtin{Name: "env.getFloat", Fn: envGetFloat})
	setHashMember(exports, "getNumber", &object.Builtin{Name: "env.getNumber", Fn: envGetFloat})
	setHashMember(exports, "getBool", &object.Builtin{Name: "env.getBool", Fn: envGetBool})
	setHashMember(exports, "getArray", &object.Builtin{Name: "env.getArray", Fn: envGetArray})
	setHashMember(exports, "getJson", &object.Builtin{Name: "env.getJson", Fn: envGetJson})
	setHashMember(exports, "has", &object.Builtin{Name: "env.has", Fn: envHas})
	setHashMember(exports, "require", &object.Builtin{Name: "env.require", Fn: envRequire})
	setHashMember(exports, "set", &object.Builtin{Name: "env.set", Fn: envSet})
	setHashMember(exports, "unset", &object.Builtin{Name: "env.unset", Fn: envUnset})
	setHashMember(exports, "toObject", &object.Builtin{Name: "env.toObject", Fn: envToObject})
	setHashMember(exports, "parse", &object.Builtin{Name: "env.parse", Fn: envParse})
}

func envLoad(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path := ".env"
	override := false

	if len(args) > 0 {
		if s, ok := args[0].(*object.String); ok {
			path = s.Value
		}
	}

	if len(args) > 1 {
		if hash, ok := args[1].(*object.Hash); ok {
			if pair, exists := hash.Pairs[object.HashKeyFor(&object.String{Value: "override"})]; exists {
				if b, ok := pair.Value.(*object.Boolean); ok {
					override = b.Value
				}
			}
		}
	}

	parsed, err := parseEnvFile(path)
	if err != nil {
		return object.NewError(pos, "env.load: %v", err)
	}

	for key, value := range parsed {
		if override || os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return object.UNDEFINED
}

func envLoadMultiple(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "loadMultiple requires array of paths")
	}

	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "loadMultiple expects array")
	}

	for _, elem := range arr.Elements {
		s, ok := elem.(*object.String)
		if !ok {
			continue
		}
		parsed, err := parseEnvFile(s.Value)
		if err != nil {
			continue
		}
		for key, value := range parsed {
			os.Setenv(key, value)
		}
	}

	return object.UNDEFINED
}

func envGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "get requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "get expects string key")
	}

	value := os.Getenv(key.Value)
	if value == "" && len(args) > 1 {
		if s, ok := args[1].(*object.String); ok {
			return s
		}
		return args[1]
	}

	if value == "" {
		return object.UNDEFINED
	}

	return &object.String{Value: value}
}

func envGetInt(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "getInt requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "getInt expects string key")
	}

	value := os.Getenv(key.Value)
	if value == "" {
		if len(args) > 1 {
			return args[1]
		}
		return object.UNDEFINED
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		if len(args) > 1 {
			return args[1]
		}
		return object.NewError(pos, "getInt: invalid integer %s", value)
	}

	return &object.Number{Value: float64(i)}
}

func envGetFloat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "getFloat requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "getFloat expects string key")
	}

	value := os.Getenv(key.Value)
	if value == "" {
		if len(args) > 1 {
			return args[1]
		}
		return object.UNDEFINED
	}

	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		if len(args) > 1 {
			return args[1]
		}
		return object.NewError(pos, "getFloat: invalid number %s", value)
	}

	return &object.Number{Value: f}
}

func envGetBool(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "getBool requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "getBool expects string key")
	}

	value := strings.ToLower(os.Getenv(key.Value))
	if value == "" {
		if len(args) > 1 {
			return args[1]
		}
		return object.UNDEFINED
	}

	switch value {
	case "true", "1", "yes", "on":
		return object.TRUE
	case "false", "0", "no", "off":
		return object.FALSE
	default:
		if len(args) > 1 {
			return args[1]
		}
		return object.NewError(pos, "getBool: invalid boolean %s", value)
	}
}

func envGetArray(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "getArray requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "getArray expects string key")
	}

	separator := ","
	if len(args) > 1 {
		if s, ok := args[1].(*object.String); ok {
			separator = s.Value
		}
	}

	value := os.Getenv(key.Value)
	if value == "" {
		return &object.Array{Elements: []object.Object{}}
	}

	parts := strings.Split(value, separator)
	elements := make([]object.Object, len(parts))
	for i, part := range parts {
		elements[i] = &object.String{Value: strings.TrimSpace(part)}
	}

	return &object.Array{Elements: elements}
}

func envGetJson(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "getJson requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "getJson expects string key")
	}

	value := os.Getenv(key.Value)
	if value == "" {
		return object.UNDEFINED
	}

	// 简单的 JSON 解析 - 使用内置的 JSON.parse
	// 实际应该通过 evaluator 调用，这里返回原始字符串
	return &object.String{Value: value}
}

func envHas(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "has requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "has expects string key")
	}

	_, exists := os.LookupEnv(key.Value)
	return &object.Boolean{Value: exists}
}

func envRequire(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "require expects array of keys")
	}

	arr, ok := args[0].(*object.Array)
	if !ok {
		return object.NewError(pos, "require expects array")
	}

	var missing []string
	for _, elem := range arr.Elements {
		key, ok := elem.(*object.String)
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key.Value); !exists {
			missing = append(missing, key.Value)
		}
	}

	if len(missing) > 0 {
		return object.NewError(pos, "Missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return object.UNDEFINED
}

func envSet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "set requires key and value")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "set expects string key")
	}

	value, ok := args[1].(*object.String)
	if !ok {
		value = &object.String{Value: args[1].Inspect()}
	}

	os.Setenv(key.Value, value.Value)
	return object.UNDEFINED
}

func envUnset(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "unset requires key")
	}

	key, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "unset expects string key")
	}

	os.Unsetenv(key.Value)
	return object.UNDEFINED
}

func envToObject(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	prefix := ""
	stripPrefix := false

	if len(args) > 0 {
		if hash, ok := args[0].(*object.Hash); ok {
			if pair, exists := hash.Pairs[object.HashKeyFor(&object.String{Value: "prefix"})]; exists {
				if s, ok := pair.Value.(*object.String); ok {
					prefix = s.Value
				}
			}
			if pair, exists := hash.Pairs[object.HashKeyFor(&object.String{Value: "stripPrefix"})]; exists {
				if b, ok := pair.Value.(*object.Boolean); ok {
					stripPrefix = b.Value
				}
			}
		}
	}

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}

		if stripPrefix && prefix != "" {
			key = strings.TrimPrefix(key, prefix)
		}

		setHashMember(result, key, &object.String{Value: value})
	}

	return result
}

func envParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "parse requires content")
	}

	content, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "parse expects string")
	}

	parsed := parseEnvContent(content.Value)
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for key, value := range parsed {
		setHashMember(result, key, &object.String{Value: value})
	}

	return result
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return parseEnvContent(content.String()), nil
}

func parseEnvContent(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentKey, currentValue string
	inMultiline := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			if !inMultiline {
				continue
			}
		}

		if inMultiline {
			// 多行值
			if strings.HasSuffix(line, "\"") {
				currentValue += "\n" + strings.TrimSuffix(line, "\"")
				result[currentKey] = currentValue
				inMultiline = false
				currentKey = ""
				currentValue = ""
			} else {
				currentValue += "\n" + line
			}
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 处理引号
		if strings.HasPrefix(value, "\"") {
			if strings.HasSuffix(value, "\"") && len(value) > 1 {
				// 单行引号
				value = strings.Trim(value, "\"")
			} else {
				// 多行引号开始
				inMultiline = true
				currentKey = key
				currentValue = strings.TrimPrefix(value, "\"")
				continue
			}
		} else if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = strings.Trim(value, "'")
		}

		// 变量展开
		value = expandVariables(value, result)

		result[key] = value
	}

	return result
}

func expandVariables(value string, env map[string]string) string {
	// 简单的 ${VAR} 展开
	for key, val := range env {
		placeholder := "${" + key + "}"
		value = strings.ReplaceAll(value, placeholder, val)
	}
	return value
}
