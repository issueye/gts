package main

import (
	"net/http"
	"net/http/httptest"
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

func TestStdPathModule(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "path.gs")
	if err := os.WriteFile(app, []byte(`
let path = require("@std/path");
path.basename(path.join("alpha", "beta", "file.txt")) + ":" + path.extname("file.txt");
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(app, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "file.txt:.txt" {
		t.Fatalf("want file.txt:.txt, got %T %v", result, result)
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
let text = fs.readTextSync(file);
let entries = fs.walkSync(root, { includeDirs: false });
let countKind = "bad";
if (entries.length === 1) {
  countKind = "one-file";
}
text + ":" + countKind + ":" + entries[0].relativePath;
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
	want := "one\ntwo:one-file:" + filepath.Join("nested", "note.txt")
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
let platformKind = "empty";
if (os.platform !== "") {
  platformKind = "set";
}
let tmpKind = "empty";
if (os.tmpdir() !== "") {
  tmpKind = "set";
}
process.chdir("__DIR__");
let after = process.cwd();
process.chdir(before);
process.unsetenv("GOSCRIPT_AGENT_TEST");
value + ":" + fallback + ":" + platformKind + ":" + tmpKind + ":" + after;
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
	want := "ok:missing:set:set:" + dir
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
okKind + ":" + badKind + ":" + crypto.sha256("abc") + ":" + bytesKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "ok:bad:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad:bytes4"
	if !ok || str.Value != want {
		t.Fatalf("want %q, got %T %v", want, result, result)
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

func TestStdStreamAndSSEModules(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "stream_sse.gs")
	if err := os.WriteFile(script, []byte(`
let stream = require("@std/stream");
let sse = require("@std/sse");
let body = stream.fromString("data: one\n\nevent: done\ndata: two\n\n");
let reader = sse.reader(body);
let first = reader.next();
let second = reader.next();
let end = reader.next();
let endKind = "not-end";
if (end === null) {
  endKind = "end";
}
first.data + ":" + second.type + ":" + second.data + ":" + endKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	if !ok || str.Value != "one:done:two:end" {
		t.Fatalf("want one:done:two:end, got %T %v", result, result)
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
