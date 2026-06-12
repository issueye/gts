package stdlib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/json", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initJsonModule(exports)
		return exports, nil
	})
}

func initJsonModule(exports *object.Hash) {
	setHashMember(exports, "parse5", &object.Builtin{Name: "json.parse5", Fn: jsonParse5})
	setHashMember(exports, "stringify5", &object.Builtin{Name: "json.stringify5", Fn: jsonStringify5})
	setHashMember(exports, "validate", &object.Builtin{Name: "json.validate", Fn: jsonValidate})
	setHashMember(exports, "get", &object.Builtin{Name: "json.get", Fn: jsonGet})
	setHashMember(exports, "set", &object.Builtin{Name: "json.set", Fn: jsonSet})
	setHashMember(exports, "has", &object.Builtin{Name: "json.has", Fn: jsonHas})
	setHashMember(exports, "remove", &object.Builtin{Name: "json.remove", Fn: jsonRemove})
	setHashMember(exports, "patch", &object.Builtin{Name: "json.patch", Fn: jsonPatch})
	setHashMember(exports, "diff", &object.Builtin{Name: "json.diff", Fn: jsonDiff})
}

// JSON5 解析
func jsonParse5(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "parse5 requires text")
	}

	text, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "parse5 expects string")
	}

	// 简化的 JSON5 预处理
	normalized := normalizeJSON5(text.Value)

	var data interface{}
	if err := json.Unmarshal([]byte(normalized), &data); err != nil {
		return object.NewError(pos, "parse5: %v", err)
	}

	return toObject(data)
}

// JSON5 序列化
func jsonStringify5(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 {
		return object.NewError(pos, "stringify5 requires value")
	}

	space := ""
	quote := "\""

	if len(args) > 1 {
		if opts, ok := args[1].(*object.Hash); ok {
			if val, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "space"})]; exists {
				if num, ok := val.Value.(*object.Number); ok {
					space = strings.Repeat(" ", int(num.Value))
				}
			}
			if val, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "quote"})]; exists {
				if s, ok := val.Value.(*object.String); ok && s.Value == "single" {
					quote = "'"
				}
			}
		}
	}

	native := toNative(args[0])
	var bytes []byte
	var err error

	if space != "" {
		bytes, err = json.MarshalIndent(native, "", space)
	} else {
		bytes, err = json.Marshal(native)
	}

	if err != nil {
		return object.NewError(pos, "stringify5: %v", err)
	}

	result := string(bytes)
	if quote == "'" {
		result = strings.ReplaceAll(result, "\"", "'")
	}

	return &object.String{Value: result}
}

// Schema 验证
func jsonValidate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "validate requires data and schema")
	}

	schema, ok := args[1].(*object.Hash)
	if !ok {
		return object.NewError(pos, "validate expects hash schema")
	}

	errors := validateValue(args[0], schema, "")

	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	if len(errors) == 0 {
		setHashMember(result, "valid", object.TRUE)
	} else {
		setHashMember(result, "valid", object.FALSE)
		errArray := &object.Array{Elements: make([]object.Object, len(errors))}
		for i, err := range errors {
			errArray.Elements[i] = &object.String{Value: err}
		}
		setHashMember(result, "errors", errArray)
	}

	return result
}

// JSON Pointer - get
func jsonGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "get requires doc and path")
	}

	path, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "get expects string path")
	}

	result := pointerGet(args[0], path.Value)
	if result == nil {
		return object.UNDEFINED
	}
	return result
}

// JSON Pointer - set
func jsonSet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 3 {
		return object.NewError(pos, "set requires doc, path, and value")
	}

	path, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "set expects string path")
	}

	pointerSet(args[0], path.Value, args[2])
	return object.UNDEFINED
}

// JSON Pointer - has
func jsonHas(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "has requires doc and path")
	}

	path, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "has expects string path")
	}

	result := pointerGet(args[0], path.Value)
	return &object.Boolean{Value: result != nil}
}

// JSON Pointer - remove
func jsonRemove(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "remove requires doc and path")
	}

	path, ok := args[1].(*object.String)
	if !ok {
		return object.NewError(pos, "remove expects string path")
	}

	pointerRemove(args[0], path.Value)
	return object.UNDEFINED
}

// JSON Patch
func jsonPatch(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "patch requires doc and operations")
	}

	ops, ok := args[1].(*object.Array)
	if !ok {
		return object.NewError(pos, "patch expects array of operations")
	}

	for _, opObj := range ops.Elements {
		op, ok := opObj.(*object.Hash)
		if !ok {
			continue
		}

		opType := getHashString(op, "op")
		path := getHashString(op, "path")

		switch opType {
		case "add", "replace":
			value, _ := hashValue(op, "value")
			pointerSet(args[0], path, value)

		case "remove":
			pointerRemove(args[0], path)

		case "move":
			from := getHashString(op, "from")
			value := pointerGet(args[0], from)
			if value != nil {
				pointerRemove(args[0], from)
				pointerSet(args[0], path, value)
			}

		case "copy":
			from := getHashString(op, "from")
			value := pointerGet(args[0], from)
			if value != nil {
				pointerSet(args[0], path, value)
			}

		case "test":
			value, _ := hashValue(op, "value")
			current := pointerGet(args[0], path)
			if !objectsEqual(current, value) {
				return object.NewError(pos, "test failed at %s", path)
			}
		}
	}

	return object.UNDEFINED
}

// JSON Diff
func jsonDiff(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "diff requires oldDoc and newDoc")
	}

	patches := &object.Array{Elements: []object.Object{}}
	diffObjects(args[0], args[1], "", patches)
	return patches
}

// === 辅助函数 ===

func normalizeJSON5(text string) string {
	// 移除单行注释
	text = regexp.MustCompile(`//[^\n]*`).ReplaceAllString(text, "")
	// 移除多行注释
	text = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(text, "")
	// 将单引号转为双引号（简化处理）
	text = strings.ReplaceAll(text, "'", "\"")
	// 移除尾逗号（简化处理）
	text = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(text, "$1")
	// 处理无引号键名
	text = regexp.MustCompile(`(\w+):`).ReplaceAllString(text, "\"$1\":")
	return text
}

func validateValue(value object.Object, schema *object.Hash, path string) []string {
	errors := []string{}

	typeVal := getHashString(schema, "type")
	if typeVal != "" {
		valid := false
		switch typeVal {
		case "string":
			_, valid = value.(*object.String)
		case "number":
			_, valid = value.(*object.Number)
		case "boolean":
			_, valid = value.(*object.Boolean)
		case "array":
			_, valid = value.(*object.Array)
		case "object":
			_, valid = value.(*object.Hash)
		case "null":
			valid = value == object.NULL
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("%s: expected type %s", path, typeVal))
		}
	}

	// 字符串验证
	if str, ok := value.(*object.String); ok {
		if minLen, exists := hashValue(schema, "minLength"); exists {
			if num, ok := minLen.(*object.Number); ok {
				if len(str.Value) < int(num.Value) {
					errors = append(errors, fmt.Sprintf("%s: string too short", path))
				}
			}
		}
		if maxLen, exists := hashValue(schema, "maxLength"); exists {
			if num, ok := maxLen.(*object.Number); ok {
				if len(str.Value) > int(num.Value) {
					errors = append(errors, fmt.Sprintf("%s: string too long", path))
				}
			}
		}
		if pattern, exists := hashValue(schema, "pattern"); exists {
			if patStr, ok := pattern.(*object.String); ok {
				if matched, _ := regexp.MatchString(patStr.Value, str.Value); !matched {
					errors = append(errors, fmt.Sprintf("%s: string does not match pattern", path))
				}
			}
		}
	}

	// 数字验证
	if num, ok := value.(*object.Number); ok {
		if min, exists := hashValue(schema, "minimum"); exists {
			if minNum, ok := min.(*object.Number); ok {
				if num.Value < minNum.Value {
					errors = append(errors, fmt.Sprintf("%s: number too small", path))
				}
			}
		}
		if max, exists := hashValue(schema, "maximum"); exists {
			if maxNum, ok := max.(*object.Number); ok {
				if num.Value > maxNum.Value {
					errors = append(errors, fmt.Sprintf("%s: number too large", path))
				}
			}
		}
	}

	// 数组验证
	if arr, ok := value.(*object.Array); ok {
		if minItems, exists := hashValue(schema, "minItems"); exists {
			if num, ok := minItems.(*object.Number); ok {
				if len(arr.Elements) < int(num.Value) {
					errors = append(errors, fmt.Sprintf("%s: array too short", path))
				}
			}
		}
		if maxItems, exists := hashValue(schema, "maxItems"); exists {
			if num, ok := maxItems.(*object.Number); ok {
				if len(arr.Elements) > int(num.Value) {
					errors = append(errors, fmt.Sprintf("%s: array too long", path))
				}
			}
		}
	}

	// 对象验证
	if hash, ok := value.(*object.Hash); ok {
		if required, exists := hashValue(schema, "required"); exists {
			if reqArr, ok := required.(*object.Array); ok {
				for _, req := range reqArr.Elements {
					if reqStr, ok := req.(*object.String); ok {
						if _, exists := hash.Pairs[object.HashKeyFor(reqStr)]; !exists {
							errors = append(errors, fmt.Sprintf("%s: missing required field %s", path, reqStr.Value))
						}
					}
				}
			}
		}

		if props, exists := hashValue(schema, "properties"); exists {
			if propsHash, ok := props.(*object.Hash); ok {
				for k, pair := range propsHash.Pairs {
					if propSchema, ok := pair.Value.(*object.Hash); ok {
						key := pair.Key.Inspect()
						if val, exists := hash.Pairs[k]; exists {
							subPath := path + "/" + key
							errors = append(errors, validateValue(val.Value, propSchema, subPath)...)
						}
					}
				}
			}
		}
	}

	return errors
}

func pointerGet(doc object.Object, path string) object.Object {
	if path == "" {
		return doc
	}

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	current := doc

	for _, part := range parts {
		part = unescapePointer(part)

		if hash, ok := current.(*object.Hash); ok {
			key := &object.String{Value: part}
			if val, exists := hash.Pairs[object.HashKeyFor(key)]; exists {
				current = val.Value
			} else {
				return nil
			}
		} else if arr, ok := current.(*object.Array); ok {
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(arr.Elements) {
				return nil
			}
			current = arr.Elements[idx]
		} else {
			return nil
		}
	}

	return current
}

func pointerSet(doc object.Object, path string, value object.Object) {
	if path == "" {
		return
	}

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 {
		return
	}

	current := doc
	for i := 0; i < len(parts)-1; i++ {
		part := unescapePointer(parts[i])

		if hash, ok := current.(*object.Hash); ok {
			key := &object.String{Value: part}
			if val, exists := hash.Pairs[object.HashKeyFor(key)]; exists {
				current = val.Value
			} else {
				newHash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
				setHashMember(hash, part, newHash)
				current = newHash
			}
		}
	}

	lastPart := unescapePointer(parts[len(parts)-1])
	if hash, ok := current.(*object.Hash); ok {
		setHashMember(hash, lastPart, value)
	}
}

func pointerRemove(doc object.Object, path string) {
	if path == "" {
		return
	}

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 {
		return
	}

	current := doc
	for i := 0; i < len(parts)-1; i++ {
		part := unescapePointer(parts[i])

		if hash, ok := current.(*object.Hash); ok {
			key := &object.String{Value: part}
			if val, exists := hash.Pairs[object.HashKeyFor(key)]; exists {
				current = val.Value
			} else {
				return
			}
		}
	}

	lastPart := unescapePointer(parts[len(parts)-1])
	if hash, ok := current.(*object.Hash); ok {
		delete(hash.Pairs, object.HashKeyFor(&object.String{Value: lastPart}))
	}
}

func diffObjects(old, new object.Object, path string, patches *object.Array) {
	if objectsEqual(old, new) {
		return
	}

	oldHash, oldIsHash := old.(*object.Hash)
	newHash, newIsHash := new.(*object.Hash)

	if !oldIsHash || !newIsHash {
		// 不同类型或基本类型，使用 replace
		patch := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(patch, "op", &object.String{Value: "replace"})
		setHashMember(patch, "path", &object.String{Value: path})
		setHashMember(patch, "value", new)
		patches.Elements = append(patches.Elements, patch)
		return
	}

	// 检查新增和修改
	for k, newPair := range newHash.Pairs {
		key := newPair.Key.Inspect()
		subPath := path + "/" + key

		if oldPair, exists := oldHash.Pairs[k]; exists {
			diffObjects(oldPair.Value, newPair.Value, subPath, patches)
		} else {
			patch := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			setHashMember(patch, "op", &object.String{Value: "add"})
			setHashMember(patch, "path", &object.String{Value: subPath})
			setHashMember(patch, "value", newPair.Value)
			patches.Elements = append(patches.Elements, patch)
		}
	}

	// 检查删除
	for k, oldPair := range oldHash.Pairs {
		if _, exists := newHash.Pairs[k]; !exists {
			key := oldPair.Key.Inspect()
			subPath := path + "/" + key
			patch := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			setHashMember(patch, "op", &object.String{Value: "remove"})
			setHashMember(patch, "path", &object.String{Value: subPath})
			patches.Elements = append(patches.Elements, patch)
		}
	}
}

func objectsEqual(a, b object.Object) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch av := a.(type) {
	case *object.String:
		bv, ok := b.(*object.String)
		return ok && av.Value == bv.Value
	case *object.Number:
		bv, ok := b.(*object.Number)
		return ok && av.Value == bv.Value
	case *object.Boolean:
		bv, ok := b.(*object.Boolean)
		return ok && av.Value == bv.Value
	case *object.Hash:
		bv, ok := b.(*object.Hash)
		if !ok || len(av.Pairs) != len(bv.Pairs) {
			return false
		}
		for k, av := range av.Pairs {
			bv, exists := bv.Pairs[k]
			if !exists || !objectsEqual(av.Value, bv.Value) {
				return false
			}
		}
		return true
	case *object.Array:
		bv, ok := b.(*object.Array)
		if !ok || len(av.Elements) != len(bv.Elements) {
			return false
		}
		for i := range av.Elements {
			if !objectsEqual(av.Elements[i], bv.Elements[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func getHashString(hash *object.Hash, key string) string {
	if val, exists := hashValue(hash, key); exists {
		if str, ok := val.(*object.String); ok {
			return str.Value
		}
	}
	return ""
}

func escapePointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

func unescapePointer(s string) string {
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

func toObject(v interface{}) object.Object {
	if v == nil {
		return object.NULL
	}

	switch val := v.(type) {
	case string:
		return &object.String{Value: val}
	case float64:
		return &object.Number{Value: val}
	case bool:
		return &object.Boolean{Value: val}
	case []interface{}:
		elements := make([]object.Object, len(val))
		for i, item := range val {
			elements[i] = toObject(item)
		}
		return &object.Array{Elements: elements}
	case map[string]interface{}:
		hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for k, v := range val {
			setHashMember(hash, k, toObject(v))
		}
		return hash
	default:
		return object.NULL
	}
}

func toNative(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.String:
		return v.Value
	case *object.Number:
		return v.Value
	case *object.Boolean:
		return v.Value
	case *object.Array:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			arr[i] = toNative(elem)
		}
		return arr
	case *object.Hash:
		m := make(map[string]interface{})
		for _, pair := range v.Pairs {
			key := pair.Key.Inspect()
			m[key] = toNative(pair.Value)
		}
		return m
	case *object.Null:
		return nil
	default:
		return nil
	}
}

func deepClone(obj object.Object) object.Object {
	switch v := obj.(type) {
	case *object.Hash:
		newHash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for k, pair := range v.Pairs {
			newHash.Pairs[k] = object.HashPair{
				Key:   pair.Key,
				Value: deepClone(pair.Value),
			}
		}
		return newHash
	case *object.Array:
		newArr := &object.Array{Elements: make([]object.Object, len(v.Elements))}
		for i, elem := range v.Elements {
			newArr.Elements[i] = deepClone(elem)
		}
		return newArr
	default:
		return obj
	}
}
