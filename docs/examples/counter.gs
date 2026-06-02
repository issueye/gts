// counter.gs —— 闭包与状态封装
function makeCounter(start: int = 0) {
  let n: int = start;
  return {
    inc(): int { n = n + 1; return n; },
    dec(): int { n = n - 1; return n; },
    value(): int { return n; },
  };
}

let c = makeCounter(10);
c.inc(); c.inc(); c.dec();
console.log("c.value =", c.value()); // 11
