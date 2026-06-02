// ============================================================
// 05-closures.gs —— 第五步：闭包与高阶函数
// ============================================================

// --- 闭包：函数记住其定义时的作用域 ---
console.log("=== 闭包基础 ===");

function makeAdder(x) {
  return function(y) {
    return x + y;    // x 被闭包"捕获"
  };
}

let add5 = makeAdder(5);
let add10 = makeAdder(10);
console.log("add5(3)  =", add5(3));    // 8
console.log("add10(3) =", add10(3));   // 13

// --- 闭包实现计数器 ---
console.log("");
console.log("=== 计数器（闭包状态封装） ===");

function makeCounter(start = 0) {
  let n = start;
  return {
    inc() { n = n + 1; return n; },
    dec() { n = n - 1; return n; },
    reset() { n = 0; return n; },
    value() { return n; },
  };
}

let c = makeCounter(10);
c.inc(); c.inc(); c.dec();
console.log("value =", c.value());  // 11

// --- 高阶函数：map / filter / reduce ---
console.log("");
console.log("=== 高阶函数 ===");

let nums = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

// map：映射
let squares = nums.map(n => n * n);
console.log("平方:", squares);

// filter：过滤
let evens = nums.filter(n => n % 2 === 0);
console.log("偶数:", evens);

// reduce：归约
let sum = nums.reduce((acc, n) => acc + n, 0);
console.log("求和:", sum);

// 链式调用
let result = nums
  .filter(n => n % 2 !== 0)   // 取奇数
  .map(n => n * 2)            // 翻倍
  .reduce((a, b) => a + b, 0); // 求和
console.log("奇数翻倍求和:", result);
