package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/issueye/goscript/internal/object"
)

func TestRunVersion(t *testing.T) {
	if code := run([]string{"--version"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRunWithoutArgs(t *testing.T) {
	if code := run(nil); code != 2 {
		t.Fatalf("want exit code 2, got %d", code)
	}
}

func TestRunCheckTypesNotImplemented(t *testing.T) {
	if code := run([]string{"--check-types", "main.gs"}); code != 2 {
		t.Fatalf("want exit code 2, got %d", code)
	}
}

func TestRunScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "main.gs")
	if err := os.WriteFile(script, []byte(`println("ok");`), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{script}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRunMissingScript(t *testing.T) {
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runFileWithOptions(filepath.Join(t.TempDir(), "missing.gs"), runOptions{})
	if err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestRunSyntaxError(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "bad.gs")
	if err := os.WriteFile(script, []byte(`let x = ;`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runFileWithOptions(script, runOptions{})
	if err == nil {
		t.Fatal("expected syntax error")
	}
	if !strings.Contains(err.Error(), "no prefix parser") {
		t.Fatalf("expected parser error, got %v", err)
	}
}

func TestRunAutoMainOnlyWhenRequested(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(script, []byte(`function main() { return 42; }`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.(*object.Function); !ok {
		t.Fatalf("without autoMain want function declaration result, got %T", result)
	}

	result, err = r.evalFile(script, runOptions{autoMain: true})
	if err != nil {
		t.Fatal(err)
	}
	num, ok := result.(*object.Number)
	if !ok || num.Value != 42 {
		t.Fatalf("with autoMain want 42, got %T %v", result, result)
	}
}

func TestRunProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte("[project]\nentry = \"app.gs\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.gs"), []byte(`function main() { println("project"); }`), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"run"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRequireRelativePathAndCache(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.gs")
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(lib, []byte(`
let current = exports.count;
if (current === undefined) {
  exports.count = 1;
} else {
  exports.count = current + 1;
}
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`
let a = require("./lib.gs");
let b = require("./lib.gs");
a.count + b.count;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	num, ok := result.(*object.Number)
	if !ok || num.Value != 2 {
		t.Fatalf("want cached module result 2, got %T %v", result, result)
	}
}

func TestRequireNativeModule(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "native.gs")
	if err := os.WriteFile(app, []byte(`
let exec = require("@std/exec");
typeof exec.run;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "BUILTIN" {
		t.Fatalf("want native builtin type, got %T %v", result, result)
	}
}

func TestImportNamedExports(t *testing.T) {
	dir := t.TempDir()
	mathFile := filepath.Join(dir, "math.gs")
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(mathFile, []byte(`
export const one = 1;
export function add(a, b) { return a + b; }
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`
import { one, add } from "./math.gs";
add(one, 2);
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	num, ok := result.(*object.Number)
	if !ok || num.Value != 3 {
		t.Fatalf("want 3, got %T %v", result, result)
	}
}

func TestImportAliasAndDefaultExport(t *testing.T) {
	dir := t.TempDir()
	values := filepath.Join(dir, "values.gs")
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(values, []byte(`
export const value = 4;
export default 6;
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`
import total, { value as localValue } from "./values.gs";
total + localValue;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	num, ok := result.(*object.Number)
	if !ok || num.Value != 10 {
		t.Fatalf("want 10, got %T %v", result, result)
	}
}

func TestImportMissingExportFails(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.gs")
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(lib, []byte(`export const value = 1;`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`import { missing } from "./lib.gs";`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	_, err := r.evalFile(app, runOptions{})
	if err == nil {
		t.Fatal("expected missing export error")
	}
	if !strings.Contains(err.Error(), "no export missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportNamespaceAndExportSpecifiers(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "lib.gs")
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(lib, []byte(`
const value = 7;
function add(a, b) { return a + b; }
export { value, add as sum };
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app, []byte(`
import * as lib from "./lib.gs";
lib.sum(lib.value, 5);
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	num, ok := result.(*object.Number)
	if !ok || num.Value != 12 {
		t.Fatalf("want 12, got %T %v", result, result)
	}
}

func TestImportAgentAliasFromProjectRoot(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "examples")
	agentDir := filepath.Join(dir, "scripts", "agent", "core")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte("[project]\nentry = \"examples/app.gs\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "message.gs"), []byte(`export function label(x) { return "agent:" + x; }`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(appDir, "app.gs")
	if err := os.WriteFile(app, []byte(`
import { label } from "@agent/core/message";
label("ok");
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "agent:ok" {
		t.Fatalf("want agent:ok, got %T %v", result, result)
	}
}

func TestRunScriptTimeout(t *testing.T) {
	if os.Getenv("GOSCRIPT_TIMEOUT_HELPER") == "1" {
		os.Exit(run([]string{"--timeout", "20ms", os.Getenv("GOSCRIPT_TIMEOUT_SCRIPT")}))
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "loop.gs")
	if err := os.WriteFile(script, []byte(`while (true) {}`), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunScriptTimeout")
	cmd.Env = append(os.Environ(),
		"GOSCRIPT_TIMEOUT_HELPER=1",
		"GOSCRIPT_TIMEOUT_SCRIPT="+script,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected timeout failure, got success: %s", out)
	}
	if !strings.Contains(string(out), "timed out") {
		t.Fatalf("expected timeout message, got: %s", out)
	}
}

func TestStableExamples(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	examples := []string{
		filepath.Join(root, "examples", "01-basics.gs"),
		filepath.Join(root, "docs", "examples", "hello.gs"),
		filepath.Join(root, "docs", "examples", "fib.gs"),
		filepath.Join(root, "docs", "examples", "counter.gs"),
		filepath.Join(root, "docs", "examples", "modules.gs"),
	}

	for _, example := range examples {
		example := example
		t.Run(filepath.ToSlash(example[len(root)+1:]), func(t *testing.T) {
			r := newRunner(options{workers: 1, timeout: time.Second})
			if err := r.runFileWithOptions(example, runOptions{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
