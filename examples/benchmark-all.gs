// GTS 性能压测脚本
// 测试各模块的性能表现

console.log("=".repeat(80));
console.log("GTS 性能压测");
console.log("=".repeat(80));
console.log("");

// 工具函数：性能测试
function benchmark(name, fn, iterations) {
  let start = Date.now();
  for (let i = 0; i < iterations; i++) {
    fn();
  }
  let end = Date.now();
  let duration = end - start;
  let opsPerSec = Math.floor((iterations / duration) * 1000);

  console.log(name + ":");
  console.log("  迭代次数: " + iterations);
  console.log("  总耗时: " + duration + "ms");
  console.log("  性能: " + opsPerSec + " ops/sec");
  console.log("");
}

// 1. collections 性能测试
console.log("【P1 - Collections 模块】");
let collections = require("@std/collections");
benchmark("unique() - 数组去重", () => {
  collections.unique([1,2,3,4,5,1,2,3,4,5]);
}, 10000);

benchmark("chunk() - 数组分块", () => {
  collections.chunk([1,2,3,4,5,6,7,8,9,10], 3);
}, 10000);

benchmark("shuffle() - 数组打乱", () => {
  collections.shuffle([1,2,3,4,5,6,7,8,9,10]);
}, 10000);

// 2. random 性能测试
console.log("【P1 - Random 模块】");
let random = require("@std/random");
benchmark("int() - 随机整数", () => {
  random.int(1, 100);
}, 10000);

benchmark("uuid() - UUID 生成", () => {
  random.uuid();
}, 1000);

// 3. cache 性能测试
console.log("【P1 - Cache 模块】");
let cache = require("@std/cache");
let c = cache.create();
benchmark("set/get - 缓存读写", () => {
  c.set("key", "value");
  c.get("key");
}, 10000);

// 4. semver 性能测试
console.log("【P1 - Semver 模块】");
let semver = require("@std/semver");
benchmark("valid() - 版本验证", () => {
  semver.valid("1.2.3");
}, 10000);

benchmark("gt() - 版本比较", () => {
  semver.gt("1.2.3", "1.2.0");
}, 10000);

// 5. jwt 性能测试
console.log("【P2 - JWT 模块】");
let jwt = require("@std/jwt");
let payload = { userId: 123, role: "admin" };
benchmark("sign() - JWT 签名", () => {
  jwt.sign(payload, "secret");
}, 1000);

let token = jwt.sign(payload, "secret");
benchmark("verify() - JWT 验证", () => {
  jwt.verify(token, "secret");
}, 1000);

// 6. compression 性能测试
console.log("【P2 - Compression 模块】");
let compression = require("@std/compression");
let testData = "hello world ".repeat(100);
benchmark("gzipCompress() - Gzip 压缩", () => {
  compression.gzipCompress(testData);
}, 1000);

let compressed = compression.gzipCompress(testData);
benchmark("gzipDecompress() - Gzip 解压", () => {
  compression.gzipDecompress(compressed);
}, 1000);

// 7. rate-limit 性能测试
console.log("【P2 - Rate Limit 模块】");
let rateLimit = require("@std/rate-limit");
let limiter = rateLimit.create({ rate: 100000, capacity: 100000 });
benchmark("tryAcquire() - 速率限制检查", () => {
  limiter.tryAcquire();
}, 10000);

// 8. glob 性能测试
console.log("【P2 - Glob 模块】");
let glob = require("@std/glob");
benchmark("match() - 模式匹配", () => {
  glob.match("*.js", "test.js");
}, 10000);

// 9. regexp 性能测试
console.log("【P2 - Regexp 模块】");
let regexp = require("@std/regexp");
benchmark("escape() - 正则转义", () => {
  regexp.escape("hello.world");
}, 10000);

benchmark("split() - 正则分割", () => {
  regexp.split(/\s+/, "a b  c   d");
}, 10000);

// 10. diff 性能测试
console.log("【P2 - Diff 模块】");
let diff = require("@std/diff");
benchmark("chars() - 字符差异", () => {
  diff.chars("hello", "hallo");
}, 1000);

console.log("=".repeat(80));
console.log("压测完成！");
console.log("=".repeat(80));
