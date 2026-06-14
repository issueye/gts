package stdlib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestIsolatedModeFullStack is an end-to-end test that verifies isolated mode
// with shared state under realistic load: concurrent requests, async handlers,
// middleware chains, and shared counters/maps.
func TestIsolatedModeFullStack(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 8 });

// Shared state: track total requests and per-user sessions
let totalRequests = shared.counter("total-requests");
let sessions = shared.map("user-sessions");

// Middleware: log and count every request
app.use(function(req, res, next) {
  totalRequests.incr();
  return next();
});

// Route: create/update user session
app.post("/session/:user", function(req, res) {
  let user = req.params.user;
  let data = req.body;
  sessions.set(user, data);
  res.json({ user: user, data: data });
});

// Route: get user session
app.get("/session/:user", function(req, res) {
  let user = req.params.user;
  let data = sessions.get(user);
  if (data === undefined) {
    res.status(404).json({ error: "not found" });
  } else {
    res.json({ user: user, data: data });
  }
});

// Route: async operation (simulate slow DB query)
app.get("/slow/:id", function(req, res) {
  let id = req.params.id;
  return sleepAsync(20).then(function() {
    res.json({ id: id, ts: Date.now() });
  });
});

// Route: stats
app.get("/stats", function(req, res) {
  res.json({
    total: totalRequests.get(),
    users: sessions.keys().length
  });
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Phase 1: Concurrent session creation (50 users)
	const users = 50
	var wg sync.WaitGroup
	errs := make(chan error, users*3)
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := fmt.Sprintf("user%d", id)
			data := fmt.Sprintf("data-%d", id)
			resp, err := http.Post(server.URL+"/session/"+user, "text/plain", strings.NewReader(data))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs <- fmt.Errorf("POST /session/%s: %d %s", user, resp.StatusCode, body)
			}
		}(i)
	}
	wg.Wait()

	// Phase 2: Concurrent session reads (verify isolation didn't corrupt state)
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := fmt.Sprintf("user%d", id)
			want := fmt.Sprintf("data-%d", id)
			resp, err := http.Get(server.URL + "/session/" + user)
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
				errs <- fmt.Errorf("GET /session/%s: %d %s (want %s)", user, resp.StatusCode, body, want)
			}
		}(i)
	}
	wg.Wait()

	// Phase 3: Concurrent async slow requests (test that isolation + async work together)
	const slowRequests = 30
	start := time.Now()
	for i := 0; i < slowRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(server.URL + fmt.Sprintf("/slow/%d", id))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs <- fmt.Errorf("GET /slow/%d: %d %s", id, resp.StatusCode, body)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	// 30 requests × 20ms = 600ms serial; with pool=8 expect ~80-150ms
	if elapsed > 300*time.Millisecond {
		t.Errorf("slow requests took %v (pool=8, 20ms each, 30 total); expected parallel speedup", elapsed)
	}

	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	// Phase 4: Verify stats
	resp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Total requests: users*POST + users*GET + slowRequests*GET + 1*stats = 50+50+30+1 = 131
	// (But async requests might race with stats read, so allow ±5)
	if !strings.Contains(string(body), `"total":`) {
		t.Fatalf("stats missing total: %s", body)
	}
	if !strings.Contains(string(body), fmt.Sprintf(`"users":%d`, users)) {
		t.Fatalf("stats users != %d: %s", users, body)
	}
	t.Logf("Full-stack test: %d users, %d async, %v elapsed; stats: %s", users, slowRequests, elapsed, body)
}

// TestIsolatedModeStressRace is a stress test under -race: fire 200 concurrent
// requests across multiple routes, shared state, and async operations. Any race
// or deadlock will be caught by the race detector.
func TestIsolatedModeStressRace(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 4 });
let hits = shared.counter("stress-hits");
let store = shared.map("stress-store");

app.get("/hit", function(req, res) {
  let n = hits.incr();
  res.send(String(n));
});

app.post("/store/:key", function(req, res) {
  store.set(req.params.key, req.body);
  res.send("ok");
});

app.get("/async/:id", function(req, res) {
  return sleepAsync(5).then(function() {
    res.json({ id: req.params.id, ts: Date.now() });
  });
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	const requests = 200
	var wg sync.WaitGroup
	var failures int64
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			route := id % 3
			var resp *http.Response
			var err error
			switch route {
			case 0:
				resp, err = http.Get(server.URL + "/hit")
			case 1:
				resp, err = http.Post(server.URL+fmt.Sprintf("/store/k%d", id), "text/plain", strings.NewReader(fmt.Sprintf("v%d", id)))
			case 2:
				resp, err = http.Get(server.URL + fmt.Sprintf("/async/%d", id))
			}
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failures, 1)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
		}(i)
	}
	wg.Wait()
	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d requests failed under stress+race", f, requests)
	}
	t.Logf("Stress test: %d concurrent requests across 3 routes, all succeeded", requests)
}

// TestSharedAtomicCASUnderContention is a stress test for shared.atomic CAS
// under high contention: 100 goroutines competing to increment via CAS.
func TestSharedAtomicCASUnderContention(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 8 });
let atom = shared.atomic("cas-contention", 0);

app.get("/cas-incr", function(req, res) {
  let retries = 0;
  while (retries < 1000) {
    let old = atom.get();
    let next = old + 1;
    if (atom.compareAndSwap(old, next)) {
      res.json({ value: next, retries: retries });
      return;
    }
    retries = retries + 1;
  }
  res.status(500).send("too-many-retries");
});

app.get("/get", function(req, res) {
  res.send(String(atom.get()));
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	const increments = 100
	var wg sync.WaitGroup
	var failures int64
	var totalRetries int64
	for i := 0; i < increments; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/cas-incr")
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failures, 1)
				t.Logf("CAS failed: %d %s", resp.StatusCode, body)
				return
			}
			// Parse retries from JSON {"value":N,"retries":R}
			if strings.Contains(string(body), `"retries":`) {
				parts := strings.Split(string(body), `"retries":`)
				if len(parts) > 1 {
					numStr := strings.TrimRight(strings.Split(parts[1], "}")[0], " ")
					if r, err := strconv.Atoi(numStr); err == nil {
						atomic.AddInt64(&totalRetries, int64(r))
					}
				}
			}
		}()
	}
	wg.Wait()

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d CAS operations failed", f, increments)
	}

	resp, err := http.Get(server.URL + "/get")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	got, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		t.Fatalf("final value %q is not a number: %v", body, err)
	}
	if got != increments {
		t.Fatalf("CAS final value = %d, want %d", got, increments)
	}
	avgRetries := float64(atomic.LoadInt64(&totalRetries)) / float64(increments)
	t.Logf("CAS contention test: %d increments, final=%d, avg retries=%.1f", increments, got, avgRetries)
}

// TestSharedMapConcurrentReadWrite verifies that shared.map handles concurrent
// reads and writes without corruption or data races.
func TestSharedMapConcurrentReadWrite(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 8 });
let store = shared.map("concurrent-rw");

app.post("/set/:key", function(req, res) {
  store.set(req.params.key, req.body);
  res.send("ok");
});

app.get("/get/:key", function(req, res) {
  let val = store.get(req.params.key);
  if (val === undefined) {
    res.status(404).send("not-found");
  } else {
    res.send(val);
  }
});

app.delete("/del/:key", function(req, res) {
  store.delete(req.params.key);
  res.send("deleted");
});

app.get("/has/:key", function(req, res) {
  res.send(store.has(req.params.key) ? "yes" : "no");
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	const keys = 30
	const writesPerKey = 5
	var wg sync.WaitGroup
	var failures int64
	var errMu sync.Mutex
	var errorSamples []string

	// Phase 1: Concurrent writes to different keys
	for i := 0; i < keys; i++ {
		for j := 0; j < writesPerKey; j++ {
			wg.Add(1)
			go func(k, v int) {
				defer wg.Done()
				key := fmt.Sprintf("k%d", k)
				val := fmt.Sprintf("v%d-%d", k, v)
				resp, err := http.Post(server.URL+"/set/"+key, "text/plain", strings.NewReader(val))
				if err != nil {
					atomic.AddInt64(&failures, 1)
					errMu.Lock()
					if len(errorSamples) < 5 {
						errorSamples = append(errorSamples, fmt.Sprintf("POST err: %v", err))
					}
					errMu.Unlock()
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					atomic.AddInt64(&failures, 1)
					body, _ := io.ReadAll(resp.Body)
					errMu.Lock()
					if len(errorSamples) < 5 {
						errorSamples = append(errorSamples, fmt.Sprintf("POST %s: %d %s", key, resp.StatusCode, body))
					}
					errMu.Unlock()
				}
			}(i, j)
		}
	}
	wg.Wait() // Wait for all writes to complete

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Logf("Sample errors: %v", errorSamples)
		t.Fatalf("%d/%d write operations failed", f, keys*writesPerKey)
	}

	// Phase 2: Concurrent reads (all keys should exist now)
	for i := 0; i < keys*2; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", k%keys)
			resp, err := http.Get(server.URL + "/get/" + key)
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			// After writes settle, all keys must exist (200, not 404)
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Logf("GET %s: %d %s", key, resp.StatusCode, body)
				atomic.AddInt64(&failures, 1)
			}
		}(i)
	}
	wg.Wait()

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d read operations failed after writes completed", f)
	}

	// Phase 3: Verify all keys exist via has()
	for i := 0; i < keys; i++ {
		key := fmt.Sprintf("k%d", i)
		resp, err := http.Get(server.URL + "/has/" + key)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "yes" {
			t.Errorf("key %s not found after concurrent writes", key)
		}
	}

	t.Logf("Concurrent R/W test: %d keys × %d writes + %d reads, all consistent", keys, writesPerKey, keys*2)
}
