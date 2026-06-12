// ============================================================
// 33-cache.gs -- @std/cache 内存缓存示例
// ============================================================

let cache = require("@std/cache");

console.log("=== @std/cache 内存缓存示例 ===\n");

// 1. 创建缓存实例
console.log("1. 创建缓存实例");
let c = cache.create({ max: 100, ttl: 60000 });
console.log("已创建缓存 (max: 100, ttl: 60s)\n");

// 2. 基础操作
console.log("2. 设置和获取");
c.set("user:1", { name: "Alice", age: 30 });
c.set("user:2", { name: "Bob", age: 25 });

let user1 = c.get("user:1");
console.log("user:1:", user1);
console.log("has user:1:", c.has("user:1"));
console.log("size:", c.size(), "\n");

// 3. 获取所有键
console.log("3. 获取所有键");
c.set("product:101", { name: "键盘" });
c.set("product:102", { name: "鼠标" });
let keys = c.keys();
console.log("keys:", keys, "\n");

// 4. 删除操作
console.log("4. 删除操作");
c.delete("user:1");
console.log("删除后 size:", c.size());
c.clear();
console.log("清空后 size:", c.size(), "\n");

// 5. 无 TTL 的缓存
console.log("5. 无 TTL 缓存");
let c2 = cache.create({});
c2.set("data", "永久数据");
console.log("data:", c2.get("data"));
console.log("size:", c2.size());

console.log("\n示例完成！");
