// JSON 增强功能示例
let json = require("@std/json");

console.log("=== JSON 增强功能示例 ===\n");

// 1. JSON5 解析
console.log("--- JSON5 解析 ---");
let json5Text = `{
  // 应用配置
  name: 'MyApp',
  version: '1.0.0',
  settings: {
    debug: true,    // 尾逗号
    port: 3000,
  }
}`;

let config = json.parse5(json5Text);
console.log("解析结果:", JSON.stringify(config));
console.log("  - name:", config.name);
console.log("  - version:", config.version);
console.log("  - debug:", config.settings.debug);

// 2. JSON5 序列化
console.log("\n--- JSON5 序列化 ---");
let data = { name: "John", age: 30, active: true };

console.log("默认:", json.stringify5(data));
console.log("格式化:", json.stringify5(data, { space: 2 }));
console.log("单引号:", json.stringify5(data, { quote: "single" }));

// 3. Schema 验证 - 基础
console.log("\n--- Schema 验证 ---");
let userSchema = {
  type: "object",
  required: ["name", "email"],
  properties: {
    name: {
      type: "string",
      minLength: 1,
      maxLength: 50
    },
    age: {
      type: "number",
      minimum: 0,
      maximum: 120
    },
    email: {
      type: "string",
      pattern: "^[^@]+@[^@]+\\.[^@]+$"
    }
  }
};

let validUser = {
  name: "John Doe",
  age: 30,
  email: "john@example.com"
};

let result = json.validate(validUser, userSchema);
console.log("验证有效用户:", result.valid ? "✓ 通过" : "✗ 失败");

let invalidUser = {
  name: "J",
  age: 150,
  email: "invalid-email"
};

result = json.validate(invalidUser, userSchema);
console.log("验证无效用户:", result.valid ? "✓ 通过" : "✗ 失败");
if (!result.valid) {
  console.log("错误:", result.errors);
}

// 4. Schema 验证 - 缺少必需字段
console.log("\n--- 验证必需字段 ---");
let incompleteUser = { name: "Jane" };
result = json.validate(incompleteUser, userSchema);
console.log("缺少 email:", result.valid ? "✓ 通过" : "✗ 失败");
if (!result.valid) {
  console.log("错误:", result.errors);
}

// 5. JSON Pointer - 获取值
console.log("\n--- JSON Pointer - 获取 ---");
let doc = {
  user: {
    name: "John",
    age: 30,
    address: {
      city: "NYC",
      zip: "10001"
    },
    tags: ["admin", "user", "editor"]
  },
  active: true
};

console.log("根路径:", json.get(doc, ""));
console.log("/user/name:", json.get(doc, "/user/name"));
console.log("/user/age:", json.get(doc, "/user/age"));
console.log("/user/address/city:", json.get(doc, "/user/address/city"));
console.log("/user/tags/0:", json.get(doc, "/user/tags/0"));
console.log("/user/tags/2:", json.get(doc, "/user/tags/2"));
console.log("/active:", json.get(doc, "/active"));
console.log("/unknown:", json.get(doc, "/unknown"));

// 6. JSON Pointer - 检查存在
console.log("\n--- JSON Pointer - 检查 ---");
console.log("has(/user/name):", json.has(doc, "/user/name"));
console.log("has(/user/email):", json.has(doc, "/user/email"));
console.log("has(/user/address/city):", json.has(doc, "/user/address/city"));

// 7. JSON Pointer - 设置值
console.log("\n--- JSON Pointer - 设置 ---");
let testDoc = { user: { name: "John" } };
console.log("原始:", JSON.stringify(testDoc));

json.set(testDoc, "/user/age", 30);
console.log("添加 age:", JSON.stringify(testDoc));

json.set(testDoc, "/user/address/city", "NYC");
console.log("嵌套路径:", JSON.stringify(testDoc));

json.set(testDoc, "/active", true);
console.log("添加 active:", JSON.stringify(testDoc));

// 8. JSON Pointer - 删除
console.log("\n--- JSON Pointer - 删除 ---");
console.log("删除前:", JSON.stringify(testDoc));
json.remove(testDoc, "/user/age");
console.log("删除 age:", JSON.stringify(testDoc));
json.remove(testDoc, "/user/address");
console.log("删除 address:", JSON.stringify(testDoc));

// 9. JSON Patch - add
console.log("\n--- JSON Patch - add ---");
let patchDoc = { name: "John" };
console.log("原始:", JSON.stringify(patchDoc));

json.patch(patchDoc, [
  { op: "add", path: "/age", value: 30 },
  { op: "add", path: "/email", value: "john@example.com" }
]);
console.log("添加后:", JSON.stringify(patchDoc));

// 10. JSON Patch - replace
console.log("\n--- JSON Patch - replace ---");
json.patch(patchDoc, [
  { op: "replace", path: "/name", value: "Jane" },
  { op: "replace", path: "/age", value: 25 }
]);
console.log("替换后:", JSON.stringify(patchDoc));

// 11. JSON Patch - remove
console.log("\n--- JSON Patch - remove ---");
json.patch(patchDoc, [
  { op: "remove", path: "/email" }
]);
console.log("删除后:", JSON.stringify(patchDoc));

// 12. JSON Patch - copy
console.log("\n--- JSON Patch - copy ---");
let copyDoc = {
  profile: { theme: "dark", lang: "en" },
  settings: {}
};
console.log("原始:", JSON.stringify(copyDoc));

json.patch(copyDoc, [
  { op: "copy", from: "/profile/theme", path: "/settings/theme" }
]);
console.log("复制后:", JSON.stringify(copyDoc));

// 13. JSON Patch - move
console.log("\n--- JSON Patch - move ---");
let moveDoc = {
  temp: { value: 123 },
  data: {}
};
console.log("原始:", JSON.stringify(moveDoc));

json.patch(moveDoc, [
  { op: "move", from: "/temp/value", path: "/data/value" }
]);
console.log("移动后:", JSON.stringify(moveDoc));

// 14. JSON Patch - test
console.log("\n--- JSON Patch - test ---");
let testPatchDoc = { status: "active", count: 5 };

try {
  json.patch(testPatchDoc, [
    { op: "test", path: "/status", value: "active" }
  ]);
  console.log("✓ 测试通过: status = active");
} catch (e) {
  console.log("✗ 测试失败:", e.message);
}

try {
  json.patch(testPatchDoc, [
    { op: "test", path: "/status", value: "inactive" }
  ]);
  console.log("✓ 测试通过: status = inactive");
} catch (e) {
  console.log("✗ 测试失败 (预期)");
}

// 15. JSON Diff
console.log("\n--- JSON Diff ---");
let oldDoc = {
  name: "John",
  age: 30,
  email: "john@old.com",
  temp: "remove"
};

let newDoc = {
  name: "Jane",
  age: 30,
  email: "jane@new.com",
  active: true
};

console.log("旧文档:", JSON.stringify(oldDoc));
console.log("新文档:", JSON.stringify(newDoc));

let diff = json.diff(oldDoc, newDoc);
console.log("差异 Patch:");
for (let i = 0; i < diff.length; i++) {
  console.log("  " + (i + 1) + ".", JSON.stringify(diff[i]));
}

// 16. 应用 Diff
console.log("\n--- 应用 Diff ---");
let applyDoc = {
  name: "John",
  age: 30,
  email: "john@old.com",
  temp: "remove"
};
console.log("应用前:", JSON.stringify(applyDoc));

json.patch(applyDoc, diff);
console.log("应用后:", JSON.stringify(applyDoc));

// 17. 完整示例 - 配置管理
console.log("\n--- 完整示例: 配置管理 ---");
let appConfig = json.parse5(`{
  // 应用配置
  app: {
    name: 'MyApp',
    version: '1.0.0',
    port: 3000,
  },
  // 数据库
  database: {
    host: 'localhost',
    port: 5432,
    name: 'mydb',
  }
}`);

console.log("配置:", JSON.stringify(appConfig));

// 验证配置
let configSchema = {
  type: "object",
  required: ["app", "database"],
  properties: {
    app: {
      type: "object",
      required: ["name", "version"]
    },
    database: {
      type: "object",
      required: ["host", "name"]
    }
  }
};

result = json.validate(appConfig, configSchema);
console.log("配置验证:", result.valid ? "✓ 通过" : "✗ 失败");

// 使用 Pointer 访问
console.log("应用名称:", json.get(appConfig, "/app/name"));
console.log("数据库主机:", json.get(appConfig, "/database/host"));

// 动态更新配置
json.patch(appConfig, [
  { op: "replace", path: "/app/port", value: 4000 },
  { op: "add", path: "/app/debug", value: true }
]);
console.log("更新后:", JSON.stringify(appConfig));

// 18. 数组验证
console.log("\n--- 数组验证 ---");
let arraySchema = {
  type: "array",
  items: { type: "number" },
  minItems: 1,
  maxItems: 5
};

result = json.validate([1, 2, 3], arraySchema);
console.log("有效数组:", result.valid ? "✓ 通过" : "✗ 失败");

result = json.validate([], arraySchema);
console.log("空数组:", result.valid ? "✓ 通过" : "✗ 失败");
if (!result.valid) {
  console.log("错误:", result.errors);
}

result = json.validate([1, "a", 3], arraySchema);
console.log("混合类型:", result.valid ? "✓ 通过" : "✗ 失败");
if (!result.valid) {
  console.log("错误:", result.errors);
}

console.log("\n=== 所有测试完成 ===");
