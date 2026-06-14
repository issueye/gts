package stdlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSharedMapCrossVMObjectStorage verifies that shared.map can store and
// retrieve object literals across isolated VMs (different requests).
func TestSharedMapCrossVMObjectStorage(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 2 });
let store = shared.map("cross-vm-test");

app.post("/set-object", function(req, res) {
  store.set("user", { name: "alice", age: 30 });
  res.send("stored");
});

app.get("/get-object", function(req, res) {
  let user = store.get("user");
  if (user === undefined) {
    res.status(404).send("not-found");
    return;
  }
  // Try to access properties
  let name = user.name;
  let age = user.age;
  res.send("name=" + name + ",age=" + age);
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Request 1: store object (runs in VM-A)
	resp, err := http.Post(server.URL+"/set-object", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("POST /set-object: want 200, got %d", resp.StatusCode)
	}

	// Request 2: retrieve object (runs in VM-B, different from VM-A)
	resp, err = http.Get(server.URL + "/get-object")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GET /get-object: want 200, got %d: %s", resp.StatusCode, body)
	}

	want := "name=alice,age=30"
	if string(body) != want {
		t.Fatalf("GET /get-object: want %q, got %q", want, body)
	}

	t.Logf("Cross-VM object storage works: %s", body)
}

// TestSharedMapJSONParse tests if JSON.parse + shared.map work together.
func TestSharedMapJSONParse(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 2 });
let store = shared.map("json-test");

app.post("/store-json", function(req, res) {
  let data = JSON.parse(req.body);
  store.set("data", data);
  res.send("ok");
});

app.get("/get-json", function(req, res) {
  let data = store.get("data");
  if (data === undefined) {
    res.status(404).send("not-found");
    return;
  }
  res.send("username=" + data.username + ",password=" + data.password);
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Store JSON
	jsonBody := `{"username":"test","password":"secret"}`
	resp, err := http.Post(server.URL+"/store-json", "application/json", strings.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("POST /store-json: want 200, got %d", resp.StatusCode)
	}

	// Retrieve JSON
	resp, err = http.Get(server.URL + "/get-json")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GET /get-json: want 200, got %d: %s", resp.StatusCode, body)
	}

	want := "username=test,password=secret"
	if string(body) != want {
		t.Fatalf("GET /get-json: want %q, got %q", want, body)
	}

	t.Logf("JSON parse + shared.map works: %s", body)
}
