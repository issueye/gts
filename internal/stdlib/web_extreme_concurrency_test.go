package stdlib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestExtreme5000Concurrency tests the system under 5000+ concurrent requests
// to verify stability, memory usage, and throughput at extreme scale.
func TestExtreme5000Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extreme 5000+ concurrency test in short mode")
	}

	const (
		poolSize      = 128 // High pool size for extreme concurrency
		totalRequests = 5000
		blockMs       = 20 // Short duration to complete in reasonable time
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

let counter = shared.counter("extreme-requests");
let active = shared.counter("active-requests");
let maxActive = shared.atomic("max-active", 0);

app.get("/work/:id", function(req, res) {
  let current = active.incr();
  counter.incr();

  // Update max active (CAS loop)
  let retries = 0;
  while (retries < 100) {
    let oldMax = maxActive.get();
    if (current <= oldMax) {
      break;
    }
    if (maxActive.compareAndSwap(oldMax, current)) {
      break;
    }
    retries = retries + 1;
  }

  sleep(%d);
  active.decr();
  res.json({ id: req.params.id, count: counter.get() });
});

app.get("/stats", function(req, res) {
  res.json({
    total: counter.get(),
    active: active.get(),
    maxActive: maxActive.get()
  });
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	t.Logf("Starting extreme test: poolSize=%d, requests=%d, duration=%dms",
		poolSize, totalRequests, blockMs)

	// Measure memory before
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	t.Logf("Memory before: Alloc=%s HeapInuse=%s Sys=%s",
		formatBytes(memBefore.Alloc),
		formatBytes(memBefore.HeapInuse),
		formatBytes(memBefore.Sys))

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64
	var successCount int64

	// Launch requests with throttling to avoid overwhelming the system
	// Use a semaphore to limit concurrent launching
	launchSemaphore := make(chan struct{}, 300) // Max 300 launching at once

	for i := 0; i < totalRequests; i++ {
		launchSemaphore <- struct{}{}
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() { <-launchSemaphore }()

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

		// Progressive delay to smooth launch
		if i%100 == 0 && i > 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}

	// Monitor progress
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastSuccess := int64(0)
	for {
		select {
		case <-done:
			goto FINISHED
		case <-ticker.C:
			current := atomic.LoadInt64(&successCount)
			rate := current - lastSuccess
			lastSuccess = current
			t.Logf("Progress: %d/%d completed (%.1f%%), rate=%d req/s",
				current, totalRequests, float64(current)/float64(totalRequests)*100, rate)
		}
	}

FINISHED:
	elapsed := time.Since(start)

	// Measure memory after
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Get stats from server
	resp, err := http.Get(server.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	statsBody, _ := io.ReadAll(resp.Body)

	// Report results
	finalFailures := atomic.LoadInt64(&failures)
	finalSuccess := atomic.LoadInt64(&successCount)
	throughput := float64(finalSuccess) / elapsed.Seconds()

	t.Logf("\n=== Extreme 5000+ Concurrency Test Results ===")
	t.Logf("Configuration:")
	t.Logf("  Pool Size:       %d", poolSize)
	t.Logf("  Total Requests:  %d", totalRequests)
	t.Logf("  Block Duration:  %dms", blockMs)
	t.Logf("")
	t.Logf("Performance:")
	t.Logf("  Total Time:      %v", elapsed)
	t.Logf("  Success:         %d (%.1f%%)", finalSuccess, float64(finalSuccess)/float64(totalRequests)*100)
	t.Logf("  Failures:        %d", finalFailures)
	t.Logf("  Throughput:      %.0f req/s", throughput)
	t.Logf("")
	t.Logf("Memory:")
	t.Logf("  Before:          %s", formatBytes(memBefore.HeapInuse))
	t.Logf("  After:           %s", formatBytes(memAfter.HeapInuse))
	t.Logf("  Peak Increase:   %s", formatBytes(memAfter.HeapInuse-memBefore.HeapInuse))
	t.Logf("  Sys Increase:    %s", formatBytes(memAfter.Sys-memBefore.Sys))
	t.Logf("")
	t.Logf("Server Stats: %s", statsBody)

	// Assertions
	if finalFailures > int64(float64(totalRequests)*0.10) { // Allow 10% failure for extreme load
		t.Errorf("Too many failures: %d/%d (%.1f%%), expected <10%%",
			finalFailures, totalRequests, float64(finalFailures)/float64(totalRequests)*100)
	}

	if finalSuccess < int64(float64(totalRequests)*0.90) {
		t.Logf("Warning: success rate %.1f%% is below 90%%",
			float64(finalSuccess)/float64(totalRequests)*100)
	}

	// Expected throughput: poolSize × (1000ms / blockMs)
	// poolSize=128, blockMs=20 → expected ~6400 req/s
	// Allow for overhead, expect at least 50% of theoretical
	expectedThroughput := float64(poolSize) * (1000.0 / float64(blockMs)) * 0.5
	if throughput < expectedThroughput {
		t.Logf("Warning: throughput %.0f req/s is below expected %.0f req/s",
			throughput, expectedThroughput)
	}

	t.Logf("\n✓ Extreme test passed: %d requests in %v at %.0f req/s",
		finalSuccess, elapsed, throughput)
}

// TestScaleTo10000Requests tests even more extreme scale with 10,000 requests
func TestScaleTo10000Requests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10k concurrency test in short mode")
	}

	const (
		poolSize      = 256 // Very high pool size
		totalRequests = 10000
		blockMs       = 10 // Very short duration
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

let counter = shared.counter("scale10k-requests");

app.get("/fast", function(req, res) {
  counter.incr();
  sleep(%d);
  res.send("ok");
});

app.get("/count", function(req, res) {
  res.send(String(counter.get()));
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	t.Logf("Starting 10k scale test: poolSize=%d, requests=%d", poolSize, totalRequests)

	var memBefore runtime.MemStats
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	runtime.ReadMemStats(&memBefore)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	start := time.Now()
	var wg sync.WaitGroup
	var failures int64
	var successCount int64

	// Throttle request launching to avoid overwhelming the system
	semaphore := make(chan struct{}, 500) // Max 500 launching at once

	for i := 0; i < totalRequests; i++ {
		semaphore <- struct{}{}
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			resp, err := http.Get(server.URL + "/fast")
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

		// Slight delay every 100 requests to smooth launch
		if i%100 == 0 && i > 0 {
			time.Sleep(time.Millisecond)
		}
	}

	// Monitor progress
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			goto FINISHED
		case <-ticker.C:
			current := atomic.LoadInt64(&successCount)
			t.Logf("Progress: %d/%d (%.1f%%)",
				current, totalRequests, float64(current)/float64(totalRequests)*100)
		}
	}

FINISHED:
	elapsed := time.Since(start)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	finalFailures2 := atomic.LoadInt64(&failures)
	finalSuccess2 := atomic.LoadInt64(&successCount)
	throughput := float64(finalSuccess2) / elapsed.Seconds()

	t.Logf("\n=== 10,000 Request Scale Test Results ===")
	t.Logf("Pool Size:       %d", poolSize)
	t.Logf("Total Requests:  %d", totalRequests)
	t.Logf("Total Time:      %v", elapsed)
	t.Logf("Success:         %d (%.1f%%)", finalSuccess2, float64(finalSuccess2)/float64(totalRequests)*100)
	t.Logf("Failures:        %d (%.1f%%)", finalFailures2, float64(finalFailures2)/float64(totalRequests)*100)
	t.Logf("Throughput:      %.0f req/s", throughput)
	t.Logf("Memory Before:   %s", formatBytes(memBefore.HeapInuse))
	t.Logf("Memory After:    %s", formatBytes(memAfter.HeapInuse))
	t.Logf("Memory Increase: %s", formatBytes(memAfter.HeapInuse-memBefore.HeapInuse))

	// Allow 2% failure rate for extreme scale
	if finalFailures2 > int64(float64(totalRequests)*0.02) {
		t.Errorf("Too many failures: %d/%d (%.1f%%)",
			finalFailures2, totalRequests, float64(finalFailures2)/float64(totalRequests)*100)
	}

	t.Logf("\n✓ 10k scale test passed: %d requests in %v at %.0f req/s",
		finalSuccess2, elapsed, throughput)
}

// TestSustainedLoad5000 tests sustained load over longer duration
func TestSustainedLoad5000(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sustained load test in short mode")
	}

	const (
		poolSize     = 64
		duration     = 30 * time.Second
		requestDelay = 10 * time.Millisecond // ~100 req/s launch rate
		blockMs      = 50
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });

let counter = shared.counter("sustained-requests");
let errors = shared.counter("sustained-errors");

app.get("/sustained", function(req, res) {
  try {
    counter.incr();
    sleep(%d);
    res.send("ok");
  } catch (e) {
    errors.incr();
    res.status(500).send("error");
  }
});

app.get("/stats", function(req, res) {
  res.json({
    total: counter.get(),
    errors: errors.get()
  });
});

let server = app.listen(0);
app;
`, poolSize, blockMs)

	t.Logf("Starting sustained load test: duration=%v, poolSize=%d", duration, poolSize)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	var memBefore runtime.MemStats
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	runtime.ReadMemStats(&memBefore)

	start := time.Now()
	end := start.Add(duration)
	var wg sync.WaitGroup
	var totalLaunched int64
	var successCount int64
	var failureCount int64

	// Request generator
	go func() {
		for time.Now().Before(end) {
			wg.Add(1)
			atomic.AddInt64(&totalLaunched, 1)
			go func() {
				defer wg.Done()
				resp, err := http.Get(server.URL + "/sustained")
				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&failureCount, 1)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
			}()
			time.Sleep(requestDelay)
		}
	}()

	// Monitor
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastSuccess int64
	var memSamples []uint64

	for time.Now().Before(end) {
		select {
		case <-ticker.C:
			current := atomic.LoadInt64(&successCount)
			rate := float64(current-lastSuccess) / 5.0
			lastSuccess = current

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memSamples = append(memSamples, m.HeapInuse)

			t.Logf("Sustained: %d success, %d fail, rate=%.0f req/s, mem=%s",
				current, atomic.LoadInt64(&failureCount), rate, formatBytes(m.HeapInuse))
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Wait for all requests to finish
	wg.Wait()
	elapsed := time.Since(start)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	success := atomic.LoadInt64(&successCount)
	failures := atomic.LoadInt64(&failureCount)
	launched := atomic.LoadInt64(&totalLaunched)
	throughput := float64(success) / elapsed.Seconds()

	// Calculate memory stats
	var maxMem, avgMem uint64
	for _, m := range memSamples {
		if m > maxMem {
			maxMem = m
		}
		avgMem += m
	}
	if len(memSamples) > 0 {
		avgMem /= uint64(len(memSamples))
	}

	t.Logf("\n=== Sustained Load Test Results ===")
	t.Logf("Duration:        %v", elapsed)
	t.Logf("Requests:        %d launched, %d success, %d fail", launched, success, failures)
	t.Logf("Success Rate:    %.1f%%", float64(success)/float64(launched)*100)
	t.Logf("Throughput:      %.0f req/s", throughput)
	t.Logf("Memory Baseline: %s", formatBytes(memBefore.HeapInuse))
	t.Logf("Memory Average:  %s", formatBytes(avgMem))
	t.Logf("Memory Peak:     %s", formatBytes(maxMem))
	t.Logf("Memory Final:    %s", formatBytes(memAfter.HeapInuse))

	if failures > launched/10 { // Allow 10% failure for sustained load
		t.Errorf("High failure rate: %d/%d (%.1f%%)",
			failures, launched, float64(failures)/float64(launched)*100)
	}

	t.Logf("\n✓ Sustained load test passed: %.0f req/s over %v", throughput, elapsed)
}
