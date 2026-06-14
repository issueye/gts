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
)

// TestSharedCounterCrossRequestState verifies that shared.counter maintains
// state across isolated-mode requests (each request runs in its own VM, but
// the counter is process-global).
func TestSharedCounterCrossRequestState(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 4 });
let counter = shared.counter("test-hits");
app.get("/incr", function(req, res) {
  let newVal = counter.incr();
  res.send(String(newVal));
});
app.get("/get", function(req, res) {
  res.send(String(counter.get()));
});
let server = app.listen(0);
"booted";
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	const requests = 20
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/incr")
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs <- fmt.Errorf("incr status %d: %s", resp.StatusCode, body)
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

	resp, err := http.Get(server.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	got, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		t.Fatalf("counter.get response %q is not a number: %v", body, err)
	}
	if got != requests {
		t.Fatalf("shared counter after %d incr = %d, want %d", requests, got, requests)
	}
}

// TestSharedMapCrossRequestState verifies that shared.map stores values across
// isolated-mode requests.
func TestSharedMapCrossRequestState(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 2 });
let store = shared.map("test-store");
app.post("/set/:key", function(req, res) {
  store.set(req.params.key, req.body);
  res.send("ok");
});
app.get("/get/:key", function(req, res) {
  let val = store.get(req.params.key);
  if (val === undefined) {
    res.status(404).send("not found");
  } else {
    res.send(val);
  }
});
app.get("/keys", function(req, res) {
  res.json(store.keys());
});
let server = app.listen(0);
"booted";
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Set multiple keys concurrently
	keys := []string{"alpha", "beta", "gamma"}
	var wg sync.WaitGroup
	for _, k := range keys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			resp, err := http.Post(server.URL+"/set/"+key, "text/plain", strings.NewReader("value-"+key))
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}(k)
	}
	wg.Wait()

	// Get each key back
	for _, k := range keys {
		resp, err := http.Get(server.URL + "/get/" + k)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get %s status %d: %s", k, resp.StatusCode, body)
		}
		want := "value-" + k
		if string(body) != want {
			t.Fatalf("map.get(%s) = %q, want %q", k, body, want)
		}
	}

	// Verify all keys are present
	resp, err := http.Get(server.URL + "/keys")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	for _, k := range keys {
		if !strings.Contains(string(body), k) {
			t.Fatalf("map.keys() missing %s: %s", k, body)
		}
	}
}

// TestSharedAtomicCompareAndSwap verifies that shared.atomic provides CAS for
// optimistic concurrency patterns across isolated requests.
func TestSharedAtomicCompareAndSwap(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 4 });
let atom = shared.atomic("test-cas", 0);
app.get("/incr-cas", function(req, res) {
  // Optimistic increment via CAS retry loop
  let retries = 0;
  while (retries < 100) {
    let old = atom.get();
    let next = old + 1;
    if (atom.compareAndSwap(old, next)) {
      res.send(String(next));
      return;
    }
    retries = retries + 1;
  }
  res.status(500).send("cas-failed");
});
app.get("/get", function(req, res) {
  res.send(String(atom.get()));
});
let server = app.listen(0);
"booted";
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	const requests = 16
	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/incr-cas")
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs <- fmt.Errorf("incr-cas status %d: %s", resp.StatusCode, body)
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

	resp, err := http.Get(server.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	got, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		t.Fatalf("atomic.get response %q is not a number: %v", body, err)
	}
	if got != requests {
		t.Fatalf("shared atomic after %d CAS incr = %d, want %d", requests, got, requests)
	}
}
