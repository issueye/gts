// 简单的 JSON 测试
let json = require("@std/json");

console.log("=== 简单 JSON 测试 ===\n");

// 1. JSON5 解析
console.log("1. JSON5 解析");
let data = json.parse5('{ name: "John", age: 30 }');
console.log("  结果:", JSON.stringify(data));

// 2. JSON5 序列化
console.log("\n2. JSON5 序列化");
console.log("  ", json.stringify5({ name: "Jane", age: 25 }));

// 3. Schema 验证
console.log("\n3. Schema 验证");
let schema = {
  type: "object",
  required: ["name"],
  properties: {
    name: { type: "string" }
  }
};
let result = json.validate({ name: "John" }, schema);
console.log("  验证结果:", result.valid);

// 4. JSON Pointer
console.log("\n4. JSON Pointer");
let doc = { user: { name: "John", age: 30 } };
console.log("  /user/name:", json.get(doc, "/user/name"));
console.log("  has /user/name:", json.has(doc, "/user/name"));

json.set(doc, "/user/email", "john@example.com");
console.log("  添加 email:", JSON.stringify(doc));

// 5. JSON Patch
console.log("\n5. JSON Patch");
let patchDoc = { name: "John" };
json.patch(patchDoc, [
  { op: "add", path: "/age", value: 30 }
]);
console.log("  patch 后:", JSON.stringify(patchDoc));

// 6. JSON Diff
console.log("\n6. JSON Diff");
let oldDoc = { name: "John", age: 30 };
let newDoc = { name: "Jane", age: 30 };
let diff = json.diff(oldDoc, newDoc);
console.log("  差异数:", diff.length);

console.log("\n=== 测试完成 ===");
