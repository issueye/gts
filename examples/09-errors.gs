// ============================================================
// 09-errors.gs —— 第九步：错误处理
// ============================================================

// --- 抛出异常 ---
console.log("=== 抛出异常 ===");
function divide(a, b) {
  if (b === 0) {
    throw new Error("除数不能为零！");
  }
  return a / b;
}

// --- try / catch ---
try {
  console.log("10 / 2 =", divide(10, 2));
  console.log("10 / 0 =", divide(10, 0));  // 抛异常
} catch (e) {
  console.error("捕获到异常:", e.message);
}

// --- finally（总是执行） ---
console.log("");
console.log("=== finally ===");
function safeDivide(a, b) {
  try {
    if (b === 0) {
      throw new Error("除零错误");
    }
    return a / b;
  } catch (e) {
    console.error("  safeDivide 错误:", e.message);
    return 0;   // 出错时返回默认值
  } finally {
    console.log("  safeDivide 清理中...");  // 无论如何都会执行
  }
}
console.log("  safeDivide(10, 2) =", safeDivide(10, 2));
console.log("  safeDivide(10, 0) =", safeDivide(10, 0));

// --- 自定义错误类型 ---
console.log("");
console.log("=== 自定义错误 ===");

class ValidationError extends Error {
  constructor(message) {
    super(message);
    this.name = "ValidationError";
  }
}

function validateAge(age) {
  if (age < 0) {
    throw new ValidationError("年龄不能为负数");
  }
  if (age > 150) {
    throw new ValidationError("年龄超出合理范围");
  }
  return true;
}

let ages = [25, -5, 200];
for (let age of ages) {
  try {
    validateAge(age);
    console.log(`  年龄 ${age}: 验证通过`);
  } catch (e) {
    if (e instanceof ValidationError) {
      console.log(`  年龄 ${age}: 验证失败 - ${e.message}`);
    } else {
      throw e;  // 非预期错误，重新抛出
    }
  }
}

// --- 资源管理模式 ---
console.log("");
console.log("=== 资源管理模式 ===");
function withResource(action) {
  console.log("  打开资源...");
  try {
    action();
  } finally {
    console.log("  关闭资源...");  // 保证资源释放
  }
}

withResource(() => {
  console.log("  使用资源进行计算...");
  // 即使这里抛异常，资源也会被关闭
});
