package stdlib

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

func TestCLIExecutesSubcommandWithFlagsAndArgs(t *testing.T) {
	stdout, result := evalCLITestScript(t, `
let cli = require("@std/cli");
let root = cli.command({ use: "app", short: "demo app" });
root.persistentFlags().string("profile", "p", "dev", "profile name");
let serve = cli.command({
  use: "serve [dir]",
  aliases: ["s"],
  args: cli.exactArgs(1),
  run: function(cmd, args) {
    println(cmd.flag("profile") + ":" + String(cmd.flag("port")) + ":" + args[0]);
  },
});
serve.flags().int("port", "", 8080, "listen port");
root.addCommand(serve);
root.execute(["serve", "--profile", "prod", "--port", "9000", "public"]);
`)
	assertCLINumber(t, result, 0)
	if !strings.Contains(stdout, "prod:9000:public") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestCLIHelpSkipsRunAndShowsUsage(t *testing.T) {
	stdout, result := evalCLITestScript(t, `
let cli = require("@std/cli");
let root = cli.command({ use: "app", short: "demo app" });
root.command({
  use: "serve",
  short: "start server",
  run: function() { println("should-not-run"); },
});
root.execute(["serve", "--help"]);
`)
	assertCLINumber(t, result, 0)
	for _, want := range []string{"serve - start server", "Usage:", "--help"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("help output missing %q in:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "should-not-run") {
		t.Fatalf("help should not run command:\n%s", stdout)
	}
}

func TestCLIArgValidationFails(t *testing.T) {
	_, result := evalCLITestScriptAllowError(t, `
let cli = require("@std/cli");
let root = cli.command({ use: "app" });
root.command({
  use: "one",
  args: cli.exactArgs(1),
  run: function(cmd, args) { println(args[0]); },
});
root.execute(["one"]);
`)
	if !object.IsRuntimeError(result) || !strings.Contains(result.Inspect(), "accepts 1 argument") {
		t.Fatalf("want arg validation error, got %T: %s", result, result.Inspect())
	}
}

func TestCLIBoolShorthandCluster(t *testing.T) {
	stdout, result := evalCLITestScript(t, `
let cli = require("@std/cli");
let root = cli.command({
  use: "app",
  run: function(cmd, args) {
    println(String(cmd.flag("all")) + ":" + String(cmd.flag("verbose")));
  },
});
root.flags().bool("all", "a", false, "all items");
root.flags().bool("verbose", "v", false, "verbose output");
root.execute(["-av"]);
`)
	assertCLINumber(t, result, 0)
	if !strings.Contains(stdout, "true:true") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func TestCLIExecuteDefaultsToApplicationArgv(t *testing.T) {
	stdout, result := evalCLITestScriptWithArgv(t, []string{"gs.exe", "main.gs", "--mode", "packed", "input.txt"}, `
let cli = require("@std/cli");
let root = cli.command({
  use: "app",
  run: function(cmd, args) {
    println(cmd.flag("mode") + ":" + args[0]);
  },
});
root.flags().string("mode", "", "dev", "mode name");
root.execute();
`)
	assertCLINumber(t, result, 0)
	if !strings.Contains(stdout, "packed:input.txt") {
		t.Fatalf("unexpected stdout: %q", stdout)
	}
}

func evalCLITestScript(t *testing.T, src string) (string, object.Object) {
	t.Helper()
	stdout, result := evalCLITestScriptAllowError(t, src)
	if object.IsRuntimeError(result) {
		t.Fatalf("runtime error: %s", result.Inspect())
	}
	return stdout, result
}

func evalCLITestScriptAllowError(t *testing.T, src string) (string, object.Object) {
	t.Helper()
	return evalCLITestScriptWithArgvAllowError(t, nil, src)
}

func evalCLITestScriptWithArgv(t *testing.T, argv []string, src string) (string, object.Object) {
	t.Helper()
	stdout, result := evalCLITestScriptWithArgvAllowError(t, argv, src)
	if object.IsRuntimeError(result) {
		t.Fatalf("runtime error: %s", result.Inspect())
	}
	return stdout, result
}

func evalCLITestScriptWithArgvAllowError(t *testing.T, argv []string, src string) (string, object.Object) {
	t.Helper()
	vm := object.NewVirtualMachine()
	if argv != nil {
		vm.SetArgv(argv)
	}
	env := vm.NewEnvironment()
	module.SetupExports(env)
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})
	l := lexer.New(src)
	p := parser.New(l, "cli_test.gs")
	program := p.ParseProgram()
	if len(l.Errors()) > 0 || len(program.Errors) > 0 {
		t.Fatalf("parse errors: %v %v", l.Errors(), program.Errors)
	}

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = oldStdout }()

	result := evaluator.Eval(program, env)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return buf.String(), result
}

func assertCLINumber(t *testing.T, value object.Object, want float64) {
	t.Helper()
	num, ok := value.(*object.Number)
	if !ok || num.Value != want {
		t.Fatalf("want number %v, got %T: %s", want, value, value.Inspect())
	}
}
