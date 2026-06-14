package stdlib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
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
let app = web.createApp({ concurrency: "serial" });
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
let app = web.createApp({ concurrency: "serial" });
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

func TestWebResponseWriteAndFlush(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let app = web.createApp({ concurrency: "serial" });
app.get("/chunks", function(req, res) {
  res.status(202);
  res.setHeader("Content-Type", "text/event-stream");
  res.write("data: one\n\n");
  res.flush();
  res.write("data: two\n\n");
  res.end();
});
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Get(server.URL + "/chunks")
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
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("want SSE content type, got %q", got)
	}
	if string(data) != "data: one\n\ndata: two\n\n" {
		t.Fatalf("unexpected chunk body %q", string(data))
	}
}

// TestWebSerialHandlersAreSerializedForConcurrentRequests pins the serial-mode
// contract: with concurrency:"serial", the handler chain runs under a global
// mutex, so module-top-level shared state stays consistent and maxActive == 1.
// Opt-in serial is preserved for apps that depend on shared closure state.
func TestWebSerialHandlersAreSerializedForConcurrentRequests(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let app = web.createApp({ concurrency: "serial" });
let count = 0;
let active = 0;
let maxActive = 0;
app.get("/hit", function(req, res) {
  active = active + 1;
  if (active > maxActive) {
    maxActive = active;
  }
  sleep(5);
  let next = count + 1;
  count = next;
  active = active - 1;
  res.send(String(next));
});
app.get("/count", function(req, res) {
  res.send(String(count));
});
app.get("/max-active", function(req, res) {
  res.send(String(maxActive));
});
let server = app.listen(0);
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	const requests = 32
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/hit")
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs <- fmt.Errorf("hit status %d: %s", resp.StatusCode, string(body))
				return
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	resp, err := http.Get(server.URL + "/count")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("count response %q is not a number: %v", string(data), err)
	}
	if got != requests {
		t.Fatalf("serialized handler count = %d, want %d", got, requests)
	}

	resp, err = http.Get(server.URL + "/max-active")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	maxActive, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("max-active response %q is not a number: %v", string(data), err)
	}
	if maxActive != 1 {
		t.Fatalf("serial max concurrent script handlers = %d, want 1", maxActive)
	}
}

// TestWebSerialAsyncHandlersCanWaitConcurrently is the serial-mode async
// contract: handlers may park on sleepAsync concurrently (maxWaiting > 1), but
// script fragments never overlap (maxScriptActive == 1). Preserved for the
// opt-in serial mode.
func TestWebSerialAsyncHandlersCanWaitConcurrently(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let app = web.createApp({ concurrency: "serial" });
let waiting = 0;
let maxWaiting = 0;
let scriptActive = 0;
let maxScriptActive = 0;

function enterScript() {
  scriptActive = scriptActive + 1;
  if (scriptActive > maxScriptActive) {
    maxScriptActive = scriptActive;
  }
}

function leaveScript() {
  scriptActive = scriptActive - 1;
}

app.get("/wait", function(req, res) {
  enterScript();
  waiting = waiting + 1;
  if (waiting > maxWaiting) {
    maxWaiting = waiting;
  }
  leaveScript();
  return sleepAsync(10).then(function() {
    enterScript();
    waiting = waiting - 1;
    res.send("ok");
    leaveScript();
  });
});
app.get("/max-waiting", function(req, res) {
  res.send(String(maxWaiting));
});
app.get("/max-script-active", function(req, res) {
  res.send(String(maxScriptActive));
});
let server = app.listen(0);
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	const requests = 24
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/wait")
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK || string(data) != "ok" {
				errs <- fmt.Errorf("wait status %d: %s", resp.StatusCode, string(data))
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	maxWaiting := readWebInt(t, server.URL+"/max-waiting")
	if maxWaiting <= 1 {
		t.Fatalf("async handlers did not overlap while waiting; maxWaiting = %d", maxWaiting)
	}
	maxScriptActive := readWebInt(t, server.URL+"/max-script-active")
	if maxScriptActive != 1 {
		t.Fatalf("script fragments overlapped inside serial worker; maxScriptActive = %d", maxScriptActive)
	}
}

func TestWebCreateAppRejectsUnsupportedConcurrency(t *testing.T) {
	result := evalWebTestScriptAllowRuntimeError(t, `
let web = require("@std/web");
web.createApp({ concurrency: "parallel" });
`)
	if !object.IsRuntimeError(result) {
		t.Fatalf("want runtime error, got %T: %s", result, result.Inspect())
	}
	if !strings.Contains(result.Inspect(), "unsupported concurrency mode") {
		t.Fatalf("unexpected error: %s", result.Inspect())
	}
}

func TestWebAppAllStarMatchesAnyPathAndMethod(t *testing.T) {
	appObj := evalWebTestScript(t, `
let web = require("@std/web");
let app = web.createApp({ concurrency: "serial" });
app.all("*", function(req, res) {
  res.status(209);
  res.send(req.method + " " + req.url);
});
app;
`)
	app := mustWebApp(t, appObj)
	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Post(server.URL+"/custom/path", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != 209 {
		t.Fatalf("want 209, got %d: %s", resp.StatusCode, string(data))
	}
	if string(data) != "POST /custom/path" {
		t.Fatalf("unexpected wildcard body %q", string(data))
	}
}

func evalWebTestScript(t *testing.T, src string) object.Object {
	t.Helper()
	result := evalWebTestScriptAllowRuntimeError(t, src)
	if object.IsRuntimeError(result) {
		t.Fatalf("runtime error: %s", result.Inspect())
	}
	return result
}

func evalWebTestScriptAllowRuntimeError(t *testing.T, src string) object.Object {
	t.Helper()
	vm := object.NewVirtualMachine()
	vm.SetBootstrapSource(src) // isolated mode replays this per request
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
	return evaluator.Eval(program, env)
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

func readWebInt(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("response %q is not a number: %v", string(data), err)
	}
	return got
}
