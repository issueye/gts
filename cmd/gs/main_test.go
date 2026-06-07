package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/packagefile"
)

func TestRunVersion(t *testing.T) {
	if code := run([]string{"--version"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRunAPIDocNativeModule(t *testing.T) {
	stdout, stderr, code := captureRunOutput(t, []string{"--api_doc", "@std/web"})
	if code != 0 {
		t.Fatalf("want exit code 0, got %d stderr=%q", code, stderr)
	}
	for _, want := range []string{
		"@std/web",
		"createApp() -> app",
		"static(root) -> middleware",
		"proxy(targetOrOptions) -> middleware",
		"app.get(path, handler, ...handlers)",
		"app.listen(port?) -> server",
		"res.json(value)",
		"创建 Web 应用实例",
		"解析 JSON 请求体并写入 req.body",
		"设置响应状态码并返回 res",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("api doc missing %q in:\n%s", want, stdout)
		}
	}
}

func TestRunAPIDocAllNativeModulesHaveChineseDocs(t *testing.T) {
	for _, path := range module.ListNative() {
		docs, ok := module.GetNativeAPIDoc(path)
		if !ok {
			t.Fatalf("%s is missing native API docs", path)
		}
		if len(docs) == 0 {
			t.Fatalf("%s has empty native API docs", path)
		}
	}
}

func TestRunAPIDocNativeModuleParametersAndChineseDescriptions(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{
			path: "@std/fs",
			want: []string{"readFileSync(path)", "writeFileSync(path, data)", "读取文件文本"},
		},
		{
			path: "@std/crypto",
			want: []string{"hmac(algorithm, key, value)", "计算 HMAC 摘要"},
		},
		{
			path: "@std/db",
			want: []string{"open(driver, dsn) -> conn", "conn.query(query, params?)", "打开数据库连接"},
		},
	}
	for _, tt := range tests {
		stdout, stderr, code := captureRunOutput(t, []string{"--api_doc", tt.path})
		if code != 0 {
			t.Fatalf("%s: want exit code 0, got %d stderr=%q", tt.path, code, stderr)
		}
		for _, want := range tt.want {
			if !strings.Contains(stdout, want) {
				t.Fatalf("%s api doc missing %q in:\n%s", tt.path, want, stdout)
			}
		}
	}
}

func TestRunAPIDocListsNativeModules(t *testing.T) {
	stdout, stderr, code := captureRunOutput(t, []string{"--api_doc", "all"})
	if code != 0 {
		t.Fatalf("want exit code 0, got %d stderr=%q", code, stderr)
	}
	for _, want := range []string{"Native modules:", "@std/web", "@std/fs"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("api doc list missing %q in:\n%s", want, stdout)
		}
	}
}

func TestRunAPIDocUnknownNativeModule(t *testing.T) {
	stdout, stderr, code := captureRunOutput(t, []string{"--api_doc", "@std/missing"})
	if code != 1 {
		t.Fatalf("want exit code 1, got %d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "native module @std/missing is not registered") {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
}

func TestRunWithoutArgs(t *testing.T) {
	oldInput := cliInput
	cliInput = strings.NewReader(".exit\n")
	defer func() { cliInput = oldInput }()

	stdout, stderr, code := captureRunOutput(t, nil)
	if code != 0 {
		t.Fatalf("want exit code 0, got %d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "GoScript "+version) || !strings.Contains(stdout, "gs> ") {
		t.Fatalf("unexpected repl output: %q", stdout)
	}
}

func TestREPLPersistsBindings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runREPL(replConfig{
		in:     strings.NewReader("let value = 40;\nvalue + 2\n.exit\n"),
		out:    &stdout,
		errOut: &stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "42\n") {
		t.Fatalf("expected persisted expression result, got %q", stdout.String())
	}
}

func TestREPLMultilineFunction(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runREPL(replConfig{
		in: strings.NewReader(`function add(a, b) {
  return a + b;
}
add(2, 3)
.exit
`),
		out:    &stdout,
		errOut: &stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "5\n") {
		t.Fatalf("expected multiline function result, got %q", stdout.String())
	}
}

func TestREPLErrorDoesNotExitSession(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runREPL(replConfig{
		in:     strings.NewReader("missingName\n1 + 1\n.exit\n"),
		out:    &stdout,
		errOut: &stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "ReferenceError") {
		t.Fatalf("expected runtime error, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "2\n") {
		t.Fatalf("expected session to continue, got %q", stdout.String())
	}
}

func TestREPLLoadCommand(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "load.gs")
	writeTestFile(t, script, "let loaded = 7;\n")

	var stdout, stderr bytes.Buffer
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runREPL(replConfig{
		in:     strings.NewReader(".load " + script + "\nloaded * 6\n.exit\n"),
		out:    &stdout,
		errOut: &stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stderr.String() != "" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "42\n") {
		t.Fatalf("expected loaded binding result, got %q", stdout.String())
	}
}

func TestREPLUnknownDotCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runREPL(replConfig{
		in:     strings.NewReader(".loadbad\n.exit\n"),
		out:    &stdout,
		errOut: &stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "unknown command: .loadbad") {
		t.Fatalf("expected unknown command error, got %q", stderr.String())
	}
}

func TestRunCheckTypes(t *testing.T) {
	dir := t.TempDir()
	okScript := filepath.Join(dir, "ok.gs")
	writeTestFile(t, okScript, `let value: number = 42;`)
	if code := run([]string{"--check-types", okScript}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}

	badScript := filepath.Join(dir, "bad.gs")
	writeTestFile(t, badScript, `let value: number = "bad";`)
	if code := run([]string{"--check-types", badScript}); code != 1 {
		t.Fatalf("want exit code 1, got %d", code)
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

func TestRunScriptPassesArgsToProcessArgv(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "args.gs")
	out := filepath.Join(dir, "argv.txt")
	writeTestFile(t, script, strings.ReplaceAll(`
let fs = require("@std/fs");
let process = require("@std/process");
fs.writeFileSync("__OUT__", process.argv.slice(1).join("|"));
`, "__OUT__", strings.ReplaceAll(out, `\`, `\\`)))

	if code := run([]string{script, "alpha", "--flag", "0"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := script + "|alpha|--flag|0"
	if string(data) != want {
		t.Fatalf("want %q, got %q", want, data)
	}
}

func TestRunScriptPassesArgsAfterDoubleDash(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "args.gs")
	out := filepath.Join(dir, "argv.txt")
	writeTestFile(t, script, strings.ReplaceAll(`
let fs = require("@std/fs");
let process = require("@std/process");
fs.writeFileSync("__OUT__", process.argv.slice(2).join("|"));
`, "__OUT__", strings.ReplaceAll(out, `\`, `\\`)))

	if code := run([]string{script, "--", "--flag", "value"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "--flag|value" {
		t.Fatalf("want forwarded args, got %q", data)
	}
}

func TestRunScriptCommandRunsExternalScriptWithMain(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "tool.gs")
	out := filepath.Join(dir, "out.txt")
	writeTestFile(t, script, strings.ReplaceAll(`
let fs = require("@std/fs");
let process = require("@std/process");
function main() {
  fs.writeFileSync("__OUT__", process.argv.slice(2).join("|"));
}
`, "__OUT__", strings.ReplaceAll(out, `\`, `\\`)))

	if code := run([]string{"run-script", script, "--", "--flag", "value"}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "--flag|value" {
		t.Fatalf("want script args, got %q", data)
	}
}

func TestEmbeddedRunScriptCommandRunsExternalScript(t *testing.T) {
	if os.Getenv("GOSCRIPT_EMBEDDED_RUN_SCRIPT_HELPER") == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Exit(run(os.Args[i+1:]))
			}
		}
		os.Exit(2)
	}

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "project.toml"), `[project]
entry = "main.gs"
`)
	writeTestFile(t, filepath.Join(root, "main.gs"), `function main() { println("embedded app"); }`)
	pkgPath := filepath.Join(root, "app.gspkg")
	if err := packagefile.PackDirectory(root, pkgPath); err != nil {
		t.Fatal(err)
	}
	stub, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(root, "app.exe")
	if err := packagefile.AppendPackageToExecutable(stub, pkgPath, exePath); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(root, "external.txt")
	script := filepath.Join(root, "external.gs")
	writeTestFile(t, script, strings.ReplaceAll(`
let fs = require("@std/fs");
let process = require("@std/process");
function main() {
  fs.writeFileSync("__OUT__", process.argv.slice(2).join("|"));
}
`, "__OUT__", strings.ReplaceAll(outFile, `\`, `\\`)))

	cmd := exec.Command(exePath, "-test.run=TestEmbeddedRunScriptCommandRunsExternalScript", "--", "run-script", script, "--", "generated", "tool")
	cmd.Env = append(os.Environ(), "GOSCRIPT_EMBEDDED_RUN_SCRIPT_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("embedded run-script failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "generated|tool" {
		t.Fatalf("want external script args, got %q", data)
	}
}

func TestSplitEmbeddedAppArgsOnlyForEmbeddedExecutable(t *testing.T) {
	old := hasAppendedPackage
	defer func() { hasAppendedPackage = old }()

	hasAppendedPackage = func() bool { return false }
	if split := splitEmbeddedAppArgs([]string{"--", "--self-test"}); split != nil {
		t.Fatalf("non-embedded executable should not split args: %#v", split)
	}

	hasAppendedPackage = func() bool { return true }
	split := splitEmbeddedAppArgs([]string{"--timeout", "1s", "--", "--self-test", "--flag"})
	if split == nil {
		t.Fatal("embedded executable should split args after --")
	}
	if split.separator != 2 {
		t.Fatalf("want separator 2, got %d", split.separator)
	}
	if strings.Join(split.app, "|") != "--self-test|--flag" {
		t.Fatalf("unexpected app args: %#v", split.app)
	}
}

func TestSplitDirectEmbeddedAppArgs(t *testing.T) {
	if split := splitDirectEmbeddedAppArgs([]string{"--timeout", "1s", "--tui"}); split == nil {
		t.Fatal("embedded app flag should split after known cli flags")
	} else if split.separator != 2 || strings.Join(split.app, "|") != "--tui" {
		t.Fatalf("unexpected split: %#v", split)
	}

	if split := splitDirectEmbeddedAppArgs([]string{"--timeout=1s", "--tui", "extra"}); split == nil {
		t.Fatal("embedded app flag should split after known cli flag with value")
	} else if split.separator != 1 || strings.Join(split.app, "|") != "--tui|extra" {
		t.Fatalf("unexpected split with value: %#v", split)
	}

	if split := splitDirectEmbeddedAppArgs([]string{"--version"}); split != nil {
		t.Fatalf("known cli flag should not split: %#v", split)
	}

	if split := splitDirectEmbeddedAppArgs([]string{"--", "--tui"}); split != nil {
		t.Fatalf("double dash is handled by splitEmbeddedAppArgs: %#v", split)
	}
}

func TestEmbeddedExecutablePassesArgsAfterDoubleDash(t *testing.T) {
	if os.Getenv("GOSCRIPT_EMBEDDED_ARGS_HELPER") == "1" {
		os.Exit(run(os.Args[1:]))
	}

	root := t.TempDir()
	outFile := filepath.Join(root, "argv.txt")
	appSource := strings.ReplaceAll(`
let fs = require("@std/fs");
let process = require("@std/process");
function main() {
  fs.writeFileSync("__OUT__", process.argv.join("|"));
}
`, "__OUT__", strings.ReplaceAll(outFile, `\`, `\\`))
	writeTestFile(t, filepath.Join(root, "project.toml"), `[project]
entry = "main.gs"
`)
	writeTestFile(t, filepath.Join(root, "main.gs"), appSource)
	pkgPath := filepath.Join(root, "app.gspkg")
	if err := packagefile.PackDirectory(root, pkgPath); err != nil {
		t.Fatal(err)
	}
	stub, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(root, "app.exe")
	if err := packagefile.AppendPackageToExecutable(stub, pkgPath, exePath); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(exePath, "--", "--self-test", "--flag", "value")
	cmd.Env = append(os.Environ(), "GOSCRIPT_EMBEDDED_ARGS_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("embedded executable failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	wantSuffix := "main.gs|--self-test|--flag|value"
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("want argv suffix %q, got %q", wantSuffix, got)
	}
}

func TestRunInlineCode(t *testing.T) {
	if code := run([]string{`console.log("ok")`}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRunInlineCodeWithSplitArgs(t *testing.T) {
	if code := run([]string{`let`, `value`, `=`, `"ok";`, `console.log(value)`}); code != 0 {
		t.Fatalf("want exit code 0, got %d", code)
	}
}

func TestRunScriptReusesResetVirtualMachine(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.gs")
	second := filepath.Join(dir, "second.gs")
	writeTestFile(t, first, `let leaked = 1;`)
	writeTestFile(t, second, `leaked;`)

	if code := run([]string{first}); code != 0 {
		t.Fatalf("first run want exit code 0, got %d", code)
	}
	r := newRunner(options{workers: 1, timeout: time.Second})
	err := r.runFileWithOptions(second, runOptions{})
	if err == nil || !strings.Contains(err.Error(), "ReferenceError") {
		t.Fatalf("second run should not see prior VM bindings, got %v", err)
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

func TestRunProjectStartsConfiguredGTPPlugin(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), fmt.Sprintf(`[project]
entry = "app.gs"

[plugins.scheduler]
command = "go"
args = ["run", "."]
cwd = %q
modules = ["@plugin/scheduler"]
capabilities = ["call", "event"]
`, filepath.Join(root, "plugins", "scheduler")))
	writeTestFile(t, filepath.Join(dir, "app.gs"), `
let fs = require("@std/fs");
function main() {
  const scheduler = require("@plugin/scheduler");
  let tasks = scheduler.list();
  fs.writeFileSync("plugin-result.txt", String(tasks.length));
}
`)
	r := newRunner(options{workers: 1, timeout: 5 * time.Second})
	if err := r.runProject(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "plugin-result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "0" {
		t.Fatalf("plugin result = %q, want 0", data)
	}
}

func TestRunProjectScriptListensToGTPPluginEvent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), fmt.Sprintf(`[project]
entry = "app.gs"

[plugins.scheduler]
command = "go"
args = ["run", "."]
cwd = %q
modules = ["@plugin/scheduler"]
capabilities = ["call", "event"]
`, filepath.Join(root, "plugins", "scheduler")))
	writeTestFile(t, filepath.Join(dir, "app.gs"), `
let fs = require("@std/fs");
function main() {
  const scheduler = require("@plugin/scheduler");
  scheduler.once("trigger", function(event) {
    let task = event.data;
    fs.writeFileSync("plugin-event-result.txt", event.event + ":" + task.name + ":" + task.payload.message);
  });
  scheduler.schedule({
    name: "script-listener-test",
    delayMs: 25,
    payload: { message: "handled by script" }
  });
}
`)
	r := newRunner(options{workers: 1, timeout: 5 * time.Second})
	if err := r.runProject(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "plugin-event-result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "trigger:script-listener-test:handled by script" {
		t.Fatalf("plugin event result = %q", data)
	}
}

func TestInitCommandCreatesProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hello-app")
	if err := initCommand([]string{dir}); err != nil {
		t.Fatal(err)
	}
	project, err := os.ReadFile(filepath.Join(dir, "project.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(project), `name = "hello-app"`) || !strings.Contains(string(project), `entry = "main.gs"`) {
		t.Fatalf("unexpected project.toml:\n%s", project)
	}
	main, err := os.ReadFile(filepath.Join(dir, "main.gs"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(main), `function main()`) {
		t.Fatalf("unexpected main.gs:\n%s", main)
	}
	r := newRunner(options{workers: 1, timeout: time.Second})
	if err := r.runProject(dir); err != nil {
		t.Fatal(err)
	}
}

func TestInitCommandDoesNotOverwriteExistingFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), `[project]
name = "existing"
`)
	if err := initCommand([]string{dir}); err == nil {
		t.Fatal("expected init to reject existing project.toml")
	}
	data, err := os.ReadFile(filepath.Join(dir, "project.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `name = "existing"`) {
		t.Fatalf("init overwrote project.toml:\n%s", data)
	}
}

func TestRunProjectUsesProjectWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), "[project]\nentry = \"app.gs\"\n")
	writeTestFile(t, filepath.Join(dir, "workspace", "task.txt"), "cwd ok")
	writeTestFile(t, filepath.Join(dir, "app.gs"), `
let fs = require("@std/fs");
let path = require("@std/path");
let process = require("@std/process");
let text = fs.readFileSync(path.join(process.cwd(), "workspace", "task.txt"));
if (text !== "cwd ok") {
  throw new Error("bad project cwd: " + process.cwd());
}
`)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	runWd := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(runWd); err != nil {
		t.Fatal(err)
	}
	runWd, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	if err := r.runProject(dir); err != nil {
		t.Fatal(err)
	}
	gotWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if gotWd != runWd {
		t.Fatalf("runProject did not restore working directory: want %s, got %s", runWd, gotWd)
	}
}

func TestRunProjectPassesArgsToProcessArgv(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), "[project]\nentry = \"app.gs\"\n")
	writeTestFile(t, filepath.Join(dir, "app.gs"), `
let fs = require("@std/fs");
let process = require("@std/process");
function main() {
  fs.writeFileSync("argv.txt", process.argv.slice(2).join("|"));
}
`)

	r := newRunner(options{workers: 1, timeout: time.Second})
	if err := r.runProject(dir, "alpha", "beta"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "alpha|beta" {
		t.Fatalf("want project args, got %q", data)
	}
}

func TestPackCommandCreatesPackageFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "project.toml"), `[package]
name = "tools"
main = "src/index.gs"
`)
	writeTestFile(t, filepath.Join(root, "src", "index.gs"), `export const value = 42;`)
	out := filepath.Join(t.TempDir(), "tools.gspkg")

	if err := packCommand([]string{root, out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}

func TestDistCommandCreatesEmbeddedExecutable(t *testing.T) {
	root := t.TempDir()
	outFile := filepath.Join(root, "argv.txt")
	writeTestFile(t, filepath.Join(root, "project.toml"), `[project]
entry = "main.gs"
`)
	writeTestFile(t, filepath.Join(root, "main.gs"), strings.ReplaceAll(`
let fs = require("@std/fs");
let lib = require("./lib");
let process = require("@std/process");
function main() {
  if (lib.value !== "embedded") {
    throw new Error("bad embedded value");
  }
  fs.writeFileSync("__OUT__", process.argv.slice(2).join("|"));
}
`, "__OUT__", strings.ReplaceAll(outFile, `\`, `\\`)))
	writeTestFile(t, filepath.Join(root, "lib.gs"), `exports.value = "embedded";`)
	out := filepath.Join(t.TempDir(), "app.exe")

	if err := distCommand([]string{root, out}); err != nil {
		t.Fatal(err)
	}
	data, err := packagefile.ReadAppendedPackage(out)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := packagefile.OpenBytes(out, data)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Close()
	r := newRunner(options{workers: 1, timeout: time.Second})
	if err := r.runPackageEntryFromExecutable(pkg, out, "main.gs", "packed", "arg"); err != nil {
		t.Fatal(err)
	}
	argvData, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(argvData) != "packed|arg" {
		t.Fatalf("want embedded args, got %q", argvData)
	}
}

func TestBundleCommandWritesOutput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "main.gs"), `let lib = require("./lib");
exports.value = lib.value;
`)
	writeTestFile(t, filepath.Join(root, "lib.gs"), `exports.value = 42;
`)
	out := filepath.Join(root, "dist", "app.bundle.gs")

	if err := bundleCommand([]string{filepath.Join(root, "main.gs"), out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "GoScript bundle") || strings.Contains(string(data), `require("./lib")`) {
		t.Fatalf("unexpected bundle output:\n%s", data)
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

func TestImportNativeStdlibModule(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "native_import.gs")
	if err := os.WriteFile(app, []byte(`
import path, { join, sep as separator } from "@std/path";
let joined = join("alpha", "beta");
let defaultJoined = path.join("gamma", "delta");
let ok = joined === path.join("alpha", "beta") && defaultJoined === join("gamma", "delta") && separator === path.sep;
if (ok) {
  "ok";
} else {
  "bad";
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok" {
		t.Fatalf("want ok, got %T %v", result, result)
	}
}

func TestStdPathModule(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "path.gs")
	if err := os.WriteFile(app, []byte(`
let path = require("@std/path");
let absKind = "rel";
if (path.isAbs(path.resolve("alpha"))) {
  absKind = "abs";
}
let slashKind = "bad";
if (path.toSlash(path.fromSlash("alpha/beta/file.txt")) === "alpha/beta/file.txt") {
  slashKind = "slash";
}
let matchKind = "no";
if (path.matches("*.txt", "file.txt")) {
  matchKind = "match";
}
let parsed = path.parse(path.join("alpha", "beta", "file.txt"));
let parseKind = "bad";
if (parsed.name === "file" && parsed.ext === ".txt" && path.basename(path.format(parsed)) === "file.txt") {
  parseKind = "parse";
}
let listKind = "bad";
if (path.splitList("alpha" + path.delimiter + "beta").length === 2) {
  listKind = "list";
}
path.basename(path.join("alpha", "beta", "file.txt")) + ":" + path.extname("file.txt") + ":" + absKind + ":" + slashKind + ":" + matchKind + ":" + parseKind + ":" + listKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "file.txt:.txt:abs:slash:match:parse:list" {
		t.Fatalf("want file.txt:.txt:abs:slash:match:parse:list, got %T %v", result, result)
	}
}

func TestStdFSModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fs.gs")
	work := filepath.Join(dir, "work")
	appSource := strings.ReplaceAll(`
let fs = require("@std/fs");
let path = require("@std/path");
let root = "__WORK__";
fs.mkdirSync(root, { recursive: true });
let file = path.join(root, "note.txt");
fs.writeFileSync(file, "hello");
let names = fs.readdirSync(root);
let stat = fs.statSync(file);
let text = fs.readFileSync(file);
fs.unlinkSync(file);
let fileKind = "not-file";
if (stat.isFile()) {
  fileKind = "file";
}
let existsKind = "exists";
if (!fs.existsSync(file)) {
  existsKind = "missing";
}
text + ":" + names[0] + ":" + fileKind + ":" + existsKind;
`, "__WORK__", strings.ReplaceAll(work, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "hello:note.txt:file:missing" {
		t.Fatalf("want fs smoke result, got %T %v", result, result)
	}
}

func TestStdFSEnhancedModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fs_enhanced.gs")
	work := filepath.Join(dir, "work")
	appSource := strings.ReplaceAll(`
let fs = require("@std/fs");
let path = require("@std/path");
let root = "__WORK__";
fs.mkdirSync(path.join(root, "nested"), { recursive: true });
let file = path.join(root, "nested", "note.txt");
fs.writeFileAtomicSync(file, "one");
fs.appendFileSync(file, "\ntwo");
fs.appendTextSync(file, "\nthree");
let tmpDir = fs.mkdtempSync(path.join(root, "tmp-"));
let copy = path.join(root, "copy.txt");
fs.copyFileSync(file, copy);
let text = fs.readTextSync(file);
let copyText = fs.readTextSync(copy);
let realKind = "bad";
if (path.isAbs(fs.realpathSync(copy))) {
  realKind = "real";
}
let lstat = fs.lstatSync(copy);
let lstatKind = "bad";
if (lstat.isFile() && !lstat.isSymlink()) {
  lstatKind = "lstat";
}
let entries = fs.walkSync(root, { includeDirs: false });
let countKind = "bad";
if (entries.length === 2) {
  countKind = "two-files";
}
let typed = fs.readdirSync(root, { withFileTypes: true });
let typedKind = "bad";
for (let i = 0; i < typed.length; i = i + 1) {
  if (typed[i].name === "nested" && typed[i].isDirectory()) {
    typedKind = "dirent";
  }
}
let globbed = fs.globSync(path.join(root, "nested", "*.txt"));
let globKind = "bad";
if (globbed.length === 1 && path.basename(globbed[0]) === "note.txt") {
  globKind = "glob";
}
fs.rmSync(path.join(root, "nested"), { recursive: true, force: true });
fs.rmSync(path.join(root, "missing"), { recursive: true, force: true });
let rmKind = "bad";
if (!fs.existsSync(file)) {
  rmKind = "removed";
}
let tmpKind = "bad";
if (fs.existsSync(tmpDir)) {
  tmpKind = "tmp";
}
text + ":" + copyText + ":" + countKind + ":" + typedKind + ":" + globKind + ":" + realKind + ":" + lstatKind + ":" + tmpKind + ":" + rmKind;
`, "__WORK__", strings.ReplaceAll(work, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "one\ntwo\nthree:one\ntwo\nthree:two-files:dirent:glob:real:lstat:tmp:removed"
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
	}
}

func TestStdProcessAndOSModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "env.gs")
	appSource := strings.ReplaceAll(`
let process = require("@std/process");
let os = require("@std/os");
let before = process.cwd();
process.setenv("GOSCRIPT_AGENT_TEST", "ok");
let value = process.getenv("GOSCRIPT_AGENT_TEST");
let fallback = process.getenv("GOSCRIPT_AGENT_MISSING", "missing");
let envNow = process.envObject();
let envKind = "missing";
if (envNow.GOSCRIPT_AGENT_TEST === "ok") {
  envKind = "env";
}
let argvKind = "empty";
if (process.argv0 !== "") {
  argvKind = "argv0";
}
let execKind = "empty";
if (process.execPath() !== "") {
  execKind = "exec";
}
let runtimeKind = "bad";
let mark = process.hrtime();
let diff = process.hrtime(mark);
if (process.uptime() >= 0 && diff.length === 2 && process.version !== "") {
  runtimeKind = "runtime";
}
let platformKind = "empty";
if (os.platform !== "") {
  platformKind = "set";
}
let tmpKind = "empty";
if (os.tmpdir() !== "") {
  tmpKind = "set";
}
let eolKind = "bad";
if (os.eol === "\n" || os.eol === "\r\n") {
  eolKind = "eol";
}
let cpuKind = "bad";
if (os.cpus() > 0) {
  cpuKind = "cpus";
}
let osKind = "bad";
let user = os.userInfo();
if (os.type() !== "" && os.release() !== "" && user.homedir !== "") {
  osKind = "os";
}
process.chdir("__DIR__");
let after = process.cwd();
process.chdir(before);
process.unsetenv("GOSCRIPT_AGENT_TEST");
value + ":" + fallback + ":" + envKind + ":" + argvKind + ":" + execKind + ":" + runtimeKind + ":" + platformKind + ":" + tmpKind + ":" + eolKind + ":" + cpuKind + ":" + osKind + ":" + after;
`, "__DIR__", strings.ReplaceAll(dir, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "ok:missing:env:argv0:exec:runtime:set:set:eol:cpus:os:" + dir
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
	}
}

func TestStdCryptoAndSchemaModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "schema.gs")
	if err := os.WriteFile(script, []byte(`
let crypto = require("@std/crypto");
let schema = require("@std/schema");

let spec = {
  type: "object",
  required: ["command"],
  additionalProperties: false,
  properties: {
    command: { type: "string", minLength: 1 },
    limit: { type: "integer", minimum: 1, maximum: 100 },
    mode: { enum: ["read", "write"] }
  }
};

let ok = schema.validate(spec, { command: "ls", limit: 3, mode: "read" });
let bad = schema.validate(spec, { limit: 0, extra: true });
let okKind = "bad";
if (ok.valid) {
  okKind = "ok";
}
let badKind = "ok";
if (!bad.valid) {
  badKind = "bad";
}
let bytes = crypto.randomBytes(4);
let bytesKind = "wrong";
if (bytes.length === 4) {
  bytesKind = "bytes4";
}
let uuid = crypto.randomUUID();
let uuidKind = "uuid-bad";
if (uuid.length === 36 && uuid.charAt(14) === "4") {
  uuidKind = "uuid";
}
let cryptoKind = "crypto-bad";
let pbk = crypto.pbkdf2("password", "salt", 1, 32, "sha256");
let pbkBuffer = crypto.pbkdf2("password", "salt", 1, 32, "sha256", { asBuffer: true });
if (
  crypto.sha1("abc") === "a9993e364706816aba3e25717850c26c9cd0d89d" &&
  crypto.sha256("abc") === "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" &&
  crypto.sha512("abc") === "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f" &&
  crypto.hmac("sha256", "key", "The quick brown fox jumps over the lazy dog") === "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8" &&
  pbk === "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b" &&
  pbkBuffer.length === 32 &&
  crypto.timingSafeEqual("same", "same") &&
  !crypto.timingSafeEqual("same", "diff")
) {
  cryptoKind = "crypto";
}
okKind + ":" + badKind + ":" + bytesKind + ":" + uuidKind + ":" + cryptoKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "ok:bad:bytes4:uuid:crypto"
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
	}
}

func TestStdURLBufferEventsTimersModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "native_more.gs")
	if err := os.WriteFile(script, []byte(`
let url = require("@std/url");
let buffer = require("@std/buffer");
let events = require("@std/events");
let timers = require("@std/timers");

let parsed = url.parse("https://example.com:8443/a/b?x=1#top");
let resolved = url.resolve("https://example.com/a/b", "../c?y=2");
let u = url.URL("/next?q=1", "https://example.com/base/");
u.searchParams.set("q", "2");
u.searchParams.append("tag", "go");
let params = url.URLSearchParams("a=1&a=2");
params.append("b", "3");
let urlKind = "url-bad";
if (parsed.protocol === "https:" && parsed.hostname === "example.com" && parsed.port === "8443" && resolved === "https://example.com/c?y=2" && u.search === "?q=2&tag=go" && params.get("a") === "1" && params.has("b")) {
  urlKind = "url";
}

let b = buffer.from("6869", "hex");
let joined = buffer.concat([b, buffer.from("!")]);
let filled = buffer.alloc(3, 65);
let bufferKind = "buffer-bad";
if (buffer.isBuffer(b) && b.length === 2 && b.toString() === "hi" && joined.toString("base64") === "aGkh" && filled.toString() === "AAA" && joined.slice(1).toString() === "i!") {
  bufferKind = "buffer";
}

let emitter = events.EventEmitter();
let seen = "";
function onValue(v) { seen = seen + "on" + v; }
emitter.on("value", onValue);
emitter.once("value", function(v) { seen = seen + "once" + v; });
let firstEmit = emitter.emit("value", "1");
let secondEmit = emitter.emit("value", "2");
emitter.off("value", onValue);
let thirdEmit = emitter.emit("value", "3");
let eventsKind = "events-bad";
if (firstEmit && secondEmit && !thirdEmit && seen === "on1once1on2" && emitter.listenerCount("value") === 0) {
  eventsKind = "events";
}

let timerKind = "timers-bad";
let ticked = false;
timers.queueMicrotask(function() { ticked = true; });
timers.sleep(10);
if (typeof timers.setTimeout === "BUILTIN" && ticked) {
  timerKind = "timers";
}

urlKind + ":" + bufferKind + ":" + eventsKind + ":" + timerKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "url:buffer:events:timers"
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
	}
}

func TestStdEncodingModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "encoding.gs")
	csvFile := filepath.Join(dir, "users.csv")
	appSource := strings.ReplaceAll(`
let base64 = require("@std/encoding/base64");
let hex = require("@std/encoding/hex");
let csv = require("@std/encoding/csv");
let buffer = require("@std/buffer");

let b64Kind = "base64-bad";
let encoded = base64.encode("hello?");
let decoded = base64.decode(encoded);
let urlEncoded = base64.encodeURL("hello?");
let urlDecoded = base64.decodeURL(urlEncoded);
let decodedBuffer = base64.decode(encoded, { asBuffer: true });
if (encoded === "aGVsbG8/" && decoded === "hello?" && urlDecoded === "hello?" && buffer.isBuffer(decodedBuffer) && decodedBuffer.toString() === "hello?") {
  b64Kind = "base64";
}

let hexKind = "hex-bad";
let hexed = hex.encode("hi");
let unhexed = hex.decode(hexed);
let hexBuffer = hex.decode(hexed, { asBuffer: true });
if (hexed === "6869" && unhexed === "hi" && buffer.isBuffer(hexBuffer) && hexBuffer.toString() === "hi") {
  hexKind = "hex";
}

let parsed = csv.parse("name,age\nAda,36\nLinus,55\n");
let arrays = csv.parse("Ada;36\nLinus;55\n", { header: false, comma: ";" });
let out = csv.stringify(parsed);
csv.writeFileSync("__CSV__", parsed);
let fromFile = csv.readFileSync("__CSV__");

let csvKind = "csv-bad";
if (parsed.length === 2 && parsed[0].name === "Ada" && parsed[1].age === "55" && arrays[0][1] === "36" && out.includes("Ada") && fromFile[1].name === "Linus") {
  csvKind = "csv";
}

b64Kind + ":" + hexKind + ":" + csvKind;
`, "__CSV__", strings.ReplaceAll(csvFile, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "base64:hex:csv" {
		t.Fatalf("want base64:hex:csv, got %T %v", result, result)
	}
}

func TestStdHashAndGzipModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "hash_gzip.gs")
	sourceFile := filepath.Join(dir, "source.txt")
	gzipFile := filepath.Join(dir, "source.txt.gz")
	outFile := filepath.Join(dir, "out.txt")
	appSource := strings.NewReplacer(
		"__SOURCE__", strings.ReplaceAll(sourceFile, `\`, `\\`),
		"__GZIP__", strings.ReplaceAll(gzipFile, `\`, `\\`),
		"__OUT__", strings.ReplaceAll(outFile, `\`, `\\`),
	).Replace(`
let fs = require("@std/fs");
let buffer = require("@std/buffer");
let hash = require("@std/hash");
let gzip = require("@std/compress/gzip");

let hashKind = "hash-bad";
if (hash.crc32("hello") === "3610a686" && hash.adler32("hello") === "062c0215" && hash.fnv1a(buffer.from("hello")) === "a430d84680aabd0b" && hash.crc32Number("hello") === 907060870) {
  hashKind = "hash";
}

let compressed = gzip.compress("hello gzip");
let decompressed = gzip.decompress(compressed);
let decompressedBuffer = gzip.decompress(compressed, { asBuffer: true });
fs.writeFileSync("__SOURCE__", "file gzip");
gzip.compressFileSync("__SOURCE__", "__GZIP__");
gzip.decompressFileSync("__GZIP__", "__OUT__");

let gzipKind = "gzip-bad";
if (buffer.isBuffer(compressed) && decompressed === "hello gzip" && buffer.isBuffer(decompressedBuffer) && decompressedBuffer.toString() === "hello gzip" && fs.readFileSync("__OUT__") === "file gzip") {
  gzipKind = "gzip";
}

hashKind + ":" + gzipKind;
`)
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "hash:gzip" {
		t.Fatalf("want hash:gzip, got %T %v", result, result)
	}
}

func TestStdArchiveZipModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "zip.gs")
	root := filepath.Join(dir, "root")
	archive := filepath.Join(dir, "app.zip")
	extract := filepath.Join(dir, "extract")
	appSource := strings.NewReplacer(
		"__ROOT__", strings.ReplaceAll(root, `\`, `\\`),
		"__ARCHIVE__", strings.ReplaceAll(archive, `\`, `\\`),
		"__EXTRACT__", strings.ReplaceAll(extract, `\`, `\\`),
	).Replace(`
let fs = require("@std/fs");
let path = require("@std/path");
let zip = require("@std/archive/zip");

fs.mkdirSync(path.join("__ROOT__", "src"), { recursive: true });
fs.writeFileSync(path.join("__ROOT__", "README.md"), "hello zip");
fs.writeFileSync(path.join("__ROOT__", "src", "main.gs"), "println(\"zip\")");

zip.create([
  { path: path.join("__ROOT__", "README.md"), name: "README.md" },
  { path: path.join("__ROOT__", "src"), name: "src" },
], "__ARCHIVE__");

let entries = zip.list("__ARCHIVE__");
zip.extract("__ARCHIVE__", "__EXTRACT__");

let hasReadme = false;
let hasMain = false;
for (let entry of entries) {
  if (entry.name === "README.md" && entry.size === 9) {
    hasReadme = true;
  }
  if (entry.name === "src/main.gs") {
    hasMain = true;
  }
}

let readme = fs.readFileSync(path.join("__EXTRACT__", "README.md"));
let main = fs.readFileSync(path.join("__EXTRACT__", "src", "main.gs"));
let zipKind = "zip-bad";
if (hasReadme && hasMain && readme === "hello zip" && main.includes("zip")) {
  zipKind = "zip";
}
zipKind;
`)
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "zip" {
		t.Fatalf("want zip, got %T %v", result, result)
	}
}

func TestStdNetIPModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "net_ip.gs")
	if err := os.WriteFile(script, []byte(`
let ip = require("@std/net/ip");

let parsed = ip.parseIP("127.0.0.1");
let cidr = ip.parseCIDR("127.0.0.0/8");
let split = ip.splitHostPort("127.0.0.1:8080");
let joined = ip.joinHostPort("::1", "443");
let hosts = ip.lookupHost("localhost");

let kind = "netip-bad";
if (
  parsed.value === "127.0.0.1" &&
  parsed.is4 &&
  parsed.isLoopback &&
  cidr.addr === "127.0.0.0" &&
  cidr.bits === 8 &&
  ip.contains("127.0.0.0/8", "127.0.0.1") &&
  !ip.contains("127.0.0.0/8", "192.168.0.1") &&
  split.host === "127.0.0.1" &&
  split.port === "8080" &&
  joined === "[::1]:443" &&
  hosts.length > 0 &&
  ip.parseIP("not-ip") === undefined
) {
  kind = "netip";
}
kind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "netip" {
		t.Fatalf("want netip, got %T %v", result, result)
	}
}

func TestStdMimeAndMailModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "mime_mail.gs")
	if err := os.WriteFile(script, []byte(`
let mime = require("@std/mime");
let mail = require("@std/mail");

let media = mime.parseMediaType("Text/HTML; Charset=UTF-8");
let formatted = mime.formatMediaType("text/plain", { charset: "utf-8" });
let ext = mime.extensionByType("text/html; charset=utf-8");

let mimeKind = "mime-bad";
if (mime.typeByExtension(".html").includes("text/html") && media.type === "text/html" && media.params.charset === "UTF-8" && formatted === "text/plain; charset=utf-8" && (ext === ".html" || ext === ".htm")) {
  mimeKind = "mime";
}

let addr = mail.parseAddress("Ada Lovelace <ada@example.com>");
let list = mail.parseAddressList("Ada <ada@example.com>, linus@example.com");
let msg = mail.parseMessage("From: Ada <ada@example.com>\r\nTo: Linus <linus@example.com>\r\nSubject: Hello\r\n\r\nBody text");
let formattedAddr = mail.formatAddress({ name: "Ada Lovelace", address: "ada@example.com" });
let formattedList = mail.formatAddressList([{ name: "Ada", address: "ada@example.com" }, "linus@example.com"]);
let parsedDate = mail.parseDate("Mon, 02 Jan 2006 15:04:05 +0000");
let formattedDate = mail.formatDate(parsedDate);
let subject = mail.getHeader(msg.headers, "subject");

let mailKind = "mail-bad";
if (addr.name === "Ada Lovelace" && addr.address === "ada@example.com" && list.length === 2 && list[1].address === "linus@example.com" && msg.headers.Subject[0] === "Hello" && msg.body === "Body text" && formattedAddr === "\"Ada Lovelace\" <ada@example.com>" && formattedList.includes("linus@example.com") && formattedDate.includes("2006") && subject === "Hello") {
  mailKind = "mail";
}

mimeKind + ":" + mailKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "mime:mail" {
		t.Fatalf("want mime:mail, got %T %v", result, result)
	}
}

func TestStdTemplateTimeAndSignalModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "template_time_signal.gs")
	tplFile := filepath.Join(dir, "hello.tmpl")
	appSource := strings.NewReplacer(
		"__TPL__", strings.ReplaceAll(tplFile, `\`, `\\`),
	).Replace(`
let fs = require("@std/fs");
let template = require("@std/template");
let time = require("@std/time");
let signal = require("@std/signal");

fs.writeFileSync("__TPL__", "File {{.Name}}");

let rendered = template.render("Hello {{upper .Name}} {{join .Items \",\"}}", {
  Name: "ada",
  Items: ["x", "y"],
});
let html = template.renderHTML("<b>{{.Value}}</b>", { Value: "<tag>" });
let fileRendered = template.renderFileSync("__TPL__", { Name: "Grace" });
let escaped = template.escapeHTML("<x>");

let parsed = time.parse("2020-01-02T03:04:05Z");
let formatted = time.format(parsed, time.RFC3339, "UTC");
let later = time.add(parsed, "2s");
let duration = time.parseDuration("1.5s");
let fromMs = time.unixMs(0);

let waited = signal.wait({ signals: ["SIGINT"], timeoutMs: 1 });
let watcher = signal.notify(["SIGINT"]);
let watcherWaited = watcher.wait(1);
watcher.stop();
let supported = signal.supported();

let kind = "std-bad";
if (
  rendered === "Hello ADA x,y" &&
  html === "<b>&lt;tag&gt;</b>" &&
  fileRendered === "File Grace" &&
  escaped === "&lt;x&gt;" &&
  formatted === "2020-01-02T03:04:05Z" &&
  later.toISOString() === "2020-01-02T03:04:07.000Z" &&
  duration.milliseconds === 1500 &&
  fromMs.toISOString() === "1970-01-01T00:00:00.000Z" &&
  waited === null &&
  watcherWaited === null &&
  supported.includes("SIGINT")
) {
  kind = "std";
}
kind;
`)
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "std" {
		t.Fatalf("want std, got %T %v", result, result)
	}
}

func TestStdLogModuleWritesToFile(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "log.gs")
	logFile := filepath.Join(dir, "app.log")
	jsonFile := filepath.Join(dir, "app.jsonl")
	appSource := strings.NewReplacer(
		"__LOG__", strings.ReplaceAll(logFile, `\`, `\\`),
		"__JSON__", strings.ReplaceAll(jsonFile, `\`, `\\`),
	).Replace(`
let fs = require("@std/fs");
let log = require("@std/log");

let logger = log.createFileLogger("__LOG__", { append: false, timestamp: false, level: "info" });
logger.debug("hidden");
logger.info("started", 1);
logger.warn("careful");
logger.error("failed");
logger.close();

let again = log.createFileLogger("__LOG__", { append: true, timestamp: false });
again.info("again");
again.close();

let json = log.createFileLogger("__JSON__", { append: false, timestamp: false, json: true });
json.info("structured");
json.close();

let text = fs.readFileSync("__LOG__");
let jsonText = fs.readFileSync("__JSON__");

let kind = "log-bad";
if (!text.includes("hidden") && text.includes("[INFO] started 1") && text.includes("[WARN] careful") && text.includes("[ERROR] failed") && text.includes("[INFO] again") && jsonText.includes("\"level\":\"info\"") && jsonText.includes("\"message\":\"structured\"")) {
  kind = "log";
}
kind;
`)
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "log" {
		t.Fatalf("want log, got %T %v", result, result)
	}
}

func TestStdLogModuleRotatesFiles(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "log_rotate.gs")
	logFile := filepath.Join(dir, "rotate.log")
	appSource := strings.ReplaceAll(`
let fs = require("@std/fs");
let log = require("@std/log");

let logger = log.createFileLogger("__LOG__", {
  append: false,
  timestamp: false,
  maxSizeBytes: 40,
  maxBackups: 2,
});

logger.info("first-line-aaaa");
logger.info("second-line-bbbb");
logger.info("third-line-cccc");
logger.close();

let current = fs.readFileSync("__LOG__");
let firstBackup = fs.readFileSync("__LOG__.1");
let kind = "rotate-bad";
if (current.includes("third-line-cccc") && firstBackup.includes("second-line-bbbb") && !current.includes("first-line-aaaa")) {
  kind = "rotate";
}
kind;
`, "__LOG__", strings.ReplaceAll(logFile, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "rotate" {
		t.Fatalf("want rotate, got %T %v", result, result)
	}
}

func TestStdConfigCodecModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "config_codecs.gs")
	work := filepath.Join(dir, "work")
	appSource := strings.ReplaceAll(`
let fs = require("@std/fs");
let path = require("@std/path");
let toml = require("@std/toml");
let yaml = require("@std/yaml");
let xml = require("@std/xml");

let root = "__WORK__";
fs.mkdirSync(root, { recursive: true });

let tomlDoc = toml.parse("[agent]\nname = \"coder\"\nsteps = [\"read\", \"write\"]\n");
let tomlFile = path.join(root, "agent.toml");
toml.writeFileSync(tomlFile, tomlDoc);
let tomlRead = toml.readFileSync(tomlFile);
let tomlText = toml.stringify(tomlRead);

let yamlDoc = yaml.parse("agent:\n  name: coder\n  enabled: true\n  tools:\n    - read\n    - write\n");
let yamlFile = path.join(root, "agent.yaml");
yaml.writeFileSync(yamlFile, yamlDoc);
let yamlRead = yaml.readFileSync(yamlFile);
let yamlText = yaml.stringify(yamlRead);

let xmlDoc = xml.parse("<agent name=\"coder\"><tool>read</tool><tool>write</tool></agent>");
let xmlFile = path.join(root, "agent.xml");
xml.writeFileSync(xmlFile, xmlDoc);
let xmlRead = xml.readFileSync(xmlFile);
let xmlText = xml.stringify(xmlRead);

let tomlKind = "bad";
if (tomlRead.agent.name === "coder" && tomlRead.agent.steps.length === 2 && tomlText.includes("agent")) {
  tomlKind = "toml";
}
let yamlKind = "bad";
if (yamlRead.agent.enabled && yamlRead.agent.tools[1] === "write" && yamlText.includes("agent")) {
  yamlKind = "yaml";
}
let xmlKind = "bad";
if (xmlRead.name === "agent" && xmlRead.attributes.name === "coder" && xmlRead.children[0].text === "read" && xmlText.includes("<agent")) {
  xmlKind = "xml";
}
tomlKind + ":" + yamlKind + ":" + xmlKind;
`, "__WORK__", strings.ReplaceAll(work, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "toml:yaml:xml" {
		t.Fatalf("want toml:yaml:xml, got %T %v", result, result)
	}
}

func TestStdExecCommandRunSuccess(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "exec_success.gs")
	if err := os.WriteFile(script, []byte(`
let exec = require("@std/exec");
let os = require("@std/os");
let cmd = exec.command("go", ["version"]);
let result = cmd.run();
let kind = "bad";
if (result.success) {
  kind = "ok";
}
kind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok" {
		t.Fatalf("want ok, got %T %v", result, result)
	}
}

func TestStdExecSpawnInteractive(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "exec_spawn.gs")
	appSource := strings.ReplaceAll(`
let exec = require("@std/exec");
let process = require("@std/process");
let cmdName = "bash";
let cmdArgs = ["-lc", "while IFS= read -r line; do echo got:$line; done"];
if (process.env.OS === "Windows_NT") {
  cmdName = "powershell";
  cmdArgs = ["-NoProfile", "-Command", "$input | ForEach-Object { 'got:' + $_ }"];
}
let child = exec.spawn(cmdName, cmdArgs, { cwd: "__DIR__", env: { GOSCRIPT_SPAWN_TEST: "ok" } });
child.writeln("one");
child.stdin.writeln("two");
child.closeStdin();
let first = child.stdout.readLine();
let second = child.stdout.readLine();
let end = child.stdout.readLine();
let result = child.wait();
let endKind = "not-end";
if (end === null) {
  endKind = "end";
}
let okKind = "bad";
if (result.success) {
  okKind = "ok";
}
first + ":" + second + ":" + endKind + ":" + okKind;
`, "__DIR__", strings.ReplaceAll(dir, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "got:one:got:two:end:ok" {
		t.Fatalf("want got:one:got:two:end:ok, got %T %v", result, result)
	}
}

func TestStdPTYSpawnInteractive(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pty_spawn.gs")
	appSource := strings.ReplaceAll(`
let pty = require("@std/pty");
let process = require("@std/process");
let cmdName = "bash";
let cmdArgs = ["-lc", "echo got:pty"];
if (process.env.OS === "Windows_NT") {
  cmdName = "C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe";
  cmdArgs = ["-NoProfile", "-Command", "Write-Output 'got:pty'"];
}
let term = pty.spawn(cmdName, cmdArgs, { cwd: "__DIR__", cols: 80, rows: 24 });
let output = "";
for (let i = 0; i < 8; i = i + 1) {
  let chunk = term.readText(4096, 500);
  if (chunk === null) {
  } else {
    output = output + chunk;
    if (output.includes("got:pty")) {
      i = 8;
    }
  }
}
term.resize(100, 30);
let result = term.wait();
term.close();
let gotKind = "bad";
if (output.includes("got:pty")) {
  gotKind = "got";
}
let okKind = "bad";
if (result.success) {
  okKind = "ok";
}
gotKind + ":" + okKind;
`, "__DIR__", strings.ReplaceAll(dir, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "got:ok" {
		t.Fatalf("want got:ok, got %T %v", result, result)
	}
}

func TestStdPTYReadTextTimeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pty_timeout.gs")
	appSource := strings.ReplaceAll(`
let pty = require("@std/pty");
let process = require("@std/process");
let cmdName = "bash";
let cmdArgs = ["-lc", "sleep 0.2; echo late"];
if (process.env.OS === "Windows_NT") {
  cmdName = "C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe";
  cmdArgs = ["-NoProfile", "-Command", "Start-Sleep -Milliseconds 200; Write-Output late"];
}
let term = pty.spawn(cmdName, cmdArgs, { cwd: "__DIR__", cols: 80, rows: 24 });
let early = term.readText(4096, 50);
let late = "";
for (let i = 0; i < 8; i = i + 1) {
  let chunk = term.readTextTimeout(4096, 500);
  if (chunk === null) {
  } else {
    late = late + chunk;
    if (late.includes("late")) {
      i = 8;
    }
  }
}
let result = term.wait();
term.close();
let kind = "bad";
if (late.includes("late") && result.success) {
  kind = "ok";
}
kind;
`, "__DIR__", strings.ReplaceAll(dir, `\`, `\\`))
	if err := os.WriteFile(script, []byte(appSource), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: 3 * time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok" {
		t.Fatalf("want ok, got %T %v", result, result)
	}
}

func TestStdTerminalModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "terminal.gs")
	if err := os.WriteFile(script, []byte(`
let terminal = require("@std/terminal");
let tty = terminal.isTTY("stdout");
let size = terminal.size();
let writeCount = terminal.write("");
let ttyKind = "bool";
if (tty === true || tty === false) {
  ttyKind = "bool";
}
let sizeKind = "bad";
if (size.cols > 0 && size.rows > 0) {
  sizeKind = "size";
}
let writeKind = "bad";
if (writeCount === 0) {
  writeKind = "write";
}
ttyKind + ":" + sizeKind + ":" + writeKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "bool:size:write" {
		t.Fatalf("want bool:size:write, got %T %v", result, result)
	}
}

func TestStdTerminalStartModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "terminal_start.gs")
	if err := os.WriteFile(script, []byte(`
let terminal = require("@std/terminal");
let session = terminal.start({ raw: false, bracketedPaste: false });
let size = session.size();
let writeCount = session.write("");
session.hideCursor();
session.showCursor();
session.disableBracketedPaste();
session.stop();
session.stop();
let sizeKind = "bad";
if (size.cols > 0 && size.rows > 0) {
  sizeKind = "size";
}
let writeKind = "bad";
if (writeCount === 0) {
  writeKind = "write";
}
sizeKind + ":" + writeKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "size:write" {
		t.Fatalf("want size:write, got %T %v", result, result)
	}
}

func TestStdTerminalModuleListeners(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "terminal_listeners.gs")
	if err := os.WriteFile(script, []byte(`
let terminal = require("@std/terminal");
function onResize(size) {
  return size.cols;
}
let resizeHandle = terminal.onResize(onResize);
let removedResize = terminal.offResize(onResize);
resizeHandle.stop();
let session = terminal.start({ raw: false, onResize: onResize });
session.stop();
let kind = "bad";
if (removedResize === 1) {
  kind = "ok";
}
kind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok" {
		t.Fatalf("want ok, got %T %v", result, result)
	}
}

func TestStdTerminalOnInputWithPTY(t *testing.T) {
	dir := t.TempDir()
	childScript := filepath.Join(dir, "terminal_child.gs")
	childSource := `
let terminal = require("@std/terminal");
let timers = require("@std/timers");
let seen = "";
let session = null;
function onInput(data) {
  seen = seen + data;
  if (seen.includes("xy")) {
    println("input:" + seen);
    session.stop();
  }
}
session = terminal.start({ raw: true, onInput: onInput });
timers.setTimeout(function() {
  println("timeout:" + seen);
  session.stop();
}, 1000);
`
	if err := os.WriteFile(childScript, []byte(childSource), 0644); err != nil {
		t.Fatal(err)
	}
	output := runGsInPTY(t, childScript, func(pty gopty.Pty) {
		_, _ = pty.Write([]byte("xy"))
	})
	if !strings.Contains(output, "input:xy") {
		t.Fatalf("want input:xy, got %q", output)
	}
}

func runGsInPTY(t *testing.T, script string, interact func(gopty.Pty)) string {
	t.Helper()
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	pty, err := gopty.New()
	if err != nil {
		t.Fatal(err)
	}
	defer pty.Close()
	_ = pty.Resize(80, 24)
	cmd := pty.Command(goPath, "run", "./cmd/gs", script)
	cmd.Dir = repoRoot(t)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	outputCh := make(chan string, 1)
	go func() {
		var out strings.Builder
		reader := bufio.NewReader(pty)
		buf := make([]byte, 4096)
		deadline := time.Now().Add(4 * time.Second)
		for time.Now().Before(deadline) {
			chunk, err := reader.Read(buf)
			if chunk > 0 {
				out.Write(buf[:chunk])
				text := out.String()
				if strings.Contains(text, "input:xy") ||
					strings.Contains(text, "resize:100x30") ||
					strings.Contains(text, "timeout:") {
					break
				}
			}
			if err != nil {
				break
			}
		}
		outputCh <- out.String()
	}()
	if interact != nil {
		interact(pty)
	}

	var output string
	select {
	case output = <-outputCh:
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		t.Fatal("pty output timed out")
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
	return output
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestStdStreamAndSSEModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "stream_sse.gs")
	if err := os.WriteFile(script, []byte(`
let stream = require("@std/stream");
let sse = require("@std/sse");

let raw = stream.fromString("abcdef");
let bytes = raw.read(2);
let rawText = raw.readText(3);
let rawRest = raw.readAll();
raw.close();
let bytesLenKind = "bad-len";
if (bytes.length === 2) {
  bytesLenKind = "len2";
}
let firstByteKind = "bad-byte";
if (bytes[0] === 97) {
  firstByteKind = "byte97";
}

let lines = stream.fromString("one\ntwo\n");
let lineOne = lines.readLine();
let lineTwo = lines.readLine();
let lineEnd = lines.readLine();
let lineEndKind = "not-end";
if (lineEnd === null) {
  lineEndKind = "end";
}

let body = stream.fromString("data: one\n\nevent: done\ndata: two\n\n");
let reader = sse.reader(body);
let first = reader.next();
let second = reader.next();
let end = reader.next();
let endKind = "not-end";
if (end === null) {
  endKind = "end";
}
bytesLenKind + ":" + firstByteKind + ":" + rawText + ":" + rawRest + ":" + lineOne + ":" + lineTwo + ":" + lineEndKind + ":" + first.data + ":" + second.type + ":" + second.data + ":" + endKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "len2:byte97:cde:f:one:two:end:one:done:two:end"
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
	}
}

func TestStdHTTPStreamModule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: first\n\n"))
		_, _ = w.Write([]byte("event: done\ndata: second\n\n"))
	}))
	defer server.Close()

	dir := t.TempDir()
	script := filepath.Join(dir, "http_stream.gs")
	source := strings.ReplaceAll(`
let http = require("@std/net/http/client");
let sse = require("@std/sse");
let resp = http.stream({ url: "__URL__", timeoutMs: 1000 });
let reader = sse.reader(resp.body);
let first = reader.next();
let second = reader.next();
resp.body.close();
let okKind = "bad";
if (resp.ok) {
  okKind = "ok";
}
okKind + ":" + first.data + ":" + second.type + ":" + second.data;
`, "__URL__", server.URL)
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok:first:done:second" {
		t.Fatalf("want ok:first:done:second, got %T %v", result, result)
	}
}

func TestStdHTTPFetchStreamOption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("alpha\n"))
		_, _ = w.Write([]byte("beta\n"))
	}))
	defer server.Close()

	dir := t.TempDir()
	script := filepath.Join(dir, "http_fetch_stream.gs")
	source := strings.ReplaceAll(`
let http = require("@std/net/http/client");
let resp = http.fetch({ url: "__URL__", stream: true, timeoutMs: 1000 });
let first = resp.body.readLine();
let second = resp.body.readLine();
resp.close();
let okKind = "bad";
if (resp.ok) {
  okKind = "ok";
}
okKind + ":" + first + ":" + second;
`, "__URL__", server.URL)
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ok:alpha:beta" {
		t.Fatalf("want ok:alpha:beta, got %T %v", result, result)
	}
}

func TestStdWebFrameworkRoutes(t *testing.T) {
	dir := t.TempDir()
	portFile := filepath.Join(dir, "port.txt")
	script := filepath.Join(dir, "web_app.gs")
	source := strings.ReplaceAll(`
let web = require("@std/web");
let fs = require("@std/fs");

let app = web.createApp();

app.use(function(req, res, next) {
  res.setHeader("X-Web-Test", "yes");
  next();
});

app.get("/users/:id", function(req, res) {
  res.status(201).json({
    id: req.params.id,
    q: req.query.q,
    method: req.method,
  });
});

app.post("/echo", function(req, res) {
  res.send("echo:" + req.body);
});

let server = app.listen(0);
fs.writeTextSync("__PORT_FILE__", server.port.toString());
"ready";
`, "__PORT_FILE__", strings.ReplaceAll(portFile, `\`, `\\`))
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ready" {
		t.Fatalf("want ready, got %T %v", result, result)
	}

	portBytes, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://127.0.0.1:" + strings.TrimSpace(string(portBytes))
	var getResp *http.Response
	for i := 0; i < 20; i++ {
		getResp, err = http.Get(baseURL + "/users/42?q=ok")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", getResp.StatusCode)
	}
	if getResp.Header.Get("X-Web-Test") != "yes" {
		t.Fatalf("middleware header missing")
	}
	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"id":"42"`) || !strings.Contains(string(body), `"q":"ok"`) || !strings.Contains(string(body), `"method":"GET"`) {
		t.Fatalf("unexpected json body: %s", body)
	}

	postResp, err := http.Post(baseURL+"/echo", "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()
	postBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(postBody) != "echo:hello" {
		t.Fatalf("want echo:hello, got %q", postBody)
	}
}

func TestStdWebFrameworkMiddlewareHelpers(t *testing.T) {
	dir := t.TempDir()
	portFile := filepath.Join(dir, "port.txt")
	publicDir := filepath.Join(dir, "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "hello.txt"), []byte("static hello"), 0644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "web_helpers.gs")
	source := strings.NewReplacer(
		"__PORT_FILE__", strings.ReplaceAll(portFile, `\`, `\\`),
		"__PUBLIC_DIR__", strings.ReplaceAll(publicDir, `\`, `\\`),
	).Replace(`
let web = require("@std/web");
let fs = require("@std/fs");

let app = web.createApp();

app.use("/assets", web.static("__PUBLIC_DIR__"));
app.use(web.json());

app.post("/json",
  function(req, res, next) {
    req.seen = "first";
    next();
  },
  function(req, res) {
    res.json({
      seen: req.seen,
      name: req.body.name,
      count: req.body.count,
      raw: req.rawBody.includes("ada"),
    });
  }
);

app.get("/chain",
  function(req, res, next) {
    req.mark = "one";
    next();
  },
  function(req, res) {
    res.send(req.mark + ":two");
  }
);

let server = app.listen(0);
fs.writeTextSync("__PORT_FILE__", server.port.toString());
"ready";
`)
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ready" {
		t.Fatalf("want ready, got %T %v", result, result)
	}

	portBytes, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://127.0.0.1:" + strings.TrimSpace(string(portBytes))
	client := &http.Client{}
	var jsonResp *http.Response
	for i := 0; i < 20; i++ {
		req, reqErr := http.NewRequest("POST", baseURL+"/json", strings.NewReader(`{"name":"ada","count":2}`))
		if reqErr != nil {
			t.Fatal(reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		jsonResp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer jsonResp.Body.Close()
	jsonBody, err := io.ReadAll(jsonResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonBody), `"seen":"first"`) || !strings.Contains(string(jsonBody), `"name":"ada"`) || !strings.Contains(string(jsonBody), `"count":2`) || !strings.Contains(string(jsonBody), `"raw":true`) {
		t.Fatalf("unexpected json middleware body: %s", jsonBody)
	}

	chainResp, err := http.Get(baseURL + "/chain")
	if err != nil {
		t.Fatal(err)
	}
	defer chainResp.Body.Close()
	chainBody, err := io.ReadAll(chainResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(chainBody) != "one:two" {
		t.Fatalf("want one:two, got %q", chainBody)
	}

	staticResp, err := http.Get(baseURL + "/assets/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer staticResp.Body.Close()
	staticBody, err := io.ReadAll(staticResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(staticBody) != "static hello" {
		t.Fatalf("want static hello, got %q", staticBody)
	}
}

func TestStdWebFrameworkProxyMiddleware(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost {
			t.Fatalf("want POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/items" || r.URL.RawQuery != "q=ok" {
			t.Fatalf("unexpected upstream URL: %s", r.URL.String())
		}
		if string(body) != "hello" {
			t.Fatalf("want forwarded body, got %q", string(body))
		}
		if r.Header.Get("X-Forwarded-Host") == "" {
			t.Fatalf("missing X-Forwarded-Host")
		}
		w.Header().Set("X-Proxy-Test", "yes")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("proxied:" + r.Header.Get("X-Extra")))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	portFile := filepath.Join(dir, "port.txt")
	script := filepath.Join(dir, "web_proxy.gs")
	source := strings.NewReplacer(
		"__PORT_FILE__", strings.ReplaceAll(portFile, `\`, `\\`),
		"__UPSTREAM__", upstream.URL,
	).Replace(`
let web = require("@std/web");
let fs = require("@std/fs");

let app = web.createApp();
app.use("/api", web.proxy({
  target: "__UPSTREAM__/v1",
  stripPrefix: "/api",
  headers: { "X-Extra": "native" },
}));

let server = app.listen(0);
fs.writeTextSync("__PORT_FILE__", server.port.toString());
"ready";
`)
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "ready" {
		t.Fatalf("want ready, got %T %v", result, result)
	}

	portBytes, err := os.ReadFile(portFile)
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://127.0.0.1:" + strings.TrimSpace(string(portBytes))
	resp, err := http.Post(baseURL+"/api/items?q=ok", "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("want 202, got %d: %s", resp.StatusCode, string(data))
	}
	if resp.Header.Get("X-Proxy-Test") != "yes" {
		t.Fatalf("missing upstream header")
	}
	if string(data) != "proxied:native" {
		t.Fatalf("want proxied:native, got %q", string(data))
	}
}

func TestAgentAnthropicProviderWithMockServer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("missing anthropic api key header")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if !strings.Contains(string(body), `"tools"`) || !strings.Contains(string(body), `"read_task"`) {
				t.Fatalf("first request missing tool schema: %s", body)
			}
			_, _ = w.Write([]byte(`{
  "id": "msg_1",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_1",
      "name": "read_task",
      "input": { "path": "task.txt" }
    }
  ],
  "stop_reason": "tool_use"
}`))
			return
		}
		if !strings.Contains(string(body), `"tool_result"`) || !strings.Contains(string(body), `"toolu_1"`) {
			t.Fatalf("second request missing tool result: %s", body)
		}
		_, _ = w.Write([]byte(`{
  "id": "msg_2",
  "type": "message",
  "role": "assistant",
  "content": [
    { "type": "text", "text": "mock anthropic completed" }
  ],
  "stop_reason": "end_turn"
}`))
	}))
	defer server.Close()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dir, err := os.MkdirTemp(root, ".agent-anthropic-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	script := filepath.Join(dir, "anthropic_mock.gs")
	source := strings.ReplaceAll(`
import { createAgent } from "@agent/core/agent";
import { createRegistry, createTool } from "@agent/tools/registry";
import { createAnthropicProvider } from "@agent/llm/anthropic";

let registry = createRegistry();
registry.register(createTool(
  "read_task",
  "Read a task by path.",
  {
    type: "object",
    required: ["path"],
    additionalProperties: false,
    properties: {
      path: { type: "string" },
    },
  },
  function(args) {
    return { path: args.path, content: "mock task" };
  }
));

let provider = createAnthropicProvider({
  apiKey: "test-key",
  baseUrl: "__URL__",
  model: "claude-test",
  maxTokens: 128,
});

let agent = createAgent({
  provider: provider,
  registry: registry,
  maxTurns: 4,
});

let answer = agent.run("read task");
answer.content;
`, "__URL__", server.URL)
	if err := os.WriteFile(script, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	runner := newRunner(options{workers: 1, timeout: 5 * time.Second})
	result, err := runner.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "mock anthropic completed" {
		t.Fatalf("want mock anthropic completed, got %T %v", result, result)
	}
	if requests != 2 {
		t.Fatalf("want 2 anthropic requests, got %d", requests)
	}
}

func TestAgentExampleReadsLocalTomlProviderConfig(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "toml-key" {
			t.Fatalf("missing toml api key header")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		requests++
		if !strings.Contains(string(body), `"model":"toml-model"`) {
			t.Fatalf("request did not use toml model: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if !strings.Contains(string(body), `"read_task"`) {
				t.Fatalf("first request missing workspace tool: %s", body)
			}
			_, _ = w.Write([]byte(`{
  "id": "msg_1",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_toml",
      "name": "read_task",
      "input": { "path": "task.txt" }
    }
  ],
  "stop_reason": "tool_use"
}`))
			return
		}
		if !strings.Contains(string(body), `"tool_result"`) || !strings.Contains(string(body), `"toolu_toml"`) {
			t.Fatalf("second request missing tool result: %s", body)
		}
		_, _ = w.Write([]byte(`{
  "id": "msg_2",
  "type": "message",
  "role": "assistant",
  "content": [
    { "type": "text", "text": "toml provider completed" }
  ],
  "stop_reason": "end_turn"
}`))
	}))
	defer server.Close()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	example := filepath.Join(root, "examples", "15-gs-agent")
	localConfig := filepath.Join(example, "agent.local.toml")
	writeTestFile(t, localConfig, strings.ReplaceAll(`
[agent]
provider = "anthropic"
system = "Use the configured model."
maxTurns = 4

[llm.anthropic]
apiKey = "toml-key"
baseUrl = "__URL__"
model = "toml-model"
maxTokens = 64
timeoutMs = 5000
`, "__URL__", server.URL))
	t.Cleanup(func() {
		_ = os.Remove(localConfig)
		_ = os.RemoveAll(filepath.Join(example, ".agent"))
	})

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	runWd := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(runWd); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: 5 * time.Second})
	if err := r.runProject(example); err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("want 2 anthropic requests, got %d", requests)
	}
	if _, err := os.Stat(filepath.Join(example, ".agent", "session.jsonl")); err != nil {
		t.Fatalf("expected example session file in project root: %v", err)
	}
}

func TestStdDBSQLiteModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "db_sqlite.gs")
	if err := os.WriteFile(script, []byte(`
let dbmod = require("@std/db");
let conn = dbmod.open("sqlite", ":memory:");
conn.exec("create table users (id integer primary key, name text, age integer)");
conn.exec("insert into users (name, age) values (?, ?)", ["Ada", 36]);
conn.exec("insert into users (name, age) values (?, ?)", ["Linus", 55]);
let rows = conn.query("select name, age from users where age > ? order by age", [40]);
let one = conn.queryOne("select name from users where name = ?", ["Ada"]);
conn.close();
let rowsKind = "rows-bad";
if (rows.length === 1) {
  rowsKind = "rows-ok";
}
rowsKind + ":" + rows[0].name + ":" + one.name;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "rows-ok:Linus:Ada" {
		t.Fatalf("want rows-ok:Linus:Ada, got %T %v", result, result)
	}
}

func TestStdDBSQLiteAdvancedModule(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "db_sqlite_advanced.gs")
	if err := os.WriteFile(script, []byte(`
let dbmod = require("@std/db");
let conn = dbmod.open("sqlite", ":memory:");
conn.setMaxOpenConns(1);
conn.setMaxIdleConns(1);
conn.exec("create table items (id integer primary key, name text)");

let tx = conn.begin();
let insert = tx.prepare("insert into items (name) values (?)");
insert.exec(["committed"]);
insert.close();
tx.commit();

let rolled = conn.begin();
rolled.exec("insert into items (name) values (?)", ["rolled-back"]);
rolled.rollback();

let select = conn.prepare("select name from items order by id");
let rows = select.query();
select.close();
let one = conn.queryOne("select name from items where id = ?", [1]);
conn.close();

let rowsKind = "bad";
if (rows.length === 1) {
  rowsKind = "one-row";
}
rowsKind + ":" + rows[0].name + ":" + one.name;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "one-row:committed:committed" {
		t.Fatalf("want one-row:committed:committed, got %T %v", result, result)
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

func TestImportPackageDependencyExports(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor", "tools")
	srcDir := filepath.Join(vendorDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte(`
[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "project.toml"), []byte(`
[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
"./extra" = "src/extra.gs"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.gs"), []byte(`export const label = "pkg";`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "extra.gs"), []byte(`export function suffix(x) { return x + ":extra"; }`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(app, []byte(`
import { label } from "tools";
import { suffix } from "tools/extra";
suffix(label);
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "pkg:extra" {
		t.Fatalf("want pkg:extra, got %T %v", result, result)
	}
}

func TestPackageDependencyUsesModuleCache(t *testing.T) {
	dir := t.TempDir()
	vendorDir := filepath.Join(dir, "vendor", "tools", "src")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte(`
[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "tools", "project.toml"), []byte(`
[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "index.gs"), []byte(`
let state = { count: 0 };
state.count = state.count + 1;
export { state };
`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(app, []byte(`
let a = require("tools");
a.state.count = a.state.count + 1;
let b = require("tools");
b.state.count;
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
		t.Fatalf("want cached package state 2, got %T %v", result, result)
	}
}

func TestPackageImportAliasRuntime(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "internal"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte(`
[package]
name = "app"
main = "src/app.gs"

[imports]
"#util" = "src/internal/util.gs"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "internal", "util.gs"), []byte(`export function label(x) { return "private:" + x; }`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(srcDir, "app.gs")
	if err := os.WriteFile(app, []byte(`
import { label } from "#util";
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
	if !ok || str.Value != "private:ok" {
		t.Fatalf("want private:ok, got %T %v", result, result)
	}
}

func TestPackageAbsoluteStyleAliasRuntime(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "project.toml"), `[package]
name = "app"
main = "src/app.gs"

[imports]
"@/*" = "src/*.gs"
`)
	writeTestFile(t, filepath.Join(dir, "src", "internal", "util.gs"), `export function label(x) { return "alias:" + x; }`)
	app := filepath.Join(dir, "src", "app.gs")
	writeTestFile(t, app, `
import { label } from "@/internal/util";
label("ok");
`)

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "alias:ok" {
		t.Fatalf("want alias:ok, got %T %v", result, result)
	}
}

func TestPackageDependencyResolvesOwnDependencies(t *testing.T) {
	dir := t.TempDir()
	toolsDir := filepath.Join(dir, "vendor", "tools")
	helperDir := filepath.Join(toolsDir, "vendor", "helper")
	if err := os.MkdirAll(filepath.Join(toolsDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(helperDir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.toml"), []byte(`
[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "project.toml"), []byte(`
[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[dependencies]
"helper" = "file:vendor/helper"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "project.toml"), []byte(`
[package]
name = "helper"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "src", "index.gs"), []byte(`export const label = "nested";`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "src", "index.gs"), []byte(`
import { label } from "helper";
export const value = "tools:" + label;
`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(app, []byte(`
import { value } from "tools";
value;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "tools:nested" {
		t.Fatalf("want tools:nested, got %T %v", result, result)
	}
}

func TestImportPackageFileDependency(t *testing.T) {
	dir := t.TempDir()
	pkgRoot := filepath.Join(dir, "tools-src")
	writeTestFile(t, filepath.Join(pkgRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`)
	writeTestFile(t, filepath.Join(pkgRoot, "src", "index.gs"), `
import { decorate } from "./format";
export const value = decorate("pkgfile");
`)
	writeTestFile(t, filepath.Join(pkgRoot, "src", "format.gs"), `
export function decorate(x) { return "[" + x + "]"; }
`)
	pkgPath := filepath.Join(dir, "vendor", "tools.gspkg")
	if err := packCommand([]string{pkgRoot, pkgPath}); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(dir, "project.toml"), `[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	app := filepath.Join(dir, "app.gs")
	writeTestFile(t, app, `
import { value } from "tools";
value;
`)

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "[pkgfile]" {
		t.Fatalf("want [pkgfile], got %T %v", result, result)
	}
}

func TestImportPackageFileDependencyWithAbsoluteStyleAlias(t *testing.T) {
	dir := t.TempDir()
	pkgRoot := filepath.Join(dir, "tools-src")
	writeTestFile(t, filepath.Join(pkgRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[imports]
"@/*" = "src/*.gs"
`)
	writeTestFile(t, filepath.Join(pkgRoot, "src", "index.gs"), `
import { decorate } from "@/internal/format";
export const value = decorate("pkgfile-alias");
`)
	writeTestFile(t, filepath.Join(pkgRoot, "src", "internal", "format.gs"), `
export function decorate(x) { return "[" + x + "]"; }
`)
	pkgPath := filepath.Join(dir, "vendor", "tools.gspkg")
	if err := packCommand([]string{pkgRoot, pkgPath}); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(dir, "project.toml"), `[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	app := filepath.Join(dir, "app.gs")
	writeTestFile(t, app, `
import { value } from "tools";
value;
`)

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "[pkgfile-alias]" {
		t.Fatalf("want [pkgfile-alias], got %T %v", result, result)
	}
}

func TestImportPackageFileNestedDependency(t *testing.T) {
	dir := t.TempDir()
	helperRoot := filepath.Join(dir, "helper-src")
	writeTestFile(t, filepath.Join(helperRoot, "project.toml"), `[package]
name = "helper"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"
`)
	writeTestFile(t, filepath.Join(helperRoot, "src", "index.gs"), `
export const label = "nested-pkgfile";
`)
	helperPkg := filepath.Join(dir, "helper.gspkg")
	if err := packCommand([]string{helperRoot, helperPkg}); err != nil {
		t.Fatal(err)
	}

	toolsRoot := filepath.Join(dir, "tools-src")
	writeTestFile(t, filepath.Join(toolsRoot, "project.toml"), `[package]
name = "tools"
version = "1.0.0"
main = "src/index.gs"

[exports]
"." = "src/index.gs"

[dependencies]
"helper" = "file:vendor/helper.gspkg"
`)
	writeTestFile(t, filepath.Join(toolsRoot, "src", "index.gs"), `
import { label } from "helper";
export const value = "tools:" + label;
`)
	toolsHelperPkg := filepath.Join(toolsRoot, "vendor", "helper.gspkg")
	if err := os.MkdirAll(filepath.Dir(toolsHelperPkg), 0755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(helperPkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(toolsHelperPkg, data, 0644); err != nil {
		t.Fatal(err)
	}
	toolsPkg := filepath.Join(dir, "vendor", "tools.gspkg")
	if err := packCommand([]string{toolsRoot, toolsPkg}); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(dir, "project.toml"), `[project]
entry = "app.gs"

[dependencies]
"tools" = "file:vendor/tools.gspkg"
`)
	app := filepath.Join(dir, "app.gs")
	writeTestFile(t, app, `
import { value } from "tools";
value;
`)

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "tools:nested-pkgfile" {
		t.Fatalf("want tools:nested-pkgfile, got %T %v", result, result)
	}
}

func TestRequireReturnsAssignedModuleExports(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lib.gs"), []byte(`
module.exports = function label(x) { return "module:" + x; };
`), 0644); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "app.gs")
	if err := os.WriteFile(app, []byte(`
let label = require("./lib");
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
	if !ok || str.Value != "module:ok" {
		t.Fatalf("want module:ok, got %T %v", result, result)
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
		filepath.Join(root, "docs", "examples", "sqlite.gs"),
		filepath.Join(root, "examples", "16-native-stdlib.gs"),
		filepath.Join(root, "examples", "17-native-stdlib-cookbook.gs"),
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

	t.Run("examples/13-package-modules", func(t *testing.T) {
		r := newRunner(options{workers: 1, timeout: time.Second})
		if err := r.runProject(filepath.Join(root, "examples", "13-package-modules")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("examples/14-nested-gspkg", func(t *testing.T) {
		r := newRunner(options{workers: 1, timeout: time.Second})
		if err := r.runProject(filepath.Join(root, "examples", "14-nested-gspkg")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("examples/15-gs-agent", func(t *testing.T) {
		r := newRunner(options{workers: 1, timeout: time.Second})
		if err := r.runProject(filepath.Join(root, "examples", "15-gs-agent")); err != nil {
			t.Fatal(err)
		}
	})
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func captureRunOutput(t *testing.T, args []string) (string, string, int) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	code := run(args)

	if err := stdoutWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatal(err)
	}
	stdout, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatal(err)
	}
	return string(stdout), string(stderr), code
}
