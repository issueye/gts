package stdlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestRuntimeRunToolExecutesExternalTool(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "tool.gs")
	if err := os.WriteFile(tool, []byte(`
exports.run = function(input) {
  return { ok: true, result: input.name + ":" + String(input.count) };
};
`), 0644); err != nil {
		t.Fatal(err)
	}
	input := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(input, "name", &object.String{Value: "dynamic"})
	setHashMember(input, "count", &object.Number{Value: 3})

	result := runtimeRunTool(object.NewEnvironment(), ast.Position{}, &object.String{Value: tool}, input)
	hash, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want hash, got %T: %s", result, result.Inspect())
	}
	okObj, _ := hashValue(hash, "ok")
	if okBool, ok := okObj.(*object.Boolean); !ok || !okBool.Value {
		t.Fatalf("want ok true, got %T: %s", okObj, okObj.Inspect())
	}
	resultObj, _ := hashValue(hash, "result")
	if resultText, ok := resultObj.(*object.String); !ok || resultText.Value != "dynamic:3" {
		t.Fatalf("want result dynamic:3, got %T: %s", resultObj, resultObj.Inspect())
	}
}

func TestRuntimeRunToolUsesToolRelativeRequires(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lib.gs"), []byte(`exports.prefix = "tool-lib";`), 0644); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(dir, "tool.gs")
	if err := os.WriteFile(tool, []byte(`
let lib = require("./lib");
exports.run = function(input) {
  return lib.prefix + ":" + input.value;
};
`), 0644); err != nil {
		t.Fatal(err)
	}
	input := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(input, "value", &object.String{Value: "ok"})

	result := runtimeRunTool(object.NewEnvironment(), ast.Position{}, &object.String{Value: tool}, input)
	assertString(t, result, "tool-lib:ok")
}

func TestRuntimeCallScriptCallsNamedExport(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "script.gs")
	if err := os.WriteFile(script, []byte(`
exports.convert = function(input, suffix) {
  return input.name + ":" + suffix;
};
`), 0644); err != nil {
		t.Fatal(err)
	}
	input := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(input, "name", &object.String{Value: "dynamic-script"})
	args := &object.Array{Elements: []object.Object{
		input,
		&object.String{Value: "ok"},
	}}

	result := runtimeCallScript(object.NewEnvironment(), ast.Position{}, &object.String{Value: script}, &object.String{Value: "convert"}, args)
	assertString(t, result, "dynamic-script:ok")
}

func TestRuntimeRunToolOptionsCwdAndArgv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("cwd-data"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(t.TempDir(), "tool.gs")
	if err := os.WriteFile(tool, []byte(`
let fs = require("@std/fs");
let process = require("@std/process");
exports.run = function(input) {
  return fs.readFileSync("data.txt") + ":" + process.argv.join("|");
};
`), 0644); err != nil {
		t.Fatal(err)
	}
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "cwd", &object.String{Value: dir})
	setHashMember(opts, "argv", strSliceToArray([]string{"agent.exe", "generated-tool", "--flag"}))

	result := runtimeRunTool(object.NewEnvironment(), ast.Position{}, &object.String{Value: tool}, object.UNDEFINED, opts)
	text, ok := result.(*object.String)
	if !ok {
		t.Fatalf("want string, got %T: %s", result, result.Inspect())
	}
	if !strings.Contains(text.Value, "cwd-data:agent.exe|generated-tool|--flag") {
		t.Fatalf("unexpected output %q", text.Value)
	}
}

func TestRuntimeRunScriptReturnsExports(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "script.gs")
	if err := os.WriteFile(script, []byte(`
exports.name = "generated";
exports.run = function() { return "ok"; };
`), 0644); err != nil {
		t.Fatal(err)
	}
	result := runtimeRunScript(object.NewEnvironment(), ast.Position{}, &object.String{Value: script})
	hash, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want hash, got %T: %s", result, result.Inspect())
	}
	name, _ := hashValue(hash, "name")
	assertString(t, name, "generated")
}
