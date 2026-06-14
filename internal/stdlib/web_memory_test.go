package stdlib

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/lexer"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/parser"
)

// memStats captures heap statistics at a specific point
type memStats struct {
	Alloc      uint64 // bytes allocated and not yet freed
	TotalAlloc uint64 // bytes allocated (cumulative)
	Sys        uint64 // bytes obtained from system
	NumGC      uint32 // number of GC runs
	HeapAlloc  uint64 // bytes allocated on heap
	HeapSys    uint64 // bytes obtained from system for heap
	HeapInuse  uint64 // bytes in in-use spans
	HeapIdle   uint64 // bytes in idle spans
}

func getMemStats() memStats {
	runtime.GC() // Force GC to get accurate measurement
	time.Sleep(10 * time.Millisecond)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return memStats{
		Alloc:      m.Alloc,
		TotalAlloc: m.TotalAlloc,
		Sys:        m.Sys,
		NumGC:      m.NumGC,
		HeapAlloc:  m.HeapAlloc,
		HeapSys:    m.HeapSys,
		HeapInuse:  m.HeapInuse,
		HeapIdle:   m.HeapIdle,
	}
}

func (m memStats) String() string {
	return fmt.Sprintf("Alloc=%s HeapInuse=%s HeapSys=%s Sys=%s",
		formatBytes(m.Alloc),
		formatBytes(m.HeapInuse),
		formatBytes(m.HeapSys),
		formatBytes(m.Sys))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func diffMemStats(before, after memStats) memStats {
	// Handle potential underflow when memory decreases (GC)
	safeDiff := func(a, b uint64) uint64 {
		if a > b {
			return a - b
		}
		return 0
	}
	return memStats{
		Alloc:      safeDiff(after.Alloc, before.Alloc),
		TotalAlloc: safeDiff(after.TotalAlloc, before.TotalAlloc),
		HeapAlloc:  safeDiff(after.HeapAlloc, before.HeapAlloc),
		HeapInuse:  safeDiff(after.HeapInuse, before.HeapInuse),
		HeapSys:    safeDiff(after.HeapSys, before.HeapSys),
		Sys:        safeDiff(after.Sys, before.Sys),
	}
}

// TestSingleVMMemoryFootprint measures the memory cost of a single VM
func TestSingleVMMemoryFootprint(t *testing.T) {
	// Baseline: measure before creating VM
	runtime.GC()
	time.Sleep(20 * time.Millisecond)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	t.Logf("Baseline: Alloc=%s HeapInuse=%s Sys=%s",
		formatBytes(before.Alloc), formatBytes(before.HeapInuse), formatBytes(before.Sys))

	// Create a single VM with minimal script
	vm := object.NewVirtualMachine()
	env := vm.NewEnvironment()
	module.SetupExports(env)
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})

	runtime.ReadMemStats(&after)
	allocDiff := after.TotalAlloc - before.TotalAlloc
	heapDiff := int64(after.HeapInuse) - int64(before.HeapInuse)
	t.Logf("After creating empty VM: TotalAlloc diff=%s HeapInuse diff=%+d bytes",
		formatBytes(allocDiff), heapDiff)

	// Now evaluate a simple script
	src := `
let x = 42;
let y = "hello";
let z = [1, 2, 3];
let obj = { a: 1, b: 2, c: 3 };
"done";
`
	var beforeEval runtime.MemStats
	runtime.ReadMemStats(&beforeEval)

	l := lexer.New(src)
	p := parser.New(l, "test.gs")
	program := p.ParseProgram()
	evaluator.Eval(program, env)

	var afterEval runtime.MemStats
	runtime.ReadMemStats(&afterEval)
	evalAlloc := afterEval.TotalAlloc - beforeEval.TotalAlloc
	t.Logf("After eval simple script: allocated %s", formatBytes(evalAlloc))

	totalAlloc := afterEval.TotalAlloc - before.TotalAlloc
	t.Logf("\n=== Single VM Total Memory Footprint ===")
	t.Logf("Empty VM creation: ~%s", formatBytes(allocDiff))
	t.Logf("Script evaluation:  %s", formatBytes(evalAlloc))
	t.Logf("Total allocated:    %s", formatBytes(totalAlloc))
	t.Logf("System memory:      %s", formatBytes(after.Sys-before.Sys))
}

// TestIsolatedSessionMemoryFootprint measures memory of a complete isolated session
func TestIsolatedSessionMemoryFootprint(t *testing.T) {
	src := `
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: 1 });
app.get("/test", function(req, res) {
  res.send("ok");
});
let server = app.listen(0);
app;
`
	before := getMemStats()
	t.Logf("Baseline: %s", before)

	// Create main VM and parse
	vm := object.NewVirtualMachine()
	vm.SetBootstrapSource(src)
	env := vm.NewEnvironment()
	module.SetupExports(env)
	evaluator.RegisterBuiltinsWithCache(env, func(path string) (object.Object, error) {
		if native, ok := module.GetNative(path, env); ok {
			return native, nil
		}
		return nil, nil
	})

	l := lexer.New(src)
	p := parser.New(l, "test.gs")
	program := p.ParseProgram()
	evaluator.Eval(program, env)

	afterMain := getMemStats()
	mainDiff := diffMemStats(before, afterMain)
	t.Logf("After main VM: %s", afterMain)
	t.Logf("Main VM cost: %s", mainDiff)

	// Now create one isolated session (simulates one request VM)
	app, _ := lookupIsolatedApp(vm)
	if app == nil {
		t.Fatal("no app found")
	}

	// Initialize isolated pool (triggers session creation)
	if err := app.initIsolated(env); err != nil {
		t.Fatal(err)
	}

	afterSession := getMemStats()
	sessionDiff := diffMemStats(afterMain, afterSession)
	t.Logf("After 1 isolated session: %s", afterSession)
	t.Logf("Session cost: %s", sessionDiff)

	totalDiff := diffMemStats(before, afterSession)
	t.Logf("\n=== Isolated Session Memory Footprint ===")
	t.Logf("Main VM:        %s", formatBytes(mainDiff.HeapAlloc))
	t.Logf("1 Session:      %s", formatBytes(sessionDiff.HeapAlloc))
	t.Logf("Total:          %s", formatBytes(totalDiff.HeapAlloc))
}

// TestPoolMemoryScaling measures memory usage with different pool sizes
func TestPoolMemoryScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory scaling test in short mode")
	}

	src := `
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });
app.get("/test", function(req, res) {
  let data = { id: req.params.id, ts: Date.now() };
  res.json(data);
});
let server = app.listen(0);
app;
`

	poolSizes := []int{1, 2, 4, 8, 16, 32, 64}
	results := make(map[int]memStats)

	for _, poolSize := range poolSizes {
		// Force GC before each measurement
		runtime.GC()
		time.Sleep(50 * time.Millisecond)

		before := getMemStats()

		script := fmt.Sprintf(src, poolSize)
		app := evalWebIsolatedApp(t, script)

		// Warm the pool
		server := httptest.NewServer(app)
		for i := 0; i < poolSize; i++ {
			go http.Get(server.URL + "/test")
		}
		time.Sleep(100 * time.Millisecond)
		server.Close()

		after := getMemStats()
		diff := diffMemStats(before, after)
		results[poolSize] = diff

		t.Logf("poolSize=%d: %s (HeapAlloc=%s)",
			poolSize, diff, formatBytes(diff.HeapAlloc))
	}

	// Calculate per-session cost
	t.Logf("\n=== Memory Scaling Analysis ===")
	if len(results) >= 2 {
		pool1 := results[1].HeapAlloc
		pool2 := results[2].HeapAlloc
		perSessionCost := pool2 - pool1
		t.Logf("Estimated per-session cost: %s (pool2 - pool1)", formatBytes(perSessionCost))

		for _, size := range []int{4, 8, 16, 32, 64} {
			if mem, ok := results[size]; ok {
				estimated := pool1 + uint64(size-1)*perSessionCost
				actual := mem.HeapAlloc
				diff := int64(actual) - int64(estimated)
				diffPct := float64(diff) / float64(estimated) * 100
				t.Logf("poolSize=%d: actual=%s estimated=%s diff=%+.1f%%",
					size, formatBytes(actual), formatBytes(estimated), diffPct)
			}
		}
	}

	// Summary
	t.Logf("\n=== Memory Usage Summary ===")
	for _, size := range poolSizes {
		if mem, ok := results[size]; ok {
			perVM := mem.HeapAlloc / uint64(size)
			t.Logf("poolSize=%2d: total=%s (~%s per VM)",
				size, formatBytes(mem.HeapAlloc), formatBytes(perVM))
		}
	}
}

// TestMemoryUnderLoad measures memory during active request processing
func TestMemoryUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory load test in short mode")
	}

	const (
		poolSize = 8
		requests = 100
	)

	src := fmt.Sprintf(`
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: %d });
app.get("/work", function(req, res) {
  let data = [];
  for (let i = 0; i < 100; i = i + 1) {
    data.push({ index: i, value: "item-" + String(i) });
  }
  sleep(10);
  res.json(data);
});
let server = app.listen(0);
app;
`, poolSize)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Measure before load
	before := getMemStats()
	t.Logf("Before load: %s", before)

	// Generate load
	done := make(chan struct{})
	go func() {
		for i := 0; i < requests; i++ {
			go func() {
				http.Get(server.URL + "/work")
			}()
			time.Sleep(5 * time.Millisecond) // Steady stream
		}
		close(done)
	}()

	// Sample memory during load
	var samples []memStats
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			goto DONE
		case <-ticker.C:
			sample := getMemStats()
			samples = append(samples, sample)
			diff := diffMemStats(before, sample)
			t.Logf("During load: %s (+%s from baseline)",
				sample, formatBytes(diff.HeapAlloc))
		}
	}

DONE:
	time.Sleep(200 * time.Millisecond) // Let requests finish

	after := getMemStats()
	diff := diffMemStats(before, after)
	t.Logf("After load: %s", after)
	t.Logf("Peak memory increase: %s", formatBytes(diff.HeapAlloc))

	// Calculate peak
	var peakDiff uint64
	for _, sample := range samples {
		d := diffMemStats(before, sample)
		if d.HeapAlloc > peakDiff {
			peakDiff = d.HeapAlloc
		}
	}
	t.Logf("\n=== Memory Under Load ===")
	t.Logf("Baseline:     %s", formatBytes(before.HeapAlloc))
	t.Logf("Peak during:  +%s", formatBytes(peakDiff))
	t.Logf("After finish: +%s", formatBytes(diff.HeapAlloc))
	t.Logf("Requests:     %d with poolSize=%d", requests, poolSize)
}

// TestVMPoolIdleMemory measures memory of idle vs active sessions
func TestVMPoolIdleMemory(t *testing.T) {
	src := `
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: 4 });
app.get("/test", function(req, res) {
  res.send("ok");
});
let server = app.listen(0);
app;
`

	before := getMemStats()
	t.Logf("Baseline: %s", before)

	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	afterInit := getMemStats()
	initDiff := diffMemStats(before, afterInit)
	t.Logf("After init (idle pool): %s", afterInit)
	t.Logf("Init cost: %s", initDiff)

	// Make 4 requests to warm the pool
	for i := 0; i < 4; i++ {
		http.Get(server.URL + "/test")
	}
	time.Sleep(100 * time.Millisecond)

	afterWarm := getMemStats()
	warmDiff := diffMemStats(afterInit, afterWarm)
	t.Logf("After warming pool: %s", afterWarm)
	t.Logf("Warm cost: %s", warmDiff)

	totalDiff := diffMemStats(before, afterWarm)
	t.Logf("\n=== VM Pool Memory Breakdown ===")
	t.Logf("Initial setup: %s", formatBytes(initDiff.HeapAlloc))
	t.Logf("Pool warm (4): %s", formatBytes(warmDiff.HeapAlloc))
	t.Logf("Total:         %s", formatBytes(totalDiff.HeapAlloc))
	t.Logf("Per session:   ~%s", formatBytes(warmDiff.HeapAlloc/4))
}
