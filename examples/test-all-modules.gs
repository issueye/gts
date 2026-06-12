// GTS 标准库整体功能测试
// 测试所有 20 个模块的基础功能

let test = require("@std/test");

// P0 模块测试
test.describe("P0 - 工程化基础", () => {
  test.it("test 模块", () => {
    test.expect(1 + 1).toBe(2);
  });

  test.it("env 模块", () => {
    let env = require("@std/env");
    test.expect(env.get).toBeDefined();
  });

  test.it("json 模块", () => {
    let json = require("@std/json");
    let obj = json.parse('{"name":"test"}');
    test.expect(obj.name).toBe("test");
  });

  test.it("validation 模块", () => {
    let v = require("@std/validation");
    test.expect(v.string).toBeDefined();
  });
});

// P1 模块测试
test.describe("P1 - 高频工具", () => {
  test.it("collections 模块", () => {
    let c = require("@std/collections");
    let arr = c.unique([1, 2, 2, 3]);
    test.expect(arr.length).toBe(3);
  });

  test.it("random 模块", () => {
    let random = require("@std/random");
    let n = random.int(1, 100);
    test.expect(n >= 1 && n < 100).toBe(true);
  });

  test.it("color 模块", () => {
    let color = require("@std/color");
    let red = color.red("test");
    test.expect(red).toContain("test");
  });

  test.it("semver 模块", () => {
    let semver = require("@std/semver");
    test.expect(semver.valid("1.2.3")).toBe(true);
  });

  test.it("cache 模块", () => {
    let cache = require("@std/cache");
    let c = cache.create();
    c.set("key", "value");
    test.expect(c.get("key")).toBe("value");
  });
});

// P2 模块测试
test.describe("P2 - 特定场景", () => {
  test.it("retry 模块", () => {
    let retry = require("@std/retry");
    test.expect(retry.run).toBeDefined();
  });

  test.it("rate-limit 模块", () => {
    let rateLimit = require("@std/rate-limit");
    let limiter = rateLimit.create({ rate: 10 });
    test.expect(limiter.tryAcquire()).toBe(true);
  });

  test.it("glob 模块", () => {
    let glob = require("@std/glob");
    test.expect(glob.match("*.js", "test.js")).toBe(true);
  });

  test.it("diff 模块", () => {
    let diff = require("@std/diff");
    let result = diff.chars("abc", "abd");
    test.expect(result.length > 0).toBe(true);
  });

  test.it("regexp 模块", () => {
    let regexp = require("@std/regexp");
    let escaped = regexp.escape("hello.world");
    test.expect(escaped).toContain("\\.");
  });

  test.it("jwt 模块", () => {
    let jwt = require("@std/jwt");
    let token = jwt.sign({ id: 1 }, "secret");
    test.expect(jwt.verify(token, "secret")).toBe(true);
  });

  test.it("watch 模块", () => {
    let watch = require("@std/watch");
    test.expect(watch.file).toBeDefined();
  });

  test.it("compression 模块", () => {
    let compression = require("@std/compression");
    let compressed = compression.gzipCompress("hello");
    let decompressed = compression.gzipDecompress(compressed);
    test.expect(decompressed).toBe("hello");
  });

  test.it("prometheus 模块", () => {
    let prometheus = require("@std/prometheus");
    let m = prometheus.create();
    m.inc("counter");
    test.expect(m.get("counter")).toBe(1);
  });
});

// 运行测试
test.run();
