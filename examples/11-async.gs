// ============================================================
// 11-async.gs —— 第十一步：异步编程
// ============================================================

// --- setTimeout ---
console.log("=== setTimeout ===");
console.log("开始...");

// setTimeout 不阻塞，回调在未来某个时刻执行
setTimeout(() => {
  console.log("  3秒后执行的回调");
}, 3000);

// --- Promise 基础 ---
console.log("");
console.log("=== Promise 基础 ===");

let promise = new Promise((resolve, reject) => {
  // 模拟异步操作
  let success = true;
  if (success) {
    resolve("操作成功");
  } else {
    reject(new Error("操作失败"));
  }
});

promise.then(value => {
  console.log("  then:", value);
}).catch(e => {
  console.error("  catch:", e.message);
});

// --- Promise 链式调用 ---
console.log("");
console.log("=== Promise 链 ===");

function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

delay(1000)
  .then(() => {
    console.log("  第1步: 等待1秒");
    return delay(500);
  })
  .then(() => {
    console.log("  第2步: 又等0.5秒");
    return "链式结果";
  })
  .then(result => {
    console.log("  第3步: 得到", result);
  });

// --- async / await ---
// 注意：当前 setTimeout 不算真正的异步事件循环。
// 以下展示 async/await 的语法形式：

console.log("");
console.log("=== async / await ===");

async function asyncTask() {
  console.log("  异步任务开始");
  let value = await Promise.resolve(42);
  console.log("  await 结果:", value);
  return value * 2;
}

async function main() {
  try {
    let result = await asyncTask();
    console.log("  最终结果:", result);
  } catch (e) {
    console.error("  错误:", e.message);
  }
}

main();

// --- Promise.all ---
console.log("");
console.log("=== Promise.all ===");

async function parallel() {
  let results = await Promise.all([
    Promise.resolve(1),
    Promise.resolve(2),
    Promise.resolve(3),
  ]);
  console.log("  并行结果:", results);   // [1, 2, 3]
}

parallel();

// --- Promise.all + 延迟示例 ---
console.log("");
console.log("=== 异步延迟示例 ===");

async function delayedValue(value, ms) {
  await delay(ms);
  return value;
}

async function fetchAll() {
  console.log("  开始并行请求...");
  let [a, b, c] = await Promise.all([
    delayedValue("数据A", 500),
    delayedValue("数据B", 300),
    delayedValue("数据C", 100),
  ]);
  console.log("  全部完成:", a, b, c);
}

fetchAll();
