package sdk

import (
	"errors"
	"fmt"
	"sort"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

// CallContext contains runtime state for a host method invocation.
type CallContext struct {
	Runtime *Runtime
	Env     *object.Environment
	Pos     Position
	Method  string
}

// Method is a Go function exposed to GoScript.
type Method func(ctx CallContext, args []Value) (Value, error)

// Module describes a host module exposed as @go/*, @host/*, or @plugin/*.
type Module struct {
	Name    string
	Version string
	Values  map[string]any
	Methods map[string]Method
	Docs    []string
}

// RegisterModule globally registers a GoScript native module.
//
// Native modules are global in the current process, matching the interpreter's
// existing module registry. Prefer stable names such as @go/image or
// @host/workspace.
func RegisterModule(mod Module) error {
	if mod.Name == "" {
		return fmt.Errorf("module name is required")
	}
	if !module.IsNativeSpecifier(mod.Name) {
		return fmt.Errorf("module %q must use @std/, @go/, @host/, or @plugin/", mod.Name)
	}
	module.RegisterNative(mod.Name, func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for _, key := range sortedAnyKeys(mod.Values) {
			value, err := ToValue(mod.Values[key])
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", mod.Name, key, err)
			}
			exports.SetMember(&object.String{Value: key}, value)
		}
		for _, key := range sortedMethodKeys(mod.Methods) {
			methodName := key
			method := mod.Methods[key]
			exports.SetMember(&object.String{Value: methodName}, &object.Builtin{
				Name: mod.Name + "." + methodName,
				Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
					value, err := method(CallContext{
						Env:    env,
						Pos:    pos,
						Method: methodName,
					}, append([]Value{}, args...))
					if err != nil {
						var runtimeErr RuntimeError
						if errors.As(err, &runtimeErr) {
							return object.NewNamedError(pos, runtimeErr.Name, runtimeErr.Message)
						}
						return object.NewError(pos, "%s.%s: %v", mod.Name, methodName, err)
					}
					if value == nil {
						return object.UNDEFINED
					}
					return value
				},
			})
		}
		return exports, nil
	})
	if len(mod.Docs) > 0 {
		module.RegisterNativeAPIDoc(mod.Name, mod.Docs)
	}
	return nil
}

func sortedAnyKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMethodKeys(values map[string]Method) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
