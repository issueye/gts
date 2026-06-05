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

// AnyMethod is a Go function exposed to GoScript that may return ordinary Go
// values. Return values are converted with ToValue.
type AnyMethod func(ctx CallContext, args []Value) (any, error)

// Module describes a host module exposed as @go/*, @host/*, or @plugin/*.
type Module struct {
	Name       string
	Version    string
	Values     map[string]any
	Methods    map[string]Method
	MethodsAny map[string]AnyMethod
	Docs       []string
}

// RegisterModule globally registers a GoScript native module.
//
// Native modules are global in the current process, matching the interpreter's
// existing module registry. Prefer stable names such as @go/image or
// @host/workspace.
func RegisterModule(mod Module) error {
	if err := validateModule(mod); err != nil {
		return err
	}
	module.RegisterNative(mod.Name, func(env *object.Environment) (object.Object, error) {
		return moduleExports(nil, mod, env)
	})
	if len(mod.Docs) > 0 {
		module.RegisterNativeAPIDoc(mod.Name, mod.Docs)
	}
	return nil
}

// RegisterModule registers a native module visible only to this Runtime.
func (r *Runtime) RegisterModule(mod Module) error {
	if r == nil {
		return fmt.Errorf("runtime is nil")
	}
	if err := validateModule(mod); err != nil {
		return err
	}
	r.modulesMu.Lock()
	if r.modules == nil {
		r.modules = make(map[string]Module)
	}
	r.modules[mod.Name] = mod
	r.modulesMu.Unlock()
	return nil
}

func validateModule(mod Module) error {
	if mod.Name == "" {
		return fmt.Errorf("module name is required")
	}
	if !module.IsNativeSpecifier(mod.Name) {
		return fmt.Errorf("module %q must use @std/, @go/, @host/, or @plugin/", mod.Name)
	}
	return nil
}

func moduleExports(rt *Runtime, mod Module, env *object.Environment) (object.Object, error) {
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
		exports.SetMember(&object.String{Value: methodName}, builtinMethod(rt, mod.Name, methodName, method))
	}
	for _, key := range sortedAnyMethodKeys(mod.MethodsAny) {
		methodName := key
		method := mod.MethodsAny[key]
		exports.SetMember(&object.String{Value: methodName}, builtinAnyMethod(rt, mod.Name, methodName, method))
	}
	return exports, nil
}

func builtinMethod(rt *Runtime, moduleName, methodName string, method Method) object.Object {
	return &object.Builtin{
		Name: moduleName + "." + methodName,
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			value, err := method(callContext(rt, env, pos, methodName), append([]Value{}, args...))
			if err != nil {
				return methodError(pos, moduleName, methodName, err)
			}
			if value == nil {
				return object.UNDEFINED
			}
			return value
		},
	}
}

func builtinAnyMethod(rt *Runtime, moduleName, methodName string, method AnyMethod) object.Object {
	return &object.Builtin{
		Name: moduleName + "." + methodName,
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			value, err := method(callContext(rt, env, pos, methodName), append([]Value{}, args...))
			if err != nil {
				return methodError(pos, moduleName, methodName, err)
			}
			if value == nil {
				return object.UNDEFINED
			}
			converted, err := ToValue(value)
			if err != nil {
				return object.NewError(pos, "%s.%s: %v", moduleName, methodName, err)
			}
			return converted
		},
	}
}

func callContext(rt *Runtime, env *object.Environment, pos ast.Position, methodName string) CallContext {
	return CallContext{
		Runtime: rt,
		Env:     env,
		Pos:     pos,
		Method:  methodName,
	}
}

func methodError(pos ast.Position, moduleName, methodName string, err error) object.Object {
	var runtimeErr RuntimeError
	if errors.As(err, &runtimeErr) {
		return object.NewNamedError(pos, runtimeErr.Name, runtimeErr.Message)
	}
	return object.NewError(pos, "%s.%s: %v", moduleName, methodName, err)
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

func sortedAnyMethodKeys(values map[string]AnyMethod) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
