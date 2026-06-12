// GTS 快速性能测试
console.log("=".repeat(60));
console.log("GTS 快速性能测试");
console.log("=".repeat(60));

function quickBench(name, fn, iterations) {
  let start = Date.now();
  for (let i = 0; i < iterations; i++) {
    fn();
  }
  let end = Date.now();
  let ms = end - start;
  let ops = Math.floor((iterations / ms) * 1000);
  console.log(name + ": " + iterations + " 次 -> " + ms + "ms (" + ops + " ops/s)");
}

console.log("\n【Collections】");
let c = require("@std/collections");
quickBench("unique", () => c.unique([1,2,2,3]), 1000);
quickBench("shuffle", () => c.shuffle([1,2,3,4,5]), 1000);

console.log("\n【Random】");
let r = require("@std/random");
quickBench("int", () => r.int(1, 100), 1000);
quickBench("uuid", () => r.uuid(), 100);

console.log("\n【Cache】");
let cache = require("@std/cache").create();
quickBench("set+get", () => { cache.set("k","v"); cache.get("k"); }, 1000);

console.log("\n【Semver】");
let s = require("@std/semver");
quickBench("valid", () => s.valid("1.2.3"), 1000);
quickBench("gt", () => s.gt("1.2.3", "1.2.0"), 1000);

console.log("\n【JWT】");
let jwt = require("@std/jwt");
let token = jwt.sign({id:1}, "secret");
quickBench("sign", () => jwt.sign({id:1}, "secret"), 100);
quickBench("verify", () => jwt.verify(token, "secret"), 100);

console.log("\n【Compression】");
let comp = require("@std/compression");
let data = "hello world";
let compressed = comp.gzipCompress(data);
quickBench("compress", () => comp.gzipCompress(data), 100);
quickBench("decompress", () => comp.gzipDecompress(compressed), 100);

console.log("\n【Rate Limit】");
let rl = require("@std/rate-limit").create({rate:10000, capacity:10000});
quickBench("tryAcquire", () => rl.tryAcquire(), 1000);

console.log("\n【Glob】");
let g = require("@std/glob");
quickBench("match", () => g.match("*.js", "test.js"), 1000);

console.log("\n【Regexp】");
let re = require("@std/regexp");
quickBench("escape", () => re.escape("hello.world"), 1000);

console.log("\n【Diff】");
let d = require("@std/diff");
quickBench("chars", () => d.chars("abc", "abd"), 100);

console.log("\n【Prometheus】");
let p = require("@std/prometheus").create();
quickBench("inc", () => p.inc("counter"), 1000);

console.log("\n" + "=".repeat(60));
console.log("测试完成！");
console.log("=".repeat(60));
