// GTS 性能压测
console.log("=== GTS 性能压测 ===");

// 1. Collections
console.log("\nCollections:");
let c = require("@std/collections");
let t1 = Date.now();
let i = 0;
while (i < 100) {
  c.unique([1,2,2,3]);
  i++;
}
console.log("unique: 100 次 -> " + String(Date.now() - t1) + "ms");

// 2. Random
console.log("\nRandom:");
let r = require("@std/random");
let t2 = Date.now();
i = 0;
while (i < 100) {
  r.int(1, 100);
  i++;
}
console.log("int: 100 次 -> " + String(Date.now() - t2) + "ms");

// 3. Cache
console.log("\nCache:");
let cache = require("@std/cache");
let ca = cache.create();
let t3 = Date.now();
i = 0;
while (i < 100) {
  ca.set("k", "v");
  ca.get("k");
  i++;
}
console.log("set+get: 100 次 -> " + String(Date.now() - t3) + "ms");

// 4. Semver
console.log("\nSemver:");
let s = require("@std/semver");
let t4 = Date.now();
i = 0;
while (i < 100) {
  s.valid("1.2.3");
  i++;
}
console.log("valid: 100 次 -> " + String(Date.now() - t4) + "ms");

// 5. JWT
console.log("\nJWT:");
let jwt = require("@std/jwt");
let t5 = Date.now();
i = 0;
while (i < 50) {
  jwt.sign({id:1}, "key");
  i++;
}
console.log("sign: 50 次 -> " + String(Date.now() - t5) + "ms");

// 6. Glob
console.log("\nGlob:");
let g = require("@std/glob");
let t6 = Date.now();
i = 0;
while (i < 100) {
  g.match("*.js", "test.js");
  i++;
}
console.log("match: 100 次 -> " + String(Date.now() - t6) + "ms");

// 7. Regexp
console.log("\nRegexp:");
let re = require("@std/regexp");
let t7 = Date.now();
i = 0;
while (i < 100) {
  re.escape("a.b");
  i++;
}
console.log("escape: 100 次 -> " + String(Date.now() - t7) + "ms");

console.log("\n=== 压测完成 ===");
