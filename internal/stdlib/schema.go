package stdlib

import (
	"fmt"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/schema", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSchemaModule(exports)
		return exports, nil
	})
}

func initSchemaModule(exports *object.Hash) {
	setHashMember(exports, "validate", &object.Builtin{Name: "schema.validate", Fn: schemaValidate})
	setHashMember(exports, "assert", &object.Builtin{Name: "schema.assert", Fn: schemaAssert})
}

func schemaValidate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "schema.validate requires schema and value")
	}
	schema, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "schema.validate: schema must be an object")
	}
	errs := validateSchemaValue(schema, args[1], "$")
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "valid", object.NativeBool(len(errs) == 0))
	setHashMember(result, "errors", strSliceToArray(errs))
	return result
}

func schemaAssert(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "schema.assert requires schema and value")
	}
	schema, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "schema.assert: schema must be an object")
	}
	errs := validateSchemaValue(schema, args[1], "$")
	if len(errs) > 0 {
		return object.NewError(pos, "schema.assert: %s", errs[0])
	}
	return args[1]
}

func validateSchemaValue(schema *object.Hash, value object.Object, path string) []string {
	var errs []string
	if typeObj, ok := hashValue(schema, "type"); ok {
		if !schemaTypeMatches(typeObj, value) {
			errs = append(errs, fmt.Sprintf("%s expected %s, got %s", path, schemaTypeText(typeObj), schemaValueType(value)))
			return errs
		}
	}
	if enumObj, ok := hashValue(schema, "enum"); ok {
		enum, ok := enumObj.(*object.Array)
		if !ok {
			errs = append(errs, path+" schema enum must be an array")
		} else if !enumContains(enum, value) {
			errs = append(errs, fmt.Sprintf("%s must be one of %s", path, enum.Inspect()))
		}
	}

	switch v := value.(type) {
	case *object.Hash:
		errs = append(errs, validateObjectSchema(schema, v, path)...)
	case *object.Array:
		errs = append(errs, validateArraySchema(schema, v, path)...)
	case *object.String:
		errs = append(errs, validateStringSchema(schema, v, path)...)
	case *object.Number:
		errs = append(errs, validateNumberSchema(schema, v, path)...)
	}
	return errs
}

func validateObjectSchema(schema, value *object.Hash, path string) []string {
	var errs []string
	if requiredObj, ok := hashValue(schema, "required"); ok {
		required, ok := requiredObj.(*object.Array)
		if !ok {
			errs = append(errs, path+" schema required must be an array")
		} else {
			for _, item := range required.Elements {
				name, ok := item.(*object.String)
				if !ok {
					errs = append(errs, path+" schema required entries must be strings")
					continue
				}
				if _, exists := hashValue(value, name.Value); !exists {
					errs = append(errs, path+"."+name.Value+" is required")
				}
			}
		}
	}

	var properties *object.Hash
	if propsObj, ok := hashValue(schema, "properties"); ok {
		if props, ok := propsObj.(*object.Hash); ok {
			properties = props
			for _, pair := range props.Pairs {
				propSchema, ok := pair.Value.(*object.Hash)
				if !ok {
					errs = append(errs, path+"."+pair.Key.Inspect()+" schema must be an object")
					continue
				}
				if propValue, exists := hashValue(value, pair.Key.Inspect()); exists {
					errs = append(errs, validateSchemaValue(propSchema, propValue, path+"."+pair.Key.Inspect())...)
				}
			}
		} else {
			errs = append(errs, path+" schema properties must be an object")
		}
	}

	if additionalObj, ok := hashValue(schema, "additionalProperties"); ok {
		if b, ok := additionalObj.(*object.Boolean); ok && !b.Value && properties != nil {
			for _, pair := range value.Pairs {
				if _, exists := hashValue(properties, pair.Key.Inspect()); !exists {
					errs = append(errs, path+"."+pair.Key.Inspect()+" is not allowed")
				}
			}
		}
	}
	return errs
}

func validateArraySchema(schema *object.Hash, value *object.Array, path string) []string {
	var errs []string
	if minObj, ok := hashValue(schema, "minItems"); ok {
		if min, ok := minObj.(*object.Number); ok && len(value.Elements) < int(min.Value) {
			errs = append(errs, fmt.Sprintf("%s must contain at least %d items", path, int(min.Value)))
		}
	}
	if maxObj, ok := hashValue(schema, "maxItems"); ok {
		if max, ok := maxObj.(*object.Number); ok && len(value.Elements) > int(max.Value) {
			errs = append(errs, fmt.Sprintf("%s must contain at most %d items", path, int(max.Value)))
		}
	}
	if itemsObj, ok := hashValue(schema, "items"); ok {
		itemSchema, ok := itemsObj.(*object.Hash)
		if !ok {
			errs = append(errs, path+" schema items must be an object")
		} else {
			for i, item := range value.Elements {
				errs = append(errs, validateSchemaValue(itemSchema, item, fmt.Sprintf("%s[%d]", path, i))...)
			}
		}
	}
	return errs
}

func validateStringSchema(schema *object.Hash, value *object.String, path string) []string {
	var errs []string
	if minObj, ok := hashValue(schema, "minLength"); ok {
		if min, ok := minObj.(*object.Number); ok && len(value.Value) < int(min.Value) {
			errs = append(errs, fmt.Sprintf("%s length must be at least %d", path, int(min.Value)))
		}
	}
	if maxObj, ok := hashValue(schema, "maxLength"); ok {
		if max, ok := maxObj.(*object.Number); ok && len(value.Value) > int(max.Value) {
			errs = append(errs, fmt.Sprintf("%s length must be at most %d", path, int(max.Value)))
		}
	}
	return errs
}

func validateNumberSchema(schema *object.Hash, value *object.Number, path string) []string {
	var errs []string
	if minObj, ok := hashValue(schema, "minimum"); ok {
		if min, ok := minObj.(*object.Number); ok && value.Value < min.Value {
			errs = append(errs, fmt.Sprintf("%s must be >= %s", path, min.Inspect()))
		}
	}
	if maxObj, ok := hashValue(schema, "maximum"); ok {
		if max, ok := maxObj.(*object.Number); ok && value.Value > max.Value {
			errs = append(errs, fmt.Sprintf("%s must be <= %s", path, max.Inspect()))
		}
	}
	return errs
}

func schemaTypeMatches(typeObj object.Object, value object.Object) bool {
	if types, ok := typeObj.(*object.Array); ok {
		for _, t := range types.Elements {
			if schemaTypeMatches(t, value) {
				return true
			}
		}
		return false
	}
	typeStr, ok := typeObj.(*object.String)
	if !ok {
		return false
	}
	switch typeStr.Value {
	case "object":
		_, ok := value.(*object.Hash)
		return ok
	case "array":
		_, ok := value.(*object.Array)
		return ok
	case "string":
		_, ok := value.(*object.String)
		return ok
	case "number":
		_, ok := value.(*object.Number)
		return ok
	case "integer":
		n, ok := value.(*object.Number)
		return ok && n.Value == float64(int64(n.Value))
	case "boolean":
		_, ok := value.(*object.Boolean)
		return ok
	case "null":
		_, ok := value.(*object.Null)
		return ok
	default:
		return false
	}
}

func schemaTypeText(typeObj object.Object) string {
	switch t := typeObj.(type) {
	case *object.String:
		return t.Value
	default:
		return t.Inspect()
	}
}

func schemaValueType(value object.Object) string {
	switch value.(type) {
	case *object.Hash:
		return "object"
	case *object.Array:
		return "array"
	case *object.String:
		return "string"
	case *object.Number:
		return "number"
	case *object.Boolean:
		return "boolean"
	case *object.Null:
		return "null"
	case *object.Undefined:
		return "undefined"
	default:
		return string(value.Type())
	}
}

func enumContains(enum *object.Array, value object.Object) bool {
	for _, candidate := range enum.Elements {
		if objectsEqual(candidate, value) {
			return true
		}
	}
	return false
}

func objectsEqual(a, b object.Object) bool {
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
	case *object.Null:
		_, ok := b.(*object.Null)
		return ok
	default:
		return a.Inspect() == b.Inspect()
	}
}
