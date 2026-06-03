package lexer

import (
	"strings"
	"testing"
)

func benchmarkSource() string {
	const unit = `
function fib(n) {
  if (n <= 1) {
    return n;
  }
  return fib(n - 1) + fib(n - 2);
}

let total = 0;
for (let i = 0; i < 20; i = i + 1) {
  total = total + Math.abs(-i) + fib(5);
}

let values = [1, 2, 3, 4, 5].map(function(v) { return v * 2; });
let record = { total: total, values: values, label: "bench" };
record.total + values.length;
`
	return strings.Repeat(unit, 20)
}

func BenchmarkLexerTokenizeScript(b *testing.B) {
	src := benchmarkSource()

	b.ReportAllocs()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := New(src)
		for {
			tok := l.NextToken()
			if tok.Type == TOKEN_EOF {
				break
			}
		}
		if errs := l.Errors(); len(errs) > 0 {
			b.Fatalf("lexer errors: %v", errs)
		}
	}
}
