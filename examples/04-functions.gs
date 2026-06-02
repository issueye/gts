// ============================================================
// 04-functions.gs —— 第四步：函数
// ============================================================

// --- 函数声明 ---
function add(a, b) {
  return a + b;
}
console.log("add(3, 5) =", add(3, 5));

// --- 默认参数 ---
function greet(name = "World") {
  return "Hello, " + name + "!";
}
console.log("greet()       =>", greet());
console.log("greet('Alice')=>", greet("Alice"));

// --- 箭头函数 ---
let multiply = (x, y) => x * y;
console.log("multiply(4, 6) =", multiply(4, 6));

// 箭头函数作为参数
let numbers = [1, 2, 3, 4, 5];
let doubled = numbers.map(n => n * 2);
console.log("map(n*2):", doubled);

// --- 递归函数 ---
function factorial(n) {
  if (n <= 1) { return 1; }
  return n * factorial(n - 1);
}
console.log("factorial(5) =", factorial(5));

// --- 函数作为值 ---
function apply(fn, value) {
  return fn(value);
}
let square = x => x * x;
console.log("apply(square, 9) =", apply(square, 9));

// --- 立即执行函数 ---
let result = (function(a, b) { return a + b; })(10, 20);
console.log("IIFE result =", result);
