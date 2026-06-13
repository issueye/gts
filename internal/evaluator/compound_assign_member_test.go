package evaluator

import (
	"testing"

	"github.com/issueye/goscript/internal/object"
)

// TestRepro_CompoundAssignMember is a regression test for compound assignment
// operators applied to object members and array elements. Previously
// `obj.x += 1` overwrote obj.x with the bare right-hand value (1) because the
// MemberExpr/IndexExpr branches of evalAssign ignored the operator and never
// read the existing value. Only identifier targets worked.
func TestRepro_CompoundAssignMember(t *testing.T) {
	cases := []struct {
		src  string
		want float64
	}{
		{`let o = {x: 1}; o.x += 5; o.x`, 6},
		{`let o = {x: 10}; o.x -= 3; o.x`, 7},
		{`let o = {x: 2}; o.x *= 4; o.x`, 8},
		{`let o = {x: 12}; o.x /= 3; o.x`, 4},
		{`let c = {n: 0}; c.n += 1; c.n += 1; c.n += 1; c.n`, 3},
	}
	for _, c := range cases {
		got := testEval(c.src)
		n, ok := got.(*object.Number)
		if !ok {
			t.Fatalf("expected number, got %T (%s) for %s", got, got.Inspect(), c.src)
		}
		if n.Value != c.want {
			t.Fatalf("for %s: want %v, got %v", c.src, c.want, n.Value)
		}
	}

	// String concatenation via += on a member
	got := testEval(`let o = {s: "ab"}; o.s += "cd"; o.s`)
	s, ok := got.(*object.String)
	if !ok || s.Value != "abcd" {
		t.Fatalf("string += on member: want abcd, got %v (%s)", got, got.Inspect())
	}

	// Array element compound assignment
	got = testEval(`let a = [1, 2, 3]; a[1] += 10; a[1]`)
	n, ok := got.(*object.Number)
	if !ok || n.Value != 12 {
		t.Fatalf("arr[i] += : want 12, got %v (%s)", got, got.Inspect())
	}
}
