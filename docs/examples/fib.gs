// fib.gs —— 闭包、循环、模板字符串
function fib(n: int): int {
  if (n < 2) return n;
  let a: int = 0, b: int = 1;
  for (let i: int = 2; i <= n; i++) {
    let t: int = a + b;
    a = b;
    b = t;
  }
  return b;
}

for (let i: int = 0; i < 10; i++) {
  console.log(`fib(${i}) = ${fib(i)}`);
}
