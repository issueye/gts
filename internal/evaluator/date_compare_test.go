package evaluator

import (
	"testing"

	"github.com/issueye/goscript/internal/object"
)

// TestRepro_DateComparison is a regression test for relational comparison
// of Date values. Standard JS converts Date to its epoch milliseconds via
// valueOf before comparing, so `date1 < date2` must be legal. Previously
// evalCompare only handled Number/String pairs and threw
// "cannot compare DATE and DATE".
func TestRepro_DateComparison(t *testing.T) {
	cases := []struct {
		src string
		want bool
	}{
		{`let a = new Date("2020-01-01T00:00:00Z"); let b = new Date("2025-01-01T00:00:00Z"); a < b`, true},
		{`let a = new Date("2020-01-01T00:00:00Z"); let b = new Date("2025-01-01T00:00:00Z"); b > a`, true},
		{`let a = new Date("2020-01-01T00:00:00Z"); let b = new Date("2020-01-01T00:00:00Z"); a <= b`, true},
		{`let a = new Date("2025-01-01T00:00:00Z"); let b = new Date("2020-01-01T00:00:00Z"); a >= b`, true},
		{`let a = new Date("2020-01-01T00:00:00Z"); let b = new Date("2025-01-01T00:00:00Z"); a > b`, false},
	}
	for _, c := range cases {
		got := testEval(c.src)
		b, ok := got.(*object.Boolean)
		if !ok {
			t.Fatalf("expected boolean, got %T (%s) for %s", got, got.Inspect(), c.src)
		}
		if b.Value != c.want {
			t.Fatalf("for %s: want %v, got %v", c.src, c.want, b.Value)
		}
	}
}
