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
