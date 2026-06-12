// ============================================================
// 31-color.gs -- 终端颜色示例
// ============================================================

let c = require("@std/color");

function main() {
  console.log("=== @std/color - 终端颜色示例 ===\n");

  // 1. 基础颜色
  console.log("1. 基础颜色");
  console.log("  " + c.red("红色文本"));
  console.log("  " + c.green("绿色文本"));
  console.log("  " + c.yellow("黄色文本"));
  console.log("  " + c.blue("蓝色文本"));
  console.log("  " + c.magenta("品红色文本"));
  console.log("  " + c.cyan("青色文本"));
  console.log("  " + c.white("白色文本"));
  console.log("  " + c.gray("灰色文本"));

  // 2. 背景色
  console.log("\n2. 背景色");
  console.log("  " + c.bgRed("红色背景"));
  console.log("  " + c.bgGreen("绿色背景"));
  console.log("  " + c.bgYellow("黄色背景"));
  console.log("  " + c.bgBlue("蓝色背景"));

  // 3. 文本样式
  console.log("\n3. 文本样式");
  console.log("  " + c.bold("粗体文本"));
  console.log("  " + c.dim("暗淡文本"));
  console.log("  " + c.italic("斜体文本"));
  console.log("  " + c.underline("下划线文本"));
  console.log("  " + c.strikethrough("删除线文本"));

  // 4. 链式调用
  console.log("\n4. 链式调用");
  console.log("  " + c.bold.red("粗体红色"));
  console.log("  " + c.bgYellow.black("黄底黑字警告"));
  console.log("  " + c.bold.underline.cyan("粗体下划线青色"));

  // 5. 自定义颜色
  console.log("\n5. 自定义颜色");
  console.log("  " + c.rgb(255, 100, 50)("RGB 橙色"));
  console.log("  " + c.hex("#FF6432")("十六进制颜色"));
  console.log("  " + c.hex("#00FF00")("十六进制绿色"));

  // 6. 实际应用场景
  console.log("\n6. 实际应用：日志级别");
  logMessage("error", "数据库连接失败");
  logMessage("warning", "API 响应时间过长");
  logMessage("success", "用户注册成功");
  logMessage("info", "开始处理任务队列");

  // 7. 工具函数
  console.log("\n7. 工具函数");
  let colored = c.red("这是红色文本");
  console.log("  原始:", colored);
  console.log("  移除颜色后:", c.strip(colored));

  console.log("\n  颜色状态:");
  console.log("    enabled:", c.enabled);
  console.log("    level:", c.level);
}

function logMessage(level, message) {
  let timestamp = new Date().toISOString();
  let prefix = "";

  if (level === "error") {
    prefix = c.bold.red("[ERROR]");
  } else if (level === "warning") {
    prefix = c.bold.yellow("[WARN]");
  } else if (level === "success") {
    prefix = c.bold.green("[OK]");
  } else if (level === "info") {
    prefix = c.cyan("[INFO]");
  }

  console.log("  " + c.gray(timestamp) + " " + prefix + " " + message);
}

main();
