// ============================================================
// 01-basics.gs —— 第一步：变量、输出、基本类型
// ============================================================

// --- 注释 ---
// 单行注释用 //，多行注释用 /* ... */

// --- 变量声明 ---
// 使用 let 声明可变变量
let name = "GoScript";

// 使用 const 声明不可变常量 （运行时不可修改）
const VERSION = "1.0";

// --- 基本数据类型 ---
let num = 42;           // number —— 所有数字都是浮点数
let str = "hello";      // string —— 双引号字符串
let flag = true;        // boolean —— true / false
let nothing = null;     // null —— 空值

// --- 控制台输出 ---
console.log("Hello, " + name + "!");    // 普通日志
console.log("Version:", VERSION);

// --- 对象字面量 ---
let user = {
  name: "Alice",
  age: 25,
};
console.log("User:", user.name, user.age);

// --- 数组字面量 ---
let numbers = [1, 2, 3, 4, 5];
console.log("Numbers:", numbers);
console.log("Length:", numbers.length);
console.log("First:", numbers[0], "Last:", numbers[numbers.length - 1]);
