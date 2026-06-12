// 环境变量管理示例
let env = require("@std/env");
let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");

console.log("=== 环境变量管理示例 ===\n");

// 1. 创建测试 .env 文件
let tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "env-test-"));
let envFile = path.join(tmpDir, ".env");

let envContent = `# 应用配置
APP_NAME=TestApp
APP_PORT=3000
APP_DEBUG=true

# 数据库
DATABASE_URL=postgres://localhost/testdb
DATABASE_POOL_SIZE=10

# API 配置
API_KEY=secret_key_12345
API_HOSTS=api1.example.com,api2.example.com

# 功能开关
FEATURE_FLAGS={"newUI":true,"betaFeature":false}

# 多行值
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"
`;

fs.writeTextSync(envFile, envContent);
console.log("✓ 创建测试 .env 文件:", envFile);

// 2. 加载 .env 文件
console.log("\n--- 加载环境变量 ---");
env.load(envFile);
console.log("✓ 已加载 .env 文件");

// 3. 基础访问
console.log("\n--- 基础访问 ---");
console.log("APP_NAME:", env.get("APP_NAME"));
console.log("APP_PORT (默认值):", env.get("UNKNOWN_PORT", "8080"));

// 4. 类型转换
console.log("\n--- 类型转换 ---");
console.log("APP_PORT (整数):", env.getInt("APP_PORT"));
console.log("APP_DEBUG (布尔):", env.getBool("APP_DEBUG"));
console.log("DATABASE_POOL_SIZE (整数):", env.getInt("DATABASE_POOL_SIZE"));

// 5. 数组访问
console.log("\n--- 数组访问 ---");
let hosts = env.getArray("API_HOSTS");
console.log("API_HOSTS:", hosts);
console.log("  - 第一个:", hosts[0]);
console.log("  - 第二个:", hosts[1]);

// 6. JSON 解析
console.log("\n--- JSON 解析 ---");
let features = env.getJson("FEATURE_FLAGS");
console.log("FEATURE_FLAGS:", JSON.stringify(features));
console.log("  - newUI:", features.newUI);
console.log("  - betaFeature:", features.betaFeature);

// 7. 检查存在
console.log("\n--- 检查存在 ---");
console.log("has(APP_NAME):", env.has("APP_NAME"));
console.log("has(UNKNOWN_KEY):", env.has("UNKNOWN_KEY"));

// 8. 验证必需变量
console.log("\n--- 验证必需变量 ---");
try {
  env.require(["DATABASE_URL", "API_KEY"]);
  console.log("✓ 所有必需变量都存在");
} catch (e) {
  console.log("✗ 缺少必需变量:", e.message);
}

try {
  env.require(["MISSING_VAR"]);
} catch (e) {
  console.log("✓ 正确捕获缺失变量:", e.message);
}

// 9. 运行时设置
console.log("\n--- 运行时设置 ---");
env.set("NEW_KEY", "new_value");
console.log("设置 NEW_KEY:", env.get("NEW_KEY"));

env.unset("NEW_KEY");
console.log("删除 NEW_KEY:", env.get("NEW_KEY", "已删除"));

// 10. 导出为对象
console.log("\n--- 导出为对象 ---");
let allEnv = env.toObject({ prefix: "APP_" });
console.log("APP_ 前缀的变量:", Object.keys(allEnv).length, "个");
for (let key in allEnv) {
  console.log("  -", key + ":", allEnv[key]);
}

// 11. 去除前缀
console.log("\n--- 去除前缀 ---");
let cleanEnv = env.toObject({ prefix: "APP_", stripPrefix: true });
for (let key in cleanEnv) {
  console.log("  -", key + ":", cleanEnv[key]);
}

// 12. 解析内容（不加载）
console.log("\n--- 解析内容 ---");
let testContent = `
KEY1=value1
KEY2=value2
# 注释
KEY3="quoted value"
`;
let parsed = env.parse(testContent);
console.log("解析结果:", Object.keys(parsed).length, "个键值对");
for (let key in parsed) {
  console.log("  -", key + ":", parsed[key]);
}

// 13. 多文件加载
console.log("\n--- 多文件加载 ---");
let envLocal = path.join(tmpDir, ".env.local");
fs.writeTextSync(envLocal, "APP_PORT=4000\nLOCAL_ONLY=true");
env.loadMultiple([envFile, envLocal]);
console.log("APP_PORT (覆盖后):", env.getInt("APP_PORT"));
console.log("LOCAL_ONLY:", env.getBool("LOCAL_ONLY"));

// 14. 构建配置对象
console.log("\n--- 构建配置对象 ---");
let config = {
  app: {
    name: env.get("APP_NAME", "DefaultApp"),
    port: env.getInt("APP_PORT", 8080),
    debug: env.getBool("APP_DEBUG", false)
  },
  database: {
    url: env.get("DATABASE_URL"),
    poolSize: env.getInt("DATABASE_POOL_SIZE", 5)
  },
  api: {
    key: env.get("API_KEY"),
    hosts: env.getArray("API_HOSTS")
  }
};

console.log("完整配置对象:");
console.log(JSON.stringify(config, null, 2));

// 15. 布尔值测试
console.log("\n--- 布尔值测试 ---");
let boolTests = [
  ["true", true],
  ["false", false],
  ["1", true],
  ["0", false],
  ["yes", true],
  ["no", false],
  ["on", true],
  ["off", false],
  ["TRUE", true],
  ["FALSE", false]
];

for (let test of boolTests) {
  env.set("BOOL_TEST", test[0]);
  let result = env.getBool("BOOL_TEST");
  let status = result === test[1] ? "✓" : "✗";
  console.log(status, test[0], "->", result);
}

// 16. 清理
console.log("\n--- 清理 ---");
fs.rmSync(tmpDir, { recursive: true });
console.log("✓ 已删除临时文件");

console.log("\n=== 所有测试完成 ===");
