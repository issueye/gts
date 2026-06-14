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

// TestConcurrencyLimitEnforcement verifies that poolSize strictly limits
// concurrent request execution. With poolSize=4 and handlers that block for
// 100ms, we fire 20 requests and assert:
// 1. Max 4 requests execute concurrently (tracked via atomic counter)
// 2. Total time ≈ (20 / 4) * 100ms = 500ms (not 20*100ms = 2000ms)
func TestConcurrencyLimitEnforcement(t *testing.T) {
	const (
		poolSize      = 4
		totalRequests = 20
		blockMs       = 100
		// Expected time: ceil(20/4) * 100ms = 5 waves * 100ms = 500ms
		// Allow overhead: cap at 700ms (well under serial 2000ms)
		maxExpectedMs = 700
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

// Track concurrent execution count using counter
let concurrent = shared.counter("concurrent");
let maxConcurrent = shared.atomic("max-concurrent", 0);

app.get("/block", function(req, res) {
  // Atomically increment concurrent counter
  let current = concurrent.incr();

  // Update max if needed (CAS loop)
  let retries = 0;
  while (retries < 100) {
    let oldMax = maxConcurrent.get();
    if (current <= oldMax) {
      break;
    }
    if (maxConcurrent.compareAndSwap(oldMax, current)) {
      break;
    }
    retries = retries + 1;
  }

  sleep(%d);

  // Decrement before responding
  concurrent.decr();
  res.send("ok");
});

app.get("/stats", function(req, res) {
  res.json({
    current: concurrent.get(),
    max: maxConcurrent.get()
  });
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64

	// Fire all requests concurrently
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/block")
			if err != nil {
				atomic.AddInt64(&failures, 1)
				t.Logf("Request %d failed: %v", id, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				atomic.AddInt64(&failures, 1)
				t.Logf("Request %d status %d: %s", id, resp.StatusCode, body)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d requests failed", f, totalRequests)
	}

	// Check total time
	if elapsed > time.Duration(maxExpectedMs)*time.Millisecond {
		t.Errorf("Total time %v exceeds expected max %vms (poolSize=%d should process %d reqs in ~5 waves)",
			elapsed, maxExpectedMs, poolSize, totalRequests)
	}

	// Verify max concurrent never exceeded poolSize
	resp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Parse max from JSON response like {"current":0,"max":4}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"max":`) {
		t.Fatalf("stats response missing max field: %s", bodyStr)
	}

	// Extract max value
	parts := strings.Split(bodyStr, `"max":`)
	if len(parts) < 2 {
		t.Fatalf("cannot parse max from stats: %s", bodyStr)
	}
	maxStr := strings.TrimSpace(strings.Split(parts[1], "}")[0])
	maxConcurrent, err := strconv.Atoi(maxStr)
	if err != nil {
		t.Fatalf("max value %q is not a number: %v", maxStr, err)
	}

	if maxConcurrent > poolSize {
		t.Errorf("Max concurrent = %d, exceeds poolSize = %d (isolation broken!)",
			maxConcurrent, poolSize)
	}

	t.Logf("✓ Concurrency limit enforced: %d requests in %v, max concurrent = %d (limit = %d)",
		totalRequests, elapsed, maxConcurrent, poolSize)
}

// TestPoolSizeScaling verifies that increasing poolSize increases throughput.
// We test poolSize 1, 2, 4, 8 with 16 requests of 50ms each and assert:
// - poolSize=1: ~800ms (16 * 50ms)
// - poolSize=2: ~400ms (8 waves)
// - poolSize=4: ~200ms (4 waves)
// - poolSize=8: ~100ms (2 waves)
func TestPoolSizeScaling(t *testing.T) {
	const (
		totalRequests = 16
		blockMs       = 50
	)

	testCases := []struct {
		poolSize       int
		expectedWaves  int
		maxTimeMs      int
	}{
		{poolSize: 1, expectedWaves: 16, maxTimeMs: 900},  // 16 * 50ms = 800ms + overhead
		{poolSize: 2, expectedWaves: 8, maxTimeMs: 500},   // 8 * 50ms = 400ms + overhead
		{poolSize: 4, expectedWaves: 4, maxTimeMs: 300},   // 4 * 50ms = 200ms + overhead
		{poolSize: 8, expectedWaves: 2, maxTimeMs: 200},   // 2 * 50ms = 100ms + overhead
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("poolSize=%d", tc.poolSize), func(t *testing.T) {
			src := fmt.Sprintf(`
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });
app.get("/work", function(req, res) {
  sleep(%d);
  res.send("done");
});
let server = app.listen(0);
app;
`, tc.poolSize, blockMs)

			app := evalWebIsolatedApp(t, src)
			server := httptest.NewServer(app)
			defer server.Close()

			start := time.Now()
			var wg sync.WaitGroup
			var failures int64

			for i := 0; i < totalRequests; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					resp, err := http.Get(server.URL + "/work")
					if err != nil {
						atomic.AddInt64(&failures, 1)
						return
					}
					defer resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						atomic.AddInt64(&failures, 1)
					}
				}()
			}
			wg.Wait()
			elapsed := time.Since(start)

			if f := atomic.LoadInt64(&failures); f > 0 {
				t.Fatalf("%d/%d requests failed", f, totalRequests)
			}

			if elapsed > time.Duration(tc.maxTimeMs)*time.Millisecond {
				t.Errorf("poolSize=%d: %d requests took %v, expected < %vms (%d waves × %dms)",
					tc.poolSize, totalRequests, elapsed, tc.maxTimeMs, tc.expectedWaves, blockMs)
			}

			t.Logf("poolSize=%d: %d requests in %v (expected ~%d waves × %dms = ~%dms)",
				tc.poolSize, totalRequests, elapsed, tc.expectedWaves, blockMs, tc.expectedWaves*blockMs)
		})
	}
}

// TestExtremeParallelism tests with a very large pool (64) and many concurrent
// requests (200) to verify the system remains stable under high parallelism.
func TestExtremeParallelism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extreme parallelism test in short mode")
	}

	const (
		poolSize      = 64
		totalRequests = 200
		blockMs       = 30
		// With 64 concurrent, 200 requests = 4 waves max
		// 4 * 30ms = 120ms ideal, allow 300ms for overhead
		maxTimeMs = 300
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

let counter = shared.counter("req-counter");

app.get("/work/:id", function(req, res) {
  let n = counter.incr();
  sleep(%d);
  res.json({ id: req.params.id, count: n });
});

app.get("/stats", function(req, res) {
  res.json({ total: counter.get() });
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64
	var successCount int64

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(server.URL + fmt.Sprintf("/work/%d", id))
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&failures, 1)
			}
			_, _ = io.Copy(io.Discard, resp.Body)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Errorf("%d/%d requests failed under extreme parallelism", f, totalRequests)
	}

	success := atomic.LoadInt64(&successCount)
	if success != totalRequests {
		t.Errorf("Expected %d successful requests, got %d", totalRequests, success)
	}

	if elapsed > time.Duration(maxTimeMs)*time.Millisecond {
		t.Logf("Warning: %d requests with poolSize=%d took %v (expected < %vms)",
			totalRequests, poolSize, elapsed, maxTimeMs)
	}

	// Verify total counter
	resp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	bodyStr := string(body)
	if !strings.Contains(bodyStr, fmt.Sprintf(`"total":%d`, totalRequests)) {
		t.Errorf("Expected total=%d in stats, got: %s", totalRequests, bodyStr)
	}

	t.Logf("✓ Extreme parallelism: poolSize=%d, %d requests in %v, all succeeded",
		poolSize, totalRequests, elapsed)
}

// TestPoolStarvation verifies behavior when requests arrive faster than the pool
// can handle them. We use poolSize=2 with slow handlers (200ms) and fire 10
// requests rapidly. The first 2 should start immediately, the rest should queue
// and wait for slots.
func TestPoolStarvation(t *testing.T) {
	const (
		poolSize      = 2
		totalRequests = 10
		blockMs       = 200
		// Expected: 5 waves × 200ms = 1000ms
		minTimeMs = 900  // Should take at least this long
		maxTimeMs = 1300 // Upper bound with overhead
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

let startTimes = shared.map("start-times");

app.get("/slow/:id", function(req, res) {
  let id = req.params.id;
  let now = Date.now();
  startTimes.set(id, String(now));
  sleep(%d);
  res.send("ok");
});

app.get("/starts", function(req, res) {
  let keys = startTimes.keys();
  res.json({ count: keys.length });
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := http.Get(server.URL + fmt.Sprintf("/slow/%d", id))
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failures, 1)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d requests failed", f, totalRequests)
	}

	elapsedMs := elapsed.Milliseconds()
	if elapsedMs < minTimeMs {
		t.Errorf("Completed too fast: %v (poolSize=%d should serialize to ~%dms minimum)",
			elapsed, poolSize, minTimeMs)
	}
	if elapsedMs > maxTimeMs {
		t.Errorf("Took too long: %v (expected ~%dms with poolSize=%d)",
			elapsed, totalRequests/poolSize*blockMs, poolSize)
	}

	t.Logf("✓ Pool starvation handled: poolSize=%d, %d requests in %v (expected ~%d waves)",
		poolSize, totalRequests, elapsed, totalRequests/poolSize)
}

// TestMixedRequestDurations tests with varying request durations to verify
// the pool efficiently schedules different workloads.
func TestMixedRequestDurations(t *testing.T) {
	const (
		poolSize = 4
		fastMs   = 10
		slowMs   = 100
		fastReqs = 20
		slowReqs = 8
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

app.get("/fast", function(req, res) {
  sleep(%d);
  res.send("fast");
});

app.get("/slow", function(req, res) {
  sleep(%d);
  res.send("slow");
});

let server = app.listen(0);
app;
`, poolSize, fastMs, slowMs)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64

	// Fire fast requests
	for i := 0; i < fastReqs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/fast")
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failures, 1)
			}
		}()
	}

	// Fire slow requests
	for i := 0; i < slowReqs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL + "/slow")
			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt64(&failures, 1)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d requests failed", f, fastReqs+slowReqs)
	}

	// With poolSize=4, slow requests would dominate if scheduled poorly
	// Ideally: fast requests slip through quickly while slow ones run
	// Max time should be dominated by slow request waves: ceil(8/4) * 100ms = 200ms
	// Allow overhead: 400ms
	maxTimeMs := 400
	if elapsed > time.Duration(maxTimeMs)*time.Millisecond {
		t.Errorf("Mixed workload took %v, expected < %vms with poolSize=%d",
			elapsed, maxTimeMs, poolSize)
	}

	t.Logf("✓ Mixed durations: %d fast (%dms) + %d slow (%dms) in %v with poolSize=%d",
		fastReqs, fastMs, slowReqs, slowMs, elapsed, poolSize)
}
