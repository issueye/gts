// ============================================================
// 30-random.gs -- 原生标准库：密码学安全的随机数生成
// ============================================================

let rand = require("@std/random");

function main() {
  console.log("=== @std/random 随机数生成示例 ===\n");

  // === 基础随机 ===
  console.log("--- 基础随机 ---");
  console.log("随机整数 [1, 100):", rand.int(1, 100));
  console.log("随机整数 [0, 10):", rand.int(0, 10));
  console.log("随机浮点数 [0.0, 1.0):", rand.float(0.0, 1.0));
  console.log("随机浮点数 [10.0, 20.0):", rand.float(10.0, 20.0));
  console.log("随机布尔值:", rand.bool());
  console.log("");

  // === 数组操作 ===
  console.log("--- 数组操作 ---");
  let colors = ["红", "橙", "黄", "绿", "蓝", "靛", "紫"];
  console.log("原数组:", colors);
  console.log("随机选择一个:", rand.pick(colors));
  console.log("随机选择三个:", rand.sample(colors, 3));
  console.log("打乱数组:", rand.shuffle(colors));
  console.log("原数组不变:", colors);
  console.log("");

  // === 随机字符串 ===
  console.log("--- 随机字符串 ---");
  console.log("16 字节十六进制 (32 字符):", rand.hex(16));
  console.log("16 字节 base64:", rand.base64(16));
  console.log("10 字符字母数字:", rand.alphanumeric(10));
  console.log("8 字符纯字母:", rand.alpha(8));
  console.log("6 字符纯数字:", rand.numeric(6));
  console.log("");

  // === UUID ===
  console.log("--- UUID 生成 ---");
  console.log("UUID v4:", rand.uuid());
  console.log("UUID v4 (别名):", rand.uuidv4());
  console.log("");

  // === 随机字节 ===
  console.log("--- 随机字节 ---");
  let bytes = rand.bytes(8);
  console.log("8 字节数组:", bytes);
  console.log("字节长度:", bytes.length);
  console.log("");

  // === 实用场景示例 ===
  console.log("--- 实用场景 ---");

  // 生成随机密码
  let password = rand.alphanumeric(16);
  console.log("随机密码:", password);

  // 生成会话 ID
  let sessionId = rand.hex(32);
  console.log("会话 ID:", sessionId);

  // 模拟掷骰子
  let dice = rand.int(1, 7);
  console.log("掷骰子:", dice);

  // 随机抽奖
  let participants = ["Alice", "Bob", "Charlie", "David", "Eve"];
  let winner = rand.pick(participants);
  console.log("参与者:", participants);
  console.log("获奖者:", winner);

  // 抽取多个幸运儿
  let winners = rand.sample(participants, 2);
  console.log("两位幸运儿:", winners);
}

main();
