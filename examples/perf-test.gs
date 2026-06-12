// GTS 性能压测 - 简化版
console.log("=== GTS 性能压测 ===\n");

// 压测函数
function bench(name, fn, n) {
  let start = Date.now();
  let i = 0;
  while (i < n) {
    fn();
    i = i + 1;
  }
  let elapsed = Date.now() - start;
  let ops = Math.floor(n * 1000 / elapsed);
  console.log(name + ": " + n + " 次 -> " + elapsed + "ms (" + ops + " ops/s)");
}

// 1. Collections
console.log("\n【Collections】");
let c = require("@std/collections");
bench("unique", function() { c.unique([1,2,2,3]); }, 500);
bench("shuffle", function() { c.shuffle([1,2,3,4,5]); }, 500);

// 2. Random
console.log("\n【Random】");
let r = require("@std/random");
bench("int", function() { r.int(1, 100); }, 500);

// 3. Cache
console.log("\n【Cache】");
let cache = require("@std/cache");
let ca = cache.create();
bench("cache", function() { ca.set("k","v"); ca.get("k"); }, 500);

// 4. Semver
console.log("\n【Semver】");
let s = require("@std/semver");
bench("valid", function() { s.valid("1.2.3"); }, 500);

// 5. JWT
console.log("\n【JWT】");
let jwt = require("@std/jwt");
bench("sign", function() { jwt.sign({id:1}, "key"); }, 100);

// 6. Compression
console.log("\n【Compression】");
let comp = require("@std/compression");
bench("compress", function() { comp.gzipCompress("test"); }, 100);

// 7. Rate Limit
console.log("\n【Rate Limit】");
let rl = require("@std/rate-limit");
let limiter = rl.create({rate:1000, capacity:1000});
bench("tryAcquire", function() { limiter.tryAcquire(); }, 500);

// 8. Glob
console.log("\n【Glob】");
let g = require("@std/glob");
bench("match", function() { g.match("*.js", "test.js"); }, 500);

// 9. Regexp
console.log("\n【Regexp】");
let re = require("@std/regexp");
bench("escape", function() { re.escape("a.b"); }, 500);

// 10. Prometheus
console.log("\n【Prometheus】");
let p = require("@std/prometheus");
let m = p.create();
bench("inc", function() { m.inc("c"); }, 500);

console.log("\n=== 压测完成 ===");
