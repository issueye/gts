// ============================================================
// 03-control-flow.gs —— 第三步：流程控制
// ============================================================

// --- if / else if / else ---
console.log("=== if / else ===");
let score = 85;

if (score >= 90) {
  console.log("优秀");
} else if (score >= 80) {
  console.log("良好");
} else if (score >= 60) {
  console.log("及格");
} else {
  console.log("不及格");
}

// --- while 循环 ---
console.log("");
console.log("=== while 循环 ===");
let count = 0;
while (count < 5) {
  console.log("  count =", count);
  count = count + 1;
}

// --- for 循环（C 风格） ---
console.log("");
console.log("=== for 循环 ===");
for (let i = 0; i < 3; i = i + 1) {
  console.log("  i =", i);
}

// --- for-in 循环（遍历对象键） ---
console.log("");
console.log("=== for-in 循环（对象键） ===");
let obj = { a: 1, b: 2, c: 3 };
for (let key in obj) {
  console.log(`  ${key}: ${obj[key]}`);
}

// --- for-of 循环（遍历可迭代对象/数组） ---
console.log("");
console.log("=== for-of 循环（数组值） ===");
let items = ["apple", "banana", "cherry"];
for (let item of items) {
  console.log("  ", item);
}

// --- break 和 continue ---
console.log("");
console.log("=== break / continue ===");
for (let i = 0; i < 10; i = i + 1) {
  if (i === 3) {
    continue;  // 跳过 3
  }
  if (i === 7) {
    break;     // 在 7 处停止
  }
  console.log("  i =", i);
}
