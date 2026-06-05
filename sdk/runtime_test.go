package sdk

import (
	"strings"
	"testing"
	"time"
)

func TestRuntimeCanCallRegisteredGoModule(t *testing.T) {
	err := RegisterModule(Module{
		Name: "@go/sdk_test_demo",
		Values: map[string]any{
			"name": "demo",
		},
		Methods: map[string]Method{
			"add": func(ctx CallContext, args []Value) (Value, error) {
				left, ok := AsNumber(args[0])
				if !ok {
					return nil, Errorf("TypeError", "left must be a number")
				}
				right, ok := AsNumber(args[1])
				if !ok {
					return nil, Errorf("TypeError", "right must be a number")
				}
				return Number(left + right), nil
			},
		},
		Docs: []string{"add(left, right) -> number"},
	})
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	result, err := rt.RunSource(`
		const demo = require("@go/sdk_test_demo");
		demo.add(2, 3);
	`, "test.gs")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := AsNumber(result)
	if !ok {
		t.Fatalf("result should be number, got %T", result)
	}
	if got != 5 {
		t.Fatalf("result = %v, want 5", got)
	}
}

func TestRuntimeCanCallAnyMethod(t *testing.T) {
	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	err := rt.RegisterModule(Module{
		Name: "@go/sdk_test_any",
		MethodsAny: map[string]AnyMethod{
			"describe": func(ctx CallContext, args []Value) (any, error) {
				reader := NewArgs(ctx, args)
				name, err := reader.String(0, "name")
				if err != nil {
					return nil, err
				}
				count, err := reader.NumberDefault(1, "count", 1)
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"name":  name,
					"count": count,
				}, nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := rt.RunSource(`
		const demo = require("@go/sdk_test_any");
		demo.describe("item", 3).count;
	`, "any.gs")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := AsNumber(result)
	if got != 3 {
		t.Fatalf("result = %v, want 3", got)
	}
}

func TestArgsReportTypeErrors(t *testing.T) {
	reader := Args{
		Method: "demo.add",
		Values: []Value{String("nope")},
	}
	_, err := reader.Number(0, "left")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "TypeError") || !strings.Contains(err.Error(), "left must be a number") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRuntimeNativeModuleImportDefaultUsesGoNamespace(t *testing.T) {
	err := RegisterModule(Module{
		Name: "@go/sdk_test_import",
		Methods: map[string]Method{
			"value": func(ctx CallContext, args []Value) (Value, error) {
				return String("ok"), nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	result, err := rt.RunSource(`
		import demo from "@go/sdk_test_import";
		demo.value();
	`, "test.gs")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := AsString(result)
	if !ok || got != "ok" {
		t.Fatalf("result = %#v, want ok", FromValue(result))
	}
}

func TestHostMethodErrorBecomesRuntimeError(t *testing.T) {
	err := RegisterModule(Module{
		Name: "@go/sdk_test_error",
		Methods: map[string]Method{
			"fail": func(ctx CallContext, args []Value) (Value, error) {
				return nil, Errorf("HostError", "boom")
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	_, err = rt.RunSource(`
		const demo = require("@go/sdk_test_error");
		demo.fail();
	`, "test.gs")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HostError") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want HostError boom", err.Error())
	}
}

func TestRuntimeCanRunSourceMoreThanOnce(t *testing.T) {
	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	first, err := rt.RunSource(`1 + 1;`, "first.gs")
	if err != nil {
		t.Fatal(err)
	}
	second, err := rt.RunSource(`2 + 3;`, "second.gs")
	if err != nil {
		t.Fatal(err)
	}
	firstNumber, _ := AsNumber(first)
	secondNumber, _ := AsNumber(second)
	if firstNumber != 2 || secondNumber != 5 {
		t.Fatalf("results = %v, %v; want 2, 5", firstNumber, secondNumber)
	}
}

func TestRuntimeLocalModuleDoesNotLeakGlobally(t *testing.T) {
	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	err := rt.RegisterModule(Module{
		Name: "@go/sdk_test_local",
		Methods: map[string]Method{
			"value": func(ctx CallContext, args []Value) (Value, error) {
				if ctx.Runtime != rt {
					t.Fatalf("ctx.Runtime was not the registering runtime")
				}
				return Number(9), nil
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := rt.RunSource(`
		const local = require("@go/sdk_test_local");
		local.value();
	`, "local.gs")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := AsNumber(result)
	if got != 9 {
		t.Fatalf("result = %v, want 9", got)
	}

	other := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer other.Close()
	_, err = other.RunSource(`
		const local = require("@go/sdk_test_local");
		local.value();
	`, "other.gs")
	if err == nil {
		t.Fatal("expected local module to be unavailable in another runtime")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("error = %q, want not registered", err.Error())
	}
}

func TestRuntimeCallExport(t *testing.T) {
	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	result, err := rt.CallExport("testdata/exports.gs", "add", Number(4), Number(6))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := AsNumber(result)
	if got != 10 {
		t.Fatalf("result = %v, want 10", got)
	}
}

func TestRuntimeCallAsyncExport(t *testing.T) {
	rt := NewRuntime(Options{Timeout: 2 * time.Second, Workers: 1})
	defer rt.Close()

	result, err := rt.CallExport("testdata/exports.gs", "addAsync", Number(7), Number(8))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := AsNumber(result)
	if got != 15 {
		t.Fatalf("result = %v, want 15", got)
	}
}
