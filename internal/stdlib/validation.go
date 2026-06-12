package stdlib

import (
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/validation", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initValidationModule(exports)
		return exports, nil
	})
}

type validatorConfig struct {
	required   bool
	optional   bool
	nullable   bool
	validators []validatorFunc
}

type validatorFunc func(value object.Object, pos ast.Position) object.Object

func initValidationModule(exports *object.Hash) {
	setHashMember(exports, "string", createStringValidator())
	setHashMember(exports, "number", createNumberValidator())
	setHashMember(exports, "boolean", createBooleanValidator())
	setHashMember(exports, "array", createArrayValidator())
	setHashMember(exports, "object", createObjectValidator())
}

func createStringValidator() *object.Builtin {
	return &object.Builtin{
		Name: "validation.string",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			cfg := &validatorConfig{}
			inst := &object.Instance{Props: make(map[string]object.Object)}

			inst.Props["min"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("min requires length")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("min expects number")
				}
				minLen := int(n.Value)
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					if len([]rune(s.Value)) < minLen {
						return object.NewError(pos, "String length must be at least %d", minLen)
					}
					return nil
				}, nil
			})

			inst.Props["max"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("max requires length")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("max expects number")
				}
				maxLen := int(n.Value)
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					if len([]rune(s.Value)) > maxLen {
						return object.NewError(pos, "String length must be at most %d", maxLen)
					}
					return nil
				}, nil
			})

			inst.Props["email"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					_, err := mail.ParseAddress(s.Value)
					if err != nil {
						return object.NewError(pos, "Invalid email format")
					}
					return nil
				}, nil
			})

			inst.Props["url"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					_, err := url.ParseRequestURI(s.Value)
					if err != nil {
						return object.NewError(pos, "Invalid URL format")
					}
					return nil
				}, nil
			})

			inst.Props["uuid"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					if !uuidRe.MatchString(strings.ToLower(s.Value)) {
						return object.NewError(pos, "Invalid UUID format")
					}
					return nil
				}, nil
			})

			inst.Props["matches"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("matches requires pattern")
				}
				pattern := ""
				if re, ok := args[0].(*object.RegExp); ok {
					pattern = re.Source
				} else if s, ok := args[0].(*object.String); ok {
					pattern = s.Value
				} else {
					return nil, fmt.Errorf("matches expects regex or string")
				}
				re, err := regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("invalid regex: %v", err)
				}
				return func(value object.Object, pos ast.Position) object.Object {
					s, ok := value.(*object.String)
					if !ok {
						return object.NewError(pos, "Expected string, got %s", value.Type())
					}
					if !re.MatchString(s.Value) {
						return object.NewError(pos, "String does not match pattern")
					}
					return nil
				}, nil
			})

			inst.Props["required"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.required = true
				return nil, nil
			})

			inst.Props["optional"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.optional = true
				return nil, nil
			})

			inst.Props["validate"] = createValidateMethod(cfg)
			inst.Props["parse"] = createParseMethod(cfg)
			return inst
		},
	}
}

func createNumberValidator() *object.Builtin {
	return &object.Builtin{
		Name: "validation.number",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			cfg := &validatorConfig{}
			inst := &object.Instance{Props: make(map[string]object.Object)}

			inst.Props["min"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("min requires value")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("min expects number")
				}
				minVal := n.Value
				return func(value object.Object, pos ast.Position) object.Object {
					num, ok := value.(*object.Number)
					if !ok {
						return object.NewError(pos, "Expected number, got %s", value.Type())
					}
					if num.Value < minVal {
						return object.NewError(pos, "Number must be at least %.0f", minVal)
					}
					return nil
				}, nil
			})

			inst.Props["max"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("max requires value")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("max expects number")
				}
				maxVal := n.Value
				return func(value object.Object, pos ast.Position) object.Object {
					num, ok := value.(*object.Number)
					if !ok {
						return object.NewError(pos, "Expected number, got %s", value.Type())
					}
					if num.Value > maxVal {
						return object.NewError(pos, "Number must be at most %.0f", maxVal)
					}
					return nil
				}, nil
			})

			inst.Props["int"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				return func(value object.Object, pos ast.Position) object.Object {
					num, ok := value.(*object.Number)
					if !ok {
						return object.NewError(pos, "Expected number, got %s", value.Type())
					}
					if num.Value != math.Floor(num.Value) {
						return object.NewError(pos, "Number must be an integer")
					}
					return nil
				}, nil
			})

			inst.Props["positive"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				return func(value object.Object, pos ast.Position) object.Object {
					num, ok := value.(*object.Number)
					if !ok {
						return object.NewError(pos, "Expected number, got %s", value.Type())
					}
					if num.Value <= 0 {
						return object.NewError(pos, "Number must be positive")
					}
					return nil
				}, nil
			})

			inst.Props["required"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.required = true
				return nil, nil
			})

			inst.Props["optional"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.optional = true
				return nil, nil
			})

			inst.Props["validate"] = createValidateMethod(cfg)
			inst.Props["parse"] = createParseMethod(cfg)
			return inst
		},
	}
}

func createBooleanValidator() *object.Builtin {
	return &object.Builtin{
		Name: "validation.boolean",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			cfg := &validatorConfig{}
			inst := &object.Instance{Props: make(map[string]object.Object)}

			inst.Props["required"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.required = true
				return nil, nil
			})

			inst.Props["validate"] = createValidateMethod(cfg)
			inst.Props["parse"] = createParseMethod(cfg)
			return inst
		},
	}
}

func createArrayValidator() *object.Builtin {
	return &object.Builtin{
		Name: "validation.array",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			cfg := &validatorConfig{}
			inst := &object.Instance{Props: make(map[string]object.Object)}

			inst.Props["min"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("min requires length")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("min expects number")
				}
				minLen := int(n.Value)
				return func(value object.Object, pos ast.Position) object.Object {
					arr, ok := value.(*object.Array)
					if !ok {
						return object.NewError(pos, "Expected array, got %s", value.Type())
					}
					if len(arr.Elements) < minLen {
						return object.NewError(pos, "Array length must be at least %d", minLen)
					}
					return nil
				}, nil
			})

			inst.Props["max"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				if len(args) == 0 {
					return nil, fmt.Errorf("max requires length")
				}
				n, ok := args[0].(*object.Number)
				if !ok {
					return nil, fmt.Errorf("max expects number")
				}
				maxLen := int(n.Value)
				return func(value object.Object, pos ast.Position) object.Object {
					arr, ok := value.(*object.Array)
					if !ok {
						return object.NewError(pos, "Expected array, got %s", value.Type())
					}
					if len(arr.Elements) > maxLen {
						return object.NewError(pos, "Array length must be at most %d", maxLen)
					}
					return nil
				}, nil
			})

			inst.Props["required"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.required = true
				return nil, nil
			})

			inst.Props["validate"] = createValidateMethod(cfg)
			inst.Props["parse"] = createParseMethod(cfg)
			return inst
		},
	}
}

func createObjectValidator() *object.Builtin {
	return &object.Builtin{
		Name: "validation.object",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			cfg := &validatorConfig{}
			inst := &object.Instance{Props: make(map[string]object.Object)}

			inst.Props["required"] = createChainMethod(cfg, inst, func(args []object.Object, pos ast.Position) (validatorFunc, error) {
				cfg.required = true
				return nil, nil
			})

			inst.Props["validate"] = createValidateMethod(cfg)
			inst.Props["parse"] = createParseMethod(cfg)
			return inst
		},
	}
}

func createChainMethod(cfg *validatorConfig, inst *object.Instance, handler func([]object.Object, ast.Position) (validatorFunc, error)) *object.Builtin {
	return &object.Builtin{
		Name: "validation.chain",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			validator, err := handler(args, pos)
			if err != nil {
				return object.NewError(pos, "%v", err)
			}
			if validator != nil {
				cfg.validators = append(cfg.validators, validator)
			}
			return inst
		},
	}
}

func createValidateMethod(cfg *validatorConfig) *object.Builtin {
	return &object.Builtin{
		Name: "validation.validate",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "validate requires value")
			}
			value := args[0]

			if value.Type() == object.UNDEFINED_OBJ || value.Type() == object.NULL_OBJ {
				if cfg.required {
					return createValidationResult(false, "Value is required", value)
				}
				if cfg.optional || cfg.nullable {
					return createValidationResult(true, "", value)
				}
			}

			for _, validator := range cfg.validators {
				if err := validator(value, pos); err != nil {
					if errObj, ok := err.(*object.Error); ok {
						return createValidationResult(false, errObj.Message, value)
					}
					return createValidationResult(false, err.Inspect(), value)
				}
			}

			return createValidationResult(true, "", value)
		},
	}
}

func createParseMethod(cfg *validatorConfig) *object.Builtin {
	return &object.Builtin{
		Name: "validation.parse",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) == 0 {
				return object.NewError(pos, "parse requires value")
			}
			value := args[0]

			if value.Type() == object.UNDEFINED_OBJ || value.Type() == object.NULL_OBJ {
				if cfg.required {
					return object.NewError(pos, "Value is required")
				}
				if cfg.optional || cfg.nullable {
					return value
				}
			}

			for _, validator := range cfg.validators {
				if err := validator(value, pos); err != nil {
					return err
				}
			}

			return value
		},
	}
}

func createValidationResult(valid bool, message string, value object.Object) *object.Hash {
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "valid", &object.Boolean{Value: valid})
	setHashMember(result, "value", value)
	if !valid {
		setHashMember(result, "error", &object.String{Value: message})
	}
	return result
}
