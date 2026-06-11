package stdlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

func TestWebResponseStreamSendsReadableStream(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let stream = require("@std/stream");
let app = web.createApp();
app.get("/events", function(req, res) {
  res.status(201);
  res.setHeader("Content-Type", "text/event-stream");
  res.stream(stream.fromString("data: one\n\ndata: two\n\n"));
});
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", resp.StatusCode, string(data))
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("want SSE content type, got %q", got)
	}
	if string(data) != "data: one\n\ndata: two\n\n" {
		t.Fatalf("unexpected stream body %q", string(data))
	}
}

func TestWebResponseSendAcceptsReadableStream(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let stream = require("@std/stream");
let app = web.createApp();
app.get("/body", function(req, res) {
  res.send(stream.fromString("hello from stream"));
});
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Get(server.URL + "/body")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", resp.StatusCode, string(data))
	}
	if string(data) != "hello from stream" {
		t.Fatalf("unexpected stream body %q", string(data))
	}
}

func evalWebTestScript(t *testing.T, src string) object.Object {
	t.Helper()
	vm := object.NewVirtualMachine()
	env := vm.NewEnvironment()
	module.SetupExports(env)
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})
	l := lexer.New(src)
	p := parser.New(l, "web_stream_test.gs")
	program := p.ParseProgram()
	if len(l.Errors()) > 0 || len(program.Errors) > 0 {
		t.Fatalf("parse errors: %v %v", l.Errors(), program.Errors)
	}
	result := evaluator.Eval(program, env)
	if object.IsRuntimeError(result) {
		t.Fatalf("runtime error: %s", result.Inspect())
	}
	return result
}

func mustWebApp(t *testing.T, obj object.Object) *webApp {
	t.Helper()
	hash, ok := obj.(*object.Hash)
	if !ok {
		t.Fatalf("want app hash, got %T: %s", obj, obj.Inspect())
	}
	raw, ok := hashValue(hash, "__webApp")
	if !ok {
		t.Fatalf("app missing __webApp")
	}
	goObj, ok := raw.(*object.GoObject)
	if !ok {
		t.Fatalf("want GoObject app, got %T: %s", raw, raw.Inspect())
	}
	app, ok := goObj.Value.(*webApp)
	if !ok {
		t.Fatalf("want *webApp, got %T", goObj.Value)
	}
	return app
}
