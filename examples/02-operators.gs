// ============================================================
// 02-operators.gs —— 第二步：运算符
// ============================================================

// --- 算术运算符 ---
console.log("=== 算术运算符 ===");
console.log("10 + 3  =", 10 + 3);     // 13
console.log("10 - 3  =", 10 - 3);     // 7
console.log("10 * 3  =", 10 * 3);     // 30
console.log("10 / 3  =", 10 / 3);     // 3.333...
console.log("10 % 3  =", 10 % 3);     // 1（取余）
console.log("2 ** 3  =", 2 ** 3);     // 8（幂运算）

// --- 复合赋值 ---
let x = 10;
x += 5;   // x = x + 5
console.log("x += 5  =>", x);
x *= 2;   // x = x * 2
console.log("x *= 2  =>", x);

// --- 比较运算符 ---
// GoScript 只有 === 和 !==，没有 == 和 !=（避免隐式类型转换）
console.log("");
console.log("=== 比较运算符 ===");
console.log("5 === 5     =>", 5 === 5);       // true
console.log("5 === '5'   =>", 5 === "5");     // false（类型不同）
console.log("5 !== 3     =>", 5 !== 3);        // true
console.log("10 > 5      =>", 10 > 5);         // true
console.log("10 >= 10    =>", 10 >= 10);       // true
console.log("5 < 10      =>", 5 < 10);         // true
console.log("5 <= 4      =>", 5 <= 4);         // false

// --- 逻辑运算符 ---
console.log("");
console.log("=== 逻辑运算符 ===");
console.log("true && false  =>", true && false);    // false (AND)
console.log("true || false  =>", true || false);    // true  (OR)
console.log("!true          =>", !true);            // false (NOT)

// 短路求值
function greet(): string {
  console.log("    greet() 被调用了");
  return "hello";
}
let flag = false;
let result = flag && greet();  // greet() 不会被调用（短路）
console.log("false && greet() =>", result);

// --- 三元运算符 ---
console.log("");
console.log("=== 三元运算符 ===");
let age = 20;
let label = age >= 18 ? "成年" : "未成年";
console.log(`${age} 岁: ${label}`);
