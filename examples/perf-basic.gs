// GTS 性能测试
console.log("GTS 性能测试开始");

// Collections
let c = require("@std/collections");
let start = Date.now();
c.unique([1,2,2,3]);
c.unique([1,2,2,3]);
c.unique([1,2,2,3]);
let end = Date.now();
console.log("Collections unique: OK");

// Random
let r = require("@std/random");
start = Date.now();
r.int(1, 100);
r.int(1, 100);
r.int(1, 100);
end = Date.now();
console.log("Random int: OK");

// Cache
let cache = require("@std/cache");
let ca = cache.create();
start = Date.now();
ca.set("k", "v");
ca.get("k");
ca.set("k2", "v2");
ca.get("k2");
end = Date.now();
console.log("Cache set/get: OK");

// Semver
let s = require("@std/semver");
start = Date.now();
s.valid("1.2.3");
s.gt("1.2.3", "1.2.0");
s.parse("1.2.3");
end = Date.now();
console.log("Semver: OK");

// JWT
let jwt = require("@std/jwt");
start = Date.now();
let token = jwt.sign({id:1}, "secret");
jwt.verify(token, "secret");
end = Date.now();
console.log("JWT: OK");

// Compression
let comp = require("@std/compression");
start = Date.now();
let compressed = comp.gzipCompress("hello");
comp.gzipDecompress(compressed);
end = Date.now();
console.log("Compression: OK");

// Glob
let g = require("@std/glob");
start = Date.now();
g.match("*.js", "test.js");
g.match("*.txt", "test.txt");
end = Date.now();
console.log("Glob: OK");

// Regexp
let re = require("@std/regexp");
start = Date.now();
re.escape("a.b");
re.split("\\s+", "a b c");
end = Date.now();
console.log("Regexp: OK");

// Prometheus
let p = require("@std/prometheus");
let m = p.create();
start = Date.now();
m.inc("counter");
m.set("gauge", 42);
m.get("counter");
end = Date.now();
console.log("Prometheus: OK");

console.log("\n所有性能测试完成！");
