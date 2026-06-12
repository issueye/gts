// GTS 标准库功能验证
console.log("=".repeat(60));
console.log("GTS 标准库功能验证 (20 个模块)");
console.log("=".repeat(60));

let passed = 0;
let failed = 0;

function test(name, fn) {
  try {
    fn();
    console.log("✅ " + name);
    passed++;
  } catch (e) {
    console.log("❌ " + name + ": " + e);
    failed++;
  }
}

// P0 - 工程化基础
console.log("\n【P0 - 工程化基础】");
test("test", () => { let m = require("@std/test"); });
test("env", () => { let m = require("@std/env"); });
test("json", () => {
  let m = require("@std/json");
  let obj = m.parse('{"a":1}');
  if (obj.a !== 1) throw "parse failed";
});
test("validation", () => { let m = require("@std/validation"); });

// P1 - 高频工具
console.log("\n【P1 - 高频工具】");
test("collections", () => {
  let m = require("@std/collections");
  let arr = m.unique([1,2,2,3]);
  if (arr.length !== 3) throw "unique failed";
});
test("random", () => {
  let m = require("@std/random");
  let n = m.int(1, 10);
  if (n < 1 || n >= 10) throw "int failed";
});
test("color", () => {
  let m = require("@std/color");
  let s = m.red("test");
  if (!s) throw "red failed";
});
test("semver", () => {
  let m = require("@std/semver");
  if (!m.valid("1.2.3")) throw "valid failed";
});
test("cache", () => {
  let m = require("@std/cache");
  let c = m.create();
  c.set("k", "v");
  if (c.get("k") !== "v") throw "cache failed";
});

// P2 - 特定场景
console.log("\n【P2 - 特定场景】");
test("retry", () => { let m = require("@std/retry"); });
test("rate-limit", () => {
  let m = require("@std/rate-limit");
  let l = m.create({rate:10});
  if (!l.tryAcquire()) throw "rate-limit failed";
});
test("glob", () => {
  let m = require("@std/glob");
  if (!m.match("*.js", "test.js")) throw "match failed";
});
test("diff", () => {
  let m = require("@std/diff");
  let r = m.chars("ab", "ac");
  if (r.length === 0) throw "diff failed";
});
test("regexp", () => {
  let m = require("@std/regexp");
  let s = m.escape("a.b");
  if (!s.includes("\\.")) throw "escape failed";
});
test("jwt", () => {
  let m = require("@std/jwt");
  let t = m.sign({id:1}, "key");
  if (!m.verify(t, "key")) throw "jwt failed";
});
test("watch", () => { let m = require("@std/watch"); });
test("compression", () => {
  let m = require("@std/compression");
  let c = m.gzipCompress("test");
  let d = m.gzipDecompress(c);
  if (d !== "test") throw "compression failed";
});
test("pdf", () => { let m = require("@std/pdf"); });
test("image", () => { let m = require("@std/image"); });
test("prometheus", () => {
  let m = require("@std/prometheus");
  let p = m.create();
  p.inc("c");
  if (p.get("c") !== 1) throw "prometheus failed";
});

console.log("\n" + "=".repeat(60));
console.log("测试结果: " + passed + " 通过, " + failed + " 失败");
console.log("完成率: " + Math.floor((passed / (passed + failed)) * 100) + "%");
console.log("=".repeat(60));
