package parser

import (
	"testing"
	"time"
)

// TestParse_ReservedKeywordAsObjectKey is a regression test for an infinite loop
// in the parser. Previously, when an object literal used a reserved keyword
// (e.g. `default`) as a property key, parseProperty hit its default switch arm,
// emitted an error, returned nil WITHOUT advancing the current token, and left
// the caller's loop stuck on the same keyword token forever.
//
// Example that used to hang:
//
//	const cfg = {
//	  "a/b": { x: 1 },
//	  default: { x: 2 },
//	};
func TestParse_ReservedKeywordAsObjectKey(t *testing.T) {
	cases := []string{
		`const a = { default: 1 };`,
		`const b = { "a/b": { x: 1 }, default: { x: 2 } };`,
		`const c = { if: 1, else: 2, return: 3 };`,
		`const d = { class: 1, new: 2, this: 3 };`,
		`const e = { true: 1, false: 2, null: 3, undefined: 4 };`,
	}
	for _, src := range cases {
		done := make(chan struct{})
		var errs []string
		go func(s string) {
			defer close(done)
			p := Parse(s)
			errs = p.Errors
		}(src)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("parser hung (infinite loop) on: %s", src)
		}
		if len(errs) > 0 {
			t.Fatalf("unexpected parse errors for %q: %v", src, errs)
		}
	}
}
