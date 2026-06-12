# GTS 原生标准库扩展计划

> 本文档规划 GTS 标准库的扩展方向，包括详细的 API 设计和实施优先级。
> 创建时间：2026-06-12

---

## 一、扩展概览

### 当前状态
- 已实现：46 个标准库模块
- 覆盖领域：文件系统、网络、编码、数据库、进程、终端、加密等

### 扩展目标
- 补充工程化基础设施（测试、验证、环境管理）
- 增强数据处理能力（集合、JSON、颜色）
- 提供常用工具（随机、版本、缓存）

### 优先级划分

| 级别 | 描述 | 模块数 |
|------|------|--------|
| **P0** | 工程化必备，立即实施 | 4 |
| **P1** | 高频使用，近期实施 | 5 |
| **P2** | 特定场景，中期规划 | 11 |

---

## 二、P0 级别：工程化基础（立即实施）

### 2.1 `@std/test` - 测试框架

**优先级**：🔴 P0（最高）

**设计目标**：
- 提供类似 Jest/Vitest 的测试体验
- 支持同步/异步测试
- 内置断言库
- 支持测试套件组织

**API 设计**：

```javascript
let test = require("@std/test");

// 基础测试
test("should add numbers", () => {
  test.expect(1 + 1).toBe(2);
});

// 测试套件
test.describe("Array operations", () => {
  test.beforeEach(() => {
    // 设置
  });
  
  test.afterEach(() => {
    // 清理
  });
  
  test.it("should push items", () => {
    let arr = [];
    arr.push(1);
    test.expect(arr.length).toBe(1);
    test.expect(arr).toContain(1);
  });
  
  test.it("should handle async", async () => {
    let result = await fetchData();
    test.expect(result).toBeDefined();
  });
});

// 跳过和专注
test.skip("not ready yet", () => {});
test.only("focus on this", () => {});

// 运行测试
test.run();
```

**断言 API**：

```javascript
// 相等性
expect(value).toBe(expected);           // ===
expect(value).toEqual(expected);        // 深度相等
expect(value).toStrictEqual(expected);  // 严格深度相等

// 真值
expect(value).toBeTruthy();
expect(value).toBeFalsy();
expect(value).toBeDefined();
expect(value).toBeUndefined();
expect(value).toBeNull();

// 数字
expect(num).toBeGreaterThan(3);
expect(num).toBeGreaterThanOrEqual(3);
expect(num).toBeLessThan(10);
expect(num).toBeLessThanOrEqual(10);
expect(num).toBeCloseTo(3.14, 2);

// 字符串
expect(str).toMatch(/pattern/);
expect(str).toContain("substring");

// 数组/对象
expect(arr).toContain(item);
expect(arr).toHaveLength(3);
expect(obj).toHaveProperty("key");
expect(obj).toHaveProperty("key", value);

// 异常
expect(() => fn()).toThrow();
expect(() => fn()).toThrow("error message");
expect(() => fn()).toThrow(TypeError);

// 异步
await expect(promise).resolves.toBe(value);
await expect(promise).rejects.toThrow();

// 否定
expect(value).not.toBe(other);
```

**钩子函数**：

```javascript
test.beforeAll(() => {});     // 套件开始前
test.afterAll(() => {});      // 套件结束后
test.beforeEach(() => {});    // 每个测试前
test.afterEach(() => {});     // 每个测试后
```

**配置**：

```javascript
test.configure({
  timeout: 5000,           // 默认超时
  verbose: true,           // 详细输出
  bail: false,             // 失败后停止
  parallel: false          // 并行执行
});
```

---

### 2.2 `@std/json` - 增强 JSON 处理

**优先级**：🔴 P0

**设计目标**：
- 支持 JSON5（注释、尾逗号、单引号）
- JSON Schema 验证
- JSON Patch/Pointer
- 流式解析

**API 设计**：

```javascript
let json = require("@std/json");

// === JSON5 支持 ===
json.parse5(`{
  // 注释
  name: 'John',  // 单引号
  age: 30,       // 尾逗号
}`);

json.stringify5(obj, { space: 2, quote: "single" });

// === Schema 验证 ===
let schema = {
  type: "object",
  properties: {
    name: { type: "string", minLength: 1 },
    age: { type: "number", minimum: 0 }
  },
  required: ["name"]
};

let result = json.validate(data, schema);
// { valid: true } 或 { valid: false, errors: [...] }

json.assertValid(data, schema); // 抛出异常

// === JSON Pointer (RFC 6901) ===
json.get(doc, "/user/name");
json.set(doc, "/user/name", "John");
json.has(doc, "/user/name");
json.remove(doc, "/user/name");

// === JSON Patch (RFC 6902) ===
let patch = [
  { op: "add", path: "/name", value: "John" },
  { op: "remove", path: "/temp" },
  { op: "replace", path: "/age", value: 31 },
  { op: "move", from: "/old", path: "/new" },
  { op: "copy", from: "/src", path: "/dst" },
  { op: "test", path: "/age", value: 30 }
];

json.patch(doc, patch);
json.applyPatch(doc, patch); // 别名

// 生成 patch
let diff = json.diff(oldDoc, newDoc);

// === 流式解析 ===
let stream = json.parseStream(readableStream);
stream.on("data", (obj) => {
  console.log(obj);
});

// 流式序列化
let writeStream = json.stringifyStream();
writeStream.write(obj1);
writeStream.write(obj2);
writeStream.end();
```

---

### 2.3 `@std/env` - 环境变量与配置

**优先级**：🔴 P0

**设计目标**：
- .env 文件加载
- 类型安全访问
- 验证必需变量
- 多环境支持

**API 设计**：

```javascript
let env = require("@std/env");

// === 加载 .env 文件 ===
env.load();                              // 加载 .env
env.load(".env.local");                  // 加载指定文件
env.load(".env.local", { override: true }); // 覆盖已有变量

// 加载多个文件（按优先级）
env.loadMultiple([
  ".env.local",
  ".env",
  ".env.defaults"
]);

// === 类型安全访问 ===
env.get("KEY");                          // string | undefined
env.get("KEY", "default");               // string
env.getString("KEY", "default");         // 别名

env.getInt("PORT", 3000);                // number
env.getFloat("RATE", 0.5);               // number
env.getBool("DEBUG", false);             // boolean (支持 true/1/yes/on)

env.getArray("HOSTS");                   // string[]（逗号分隔）
env.getArray("HOSTS", ",");              // 自定义分隔符

env.getJson("CONFIG");                   // any（JSON 解析）

// === 验证 ===
env.require(["DATABASE_URL", "API_KEY"]); // 抛出异常
env.has("KEY");                           // boolean

// === 导出/转换 ===
env.toObject();                          // 所有环境变量
env.toObject({ prefix: "APP_" });        // 过滤前缀
env.toObject({ 
  prefix: "APP_", 
  stripPrefix: true 
}); // 去除前缀

// === 设置（运行时） ===
env.set("KEY", "value");
env.unset("KEY");

// === 解析（不加载） ===
let parsed = env.parse(`
KEY=value
# comment
MULTI="line1
line2"
`);
```

---

### 2.4 `@std/validation` - 数据验证

**优先级**：🔴 P0

**设计目标**：
- 链式 API
- 丰富的验证器
- 详细错误信息
- 自定义验证

**API 设计**：

```javascript
let v = require("@std/validation");

// === 基础类型 ===
v.string();
v.number();
v.boolean();
v.array();
v.object();
v.date();
v.any();

// === 字符串验证 ===
v.string()
  .min(3)
  .max(50)
  .length(10)                    // 精确长度
  .matches(/^[a-z]+$/)
  .email()
  .url()
  .uuid()
  .ip()                          // IPv4 或 IPv6
  .ipv4()
  .ipv6()
  .trim()                        // 自动修剪
  .lowercase()                   // 转小写
  .uppercase()                   // 转大写
  .oneOf(["a", "b", "c"])       // 枚举
  .required();

// === 数字验证 ===
v.number()
  .min(0)
  .max(100)
  .int()                         // 整数
  .positive()
  .negative()
  .multipleOf(5)
  .finite()
  .safe()                        // 安全整数
  .required();

// === 数组验证 ===
v.array()
  .of(v.string())                // 元素类型
  .min(1)
  .max(10)
  .length(5)
  .unique()                      // 唯一元素
  .uniqueBy(item => item.id)     // 按键唯一
  .required();

// === 对象验证 ===
v.object({
  name: v.string().required(),
  age: v.number().int().min(0),
  email: v.string().email(),
  tags: v.array().of(v.string()),
  address: v.object({
    city: v.string(),
    zip: v.string().matches(/^\d{5}$/)
  }).optional()
});

// === 可选与默认值 ===
v.string().optional();
v.string().default("default value");
v.string().nullable();           // 允许 null

// === 自定义验证 ===
v.string().test("custom", "error message", (value) => {
  return value.startsWith("prefix-");
});

// === 条件验证 ===
v.string().when("otherField", {
  is: "special",
  then: v.string().required(),
  otherwise: v.string().optional()
});

// === 执行验证 ===
let schema = v.object({
  name: v.string().required(),
  age: v.number().min(0)
});

// 同步验证
let result = schema.validate(data);
// { valid: true, value: data } 或
// { valid: false, errors: [{path, message, value}] }

// 抛出异常
let validated = schema.parse(data);

// 异步验证
let result = await schema.validateAsync(data);

// === 类型推断 ===
let userSchema = v.object({
  name: v.string(),
  age: v.number()
});

// 类型安全（运行时）
let user = userSchema.parse(data);
// user.name: string
// user.age: number
```

---

## 三、P1 级别：高频工具（近期实施）

### 3.1 `@std/collections` - 集合操作

```javascript
let col = require("@std/collections");

// 分组
col.groupBy(users, u => u.role);
col.groupBy(users, "role"); // 按属性

// 去重
col.unique([1,2,2,3]); // [1,2,3]
col.uniqueBy(items, item => item.id);
col.uniqueBy(items, "id"); // 按属性

// 分块
col.chunk([1,2,3,4,5], 2); // [[1,2], [3,4], [5]]

// 集合运算
col.difference([1,2,3], [2,3,4]); // [1]
col.intersection([1,2,3], [2,3,4]); // [2,3]
col.union([1,2], [2,3]); // [1,2,3]

// 扁平化
col.flatten([[1,2], [3,4]]); // [1,2,3,4]
col.flattenDeep([[1, [2, [3]]]]); // [1,2,3]

// 分区
let [evens, odds] = col.partition(nums, n => n % 2 === 0);

// 排序
col.sortBy(items, item => item.age);
col.sortBy(items, ["age", "name"]); // 多键

// 计数
col.countBy(items, item => item.type);

// 索引
col.keyBy(items, item => item.id);

// 采样
col.sample([1,2,3,4,5]); // 随机一个
col.sampleSize([1,2,3,4,5], 3); // 随机 N 个

// 范围
col.range(0, 10); // [0,1,2,...,9]
col.range(0, 10, 2); // [0,2,4,6,8]
```

---

### 3.2 `@std/random` - 随机数增强

```javascript
let rand = require("@std/random");

// 密码学安全随机
rand.int(0, 100);
rand.float(0.0, 1.0);
rand.bool();

// 数组操作
rand.pick(["a", "b", "c"]); // 随机一个
rand.sample(["a", "b", "c"], 2); // 随机 N 个
rand.shuffle([1,2,3,4,5]); // 洗牌

// 随机字符串
rand.hex(16); // 32 字符十六进制
rand.base64(16); // 24 字符 base64
rand.alphanumeric(10); // a-zA-Z0-9
rand.alpha(10); // a-zA-Z
rand.numeric(10); // 0-9

// UUID
rand.uuid(); // v4
rand.uuidv4();
rand.uuidv7(); // 时间排序

// 随机字节
rand.bytes(16); // Buffer

// 权重随机
rand.weighted([
  { value: "a", weight: 70 },
  { value: "b", weight: 20 },
  { value: "c", weight: 10 }
]);
```

---

### 3.3 `@std/color` - 终端颜色

```javascript
let c = require("@std/color");

// 基础颜色
console.log(c.red("error"));
console.log(c.green("success"));
console.log(c.yellow("warning"));
console.log(c.blue("info"));
console.log(c.magenta("debug"));
console.log(c.cyan("notice"));
console.log(c.white("text"));
console.log(c.gray("muted"));

// 背景色
console.log(c.bgRed("alert"));
console.log(c.bgGreen("ok"));

// 样式
console.log(c.bold("bold"));
console.log(c.dim("dim"));
console.log(c.italic("italic"));
console.log(c.underline("underline"));
console.log(c.strikethrough("strike"));

// 链式
console.log(c.bold.red("bold red"));
console.log(c.bgYellow.black("warning"));
console.log(c.bold.underline.cyan("styled"));

// RGB/Hex
console.log(c.rgb(255, 100, 50)("custom"));
console.log(c.hex("#FF6432")("hex color"));

// 条件启用
c.enabled = process.env.NO_COLOR !== "1";
c.level = 3; // 0=无, 1=基础, 2=256, 3=真彩

// 工具
c.strip(coloredString); // 去除颜色码
c.hasColor(str); // 是否包含颜色
```

---

### 3.4 `@std/semver` - 语义化版本

```javascript
let semver = require("@std/semver");

// 解析
let v = semver.parse("1.2.3-alpha.1+build.123");
// { major: 1, minor: 2, patch: 3, prerelease: ["alpha", 1], build: ["build", 123] }

// 验证
semver.valid("1.2.3"); // true
semver.valid("1.2"); // false

// 比较
semver.compare("1.2.3", "1.3.0"); // -1
semver.gt("1.3.0", "1.2.3"); // true
semver.gte("1.2.3", "1.2.3"); // true
semver.lt("1.2.0", "1.2.3"); // true
semver.lte("1.2.3", "1.2.3"); // true
semver.eq("1.2.3", "1.2.3"); // true
semver.neq("1.2.3", "1.3.0"); // true

// 范围
semver.satisfies("1.2.5", "^1.2.0"); // true
semver.satisfies("1.3.0", "~1.2.0"); // false
semver.maxSatisfying(["1.2.3", "1.2.5", "1.3.0"], "^1.2.0"); // "1.2.5"
semver.minSatisfying(["1.2.3", "1.2.5"], "^1.2.0"); // "1.2.3"

// 递增
semver.inc("1.2.3", "major"); // "2.0.0"
semver.inc("1.2.3", "minor"); // "1.3.0"
semver.inc("1.2.3", "patch"); // "1.2.4"
semver.inc("1.2.3", "prerelease", "alpha"); // "1.2.4-alpha.0"

// 范围解析
semver.validRange("^1.2.0"); // true
semver.parseRange("^1.2.0 || ~2.0.0");
```

---

### 3.5 `@std/cache` - 内存缓存

```javascript
let cache = require("@std/cache");

// 创建缓存
let c = cache.create({
  max: 100,              // 最大条目数
  ttl: 60000,            // 默认 TTL（毫秒）
  updateAgeOnGet: false, // 获取时更新访问时间
  dispose: (key, value) => {} // 淘汰回调
});

// 基础操作
c.set("key", value);
c.set("key", value, 30000); // 自定义 TTL
c.get("key");
c.has("key");
c.delete("key");
c.clear();

// 批量
c.setMany([["k1", "v1"], ["k2", "v2"]]);
c.getMany(["k1", "k2"]);
c.deleteMany(["k1", "k2"]);

// 信息
c.size; // 当前大小
c.keys(); // 所有键（LRU 顺序）
c.values(); // 所有值
c.entries(); // 所有条目

// TTL 管理
c.ttl("key"); // 剩余毫秒
c.touch("key"); // 刷新 TTL
c.expire("key", 10000); // 设置 TTL

// 统计
c.stats(); // { hits, misses, hitRate, size, maxSize }
c.resetStats();

// 包装函数（Memoize）
let cached = cache.wrap(expensiveFn, {
  ttl: 60000,
  keyGenerator: (...args) => JSON.stringify(args)
});
```

---

## 四、P2 级别：特定场景（中期规划）

### 4.1 `@std/diff` - 文本差异

```javascript
let diff = require("@std/diff");

diff.chars(str1, str2);
diff.words(str1, str2);
diff.lines(str1, str2);

let patch = diff.createPatch("file.txt", oldStr, newStr);
diff.applyPatch(oldStr, patch);
```

### 4.2 `@std/glob` - 高级匹配

```javascript
let glob = require("@std/glob");

glob.match("**/*.{js,ts}", { 
  cwd: "src",
  ignore: ["node_modules/**", "**/*.test.js"]
});

glob.stream("**/*.md"); // 流式
glob.isMatch("src/file.js", "**/*.js");
```

### 4.3 `@std/watch` - 文件监听

```javascript
let watch = require("@std/watch");

let watcher = watch.create("src/**/*.gs", {
  ignored: ["**/*.tmp"],
  persistent: true,
  ignoreInitial: true
});

watcher.on("add", (path) => {});
watcher.on("change", (path) => {});
watcher.on("unlink", (path) => {});

watcher.close();
```

### 4.4 `@std/compression` - 更多压缩

```javascript
let compress = require("@std/compression");

compress.brotli.compress(data);
compress.brotli.decompress(data);

compress.zstd.compress(data, { level: 9 });
compress.lz4.compress(data);
```

### 4.5 `@std/rate-limit` - 速率限制

```javascript
let rateLimit = require("@std/rate-limit");

let limiter = rateLimit.create({ 
  max: 100, 
  window: 60000 
});

if (limiter.consume(userId)) {
  // 允许
} else {
  // 限流
}
```

### 4.6 `@std/retry` - 重试逻辑

```javascript
let retry = require("@std/retry");

await retry.run(() => fetchData(), {
  attempts: 3,
  delay: 1000,
  backoff: "exponential", // linear, exponential
  maxDelay: 10000,
  onRetry: (err, attempt) => {}
});
```

### 4.7 `@std/jwt` - JWT 令牌

```javascript
let jwt = require("@std/jwt");

let token = jwt.sign({ userId: 123 }, secret, { 
  expiresIn: "1h",
  algorithm: "HS256"
});

let payload = jwt.verify(token, secret);
let decoded = jwt.decode(token); // 不验证
```

### 4.8 `@std/regexp` - 正则增强

```javascript
let re = require("@std/regexp");

re.escape("a.b*c?"); // "a\\.b\\*c\\?"
re.explain(/\d{3}-\d{4}/); // 可视化
```

### 4.9 `@std/pdf` - PDF 处理

```javascript
let pdf = require("@std/pdf");

pdf.create({ title: "Report" })
  .addPage().text("Hello", 50, 50)
  .save("output.pdf");

pdf.parse("input.pdf").then(doc => {
  console.log(doc.pages[0].text);
});
```

### 4.10 `@std/image` - 图像处理

```javascript
let img = require("@std/image");

img.load("input.jpg")
  .resize(800, 600)
  .grayscale()
  .save("output.jpg");
```

### 4.11 `@std/prometheus` - 指标采集

```javascript
let prom = require("@std/prometheus");

let counter = prom.counter("requests_total", "Total requests");
counter.inc();
counter.inc(5);

prom.register.metrics();
```

---

## 五、实施优先级总结

### 第一阶段（P0，立即实施）
1. **`@std/test`** - 测试框架（最高优先级）
2. **`@std/env`** - 环境变量管理
3. **`@std/json`** - JSON 增强
4. **`@std/validation`** - 数据验证

### 第二阶段（P1，2-4 周内）
5. **`@std/collections`** - 集合操作
6. **`@std/random`** - 随机数增强
7. **`@std/color`** - 终端颜色
8. **`@std/semver`** - 版本管理
9. **`@std/cache`** - 内存缓存

### 第三阶段（P2，1-3 月内）
10-20. 其余 11 个特定场景库

---

## 六、设计原则

1. **一致性**：与现有 `@std/*` 模块风格对齐
2. **简洁性**：API 简单易用，符合直觉
3. **完整性**：覆盖常见用例，避免半成品
4. **性能**：利用 Go 底层能力，避免纯 JS 实现
5. **文档**：每个 API 都有示例和说明
6. **测试**：每个模块都有完整的测试覆盖

---

## 七、下一步

参见 [native-stdlib-development-plan.md](native-stdlib-development-plan.md) 获取详细的开发计划和实施时间表。
