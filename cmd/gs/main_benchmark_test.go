package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkRunnerEvalFileArithmetic(b *testing.B) {
	dir := b.TempDir()
	script := filepath.Join(dir, "app.gs")
	writeBenchmarkFile(b, script, `
let total = 0;
for (let i = 0; i < 500; i = i + 1) {
  total = total + i;
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newRunner(options{workers: 1, timeout: time.Second})
		if _, err := r.evalFile(script, runOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunnerEvalFileWithModules(b *testing.B) {
	dir := b.TempDir()
	writeBenchmarkFile(b, filepath.Join(dir, "lib.gs"), `
exports.value = 0;
for (let i = 0; i < 50; i = i + 1) {
  exports.value = exports.value + i;
}
`)
	script := filepath.Join(dir, "app.gs")
	writeBenchmarkFile(b, script, `
let lib = require("./lib");
let total = 0;
for (let i = 0; i < 100; i = i + 1) {
  total = total + lib.value;
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := newRunner(options{workers: 1, timeout: time.Second})
		if _, err := r.evalFile(script, runOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunCommandFile(b *testing.B) {
	dir := b.TempDir()
	script := filepath.Join(dir, "app.gs")
	writeBenchmarkFile(b, script, `
let total = 0;
for (let i = 0; i < 100; i = i + 1) {
  total = total + Math.abs(-i);
}
total;
`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if code := run([]string{"--timeout", "1s", script}); code != 0 {
			b.Fatalf("run returned exit code %d", code)
		}
	}
}

func writeBenchmarkFile(b *testing.B, path, contents string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		b.Fatal(err)
	}
}
