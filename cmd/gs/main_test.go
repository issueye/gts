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
	want := "one\ntwo:one\ntwo:two-files:dirent:glob:real:lstat:tmp:removed"
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
okKind + ":" + badKind + ":" + crypto.sha256("abc") + ":" + bytesKind + ":" + uuidKind;
`), 0644); err != nil {
		t.Fatal(err)
	}

	r := newRunner(options{workers: 1, timeout: time.Second})
	result, err := r.evalFile(script, runOptions{})
	if err != nil {
		t.Fatal(err)
	}
	str, ok := result.(*object.String)
	want := "ok:bad:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad:bytes4:uuid"
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
  let chunk = term.readText(4096);
  if (chunk === null) {
    i = 8;
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
