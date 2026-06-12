# `@std/test` - 测试框架

> 提供完整的测试框架，支持同步/异步测试、断言库、测试套件组织和钩子函数。

---

## 基础用法

```javascript
let test = require("@std/test");

test("should add numbers", () => {
  test.expect(1 + 1).toBe(2);
});

test.run();
```

---

## 测试套件

使用 `describe` 组织相关测试：

```javascript
test.describe("Array operations", () => {
  test.it("should push items", () => {
    let arr = [];
    arr.push(1);
    test.expect(arr.length).toBe(1);
  });
  
  test.it("should pop items", () => {
    let arr = [1, 2, 3];
    let item = arr.pop();
    test.expect(item).toBe(3);
    test.expect(arr.length).toBe(2);
  });
});
```

---

## 异步测试

```javascript
test("should handle async", async () => {
  let result = await fetchData();
  test.expect(result).toBeDefined();
});

test.it("should handle promises", () => {
  return fetchData().then(result => {
    test.expect(result).toBe("data");
  });
});
```

---

## 断言 API

### 相等性断言

```javascript
// 严格相等 (===)
test.expect(value).toBe(expected);

// 深度相等
test.expect([1, 2]).toEqual([1, 2]);
test.expect({a: 1}).toEqual({a: 1});

// 严格深度相等（类型也要匹配）
test.expect(value).toStrictEqual(expected);
```

### 真值断言

```javascript
test.expect(true).toBeTruthy();
test.expect(false).toBeFalsy();
test.expect(value).toBeDefined();
test.expect(undefined).toBeUndefined();
test.expect(null).toBeNull();
```

### 数字断言

```javascript
test.expect(10).toBeGreaterThan(5);
test.expect(10).toBeGreaterThanOrEqual(10);
test.expect(5).toBeLessThan(10);
test.expect(5).toBeLessThanOrEqual(5);
test.expect(3.14159).toBeCloseTo(3.14, 2); // 精度 2 位
```

### 字符串断言

```javascript
test.expect("hello world").toMatch(/world/);
test.expect("hello").toContain("ell");
test.expect("test@example.com").toMatch(/^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$/);
```

### 数组/对象断言

```javascript
test.expect([1, 2, 3]).toContain(2);
test.expect([1, 2, 3]).toHaveLength(3);
test.expect({name: "John", age: 30}).toHaveProperty("name");
test.expect({name: "John"}).toHaveProperty("name", "John");
```

### 异常断言

```javascript
test.expect(() => {
  throw new Error("boom");
}).toThrow();

test.expect(() => {
  throw new Error("boom");
}).toThrow("boom");

test.expect(() => {
  throw new TypeError("type error");
}).toThrow(TypeError);
```

### 异步断言

```javascript
await test.expect(Promise.resolve(42)).resolves.toBe(42);
await test.expect(Promise.reject("error")).rejects.toBe("error");
await test.expect(Promise.reject(new Error("boom"))).rejects.toThrow();
```

### 否定断言

```javascript
test.expect(5).not.toBe(10);
test.expect("hello").not.toContain("world");
test.expect(() => fn()).not.toThrow();
```

---

## 钩子函数

```javascript
test.describe("User API", () => {
  let db;
  
  // 套件开始前执行一次
  test.beforeAll(() => {
    db = connectDatabase();
  });
  
  // 套件结束后执行一次
  test.afterAll(() => {
    db.close();
  });
  
  // 每个测试前执行
  test.beforeEach(() => {
    db.clear();
  });
  
  // 每个测试后执行
  test.afterEach(() => {
    // 清理
  });
  
  test.it("should create user", () => {
    let user = db.createUser({name: "John"});
    test.expect(user.name).toBe("John");
  });
});
```

---

## 跳过和专注

```javascript
// 跳过测试
test.skip("not ready yet", () => {
  // 不会执行
});

test.describe.skip("entire suite", () => {
  // 整个套件都跳过
});

// 只运行这个测试
test.only("focus on this", () => {
  test.expect(true).toBe(true);
});

test.describe.only("focus suite", () => {
  // 只运行这个套件
});
```

---

## 配置

```javascript
test.configure({
  timeout: 5000,      // 默认超时（毫秒）
  verbose: true,      // 详细输出
  bail: false,        // 失败后停止
  parallel: false     // 并行执行（暂不支持）
});
```

---

## 运行测试

```javascript
// 运行所有测试
test.run();

// 运行并获取结果
let result = test.run();
console.log(result);
// {
//   total: 10,
//   passed: 9,
//   failed: 1,
//   skipped: 0,
//   duration: 123,
//   errors: [...]
// }
```

---

## 完整示例

```javascript
let test = require("@std/test");

test.describe("Calculator", () => {
  let calc;
  
  test.beforeEach(() => {
    calc = { value: 0 };
  });
  
  test.describe("addition", () => {
    test.it("should add positive numbers", () => {
      calc.value = 5 + 3;
      test.expect(calc.value).toBe(8);
    });
    
    test.it("should add negative numbers", () => {
      calc.value = -5 + (-3);
      test.expect(calc.value).toBe(-8);
    });
  });
  
  test.describe("async operations", () => {
    test.it("should handle promises", async () => {
      let result = await Promise.resolve(42);
      test.expect(result).toBe(42);
    });
  });
});

test.run();
```

---

## 最佳实践

### 1. 使用描述性测试名称

```javascript
// ❌ 不好
test("test1", () => {});

// ✅ 好
test("should return user when ID exists", () => {});
```

### 2. 一个测试一个断言主题

```javascript
// ❌ 不好
test("user operations", () => {
  test.expect(createUser()).toBeDefined();
  test.expect(deleteUser()).toBe(true);
  test.expect(listUsers()).toHaveLength(0);
});

// ✅ 好
test("should create user", () => {
  test.expect(createUser()).toBeDefined();
});

test("should delete user", () => {
  test.expect(deleteUser()).toBe(true);
});
```

### 3. 使用钩子函数设置和清理

```javascript
test.describe("Database", () => {
  test.beforeEach(() => {
    // 每个测试前清理
    db.clear();
  });
  
  test.it("should insert record", () => {
    // 测试
  });
});
```

### 4. 测试边界条件

```javascript
test.describe("divide", () => {
  test.it("should handle normal case", () => {
    test.expect(divide(10, 2)).toBe(5);
  });
  
  test.it("should handle zero divisor", () => {
    test.expect(() => divide(10, 0)).toThrow();
  });
  
  test.it("should handle negative numbers", () => {
    test.expect(divide(-10, 2)).toBe(-5);
  });
});
```

---

## 与其他模块配合

### 文件系统测试

```javascript
let test = require("@std/test");
let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");

test.describe("File operations", () => {
  let tmpDir;
  
  test.beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-"));
  });
  
  test.afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });
  
  test.it("should write file", () => {
    let file = path.join(tmpDir, "test.txt");
    fs.writeTextSync(file, "hello");
    test.expect(fs.readTextSync(file)).toBe("hello");
  });
});
```

### HTTP 测试

```javascript
let test = require("@std/test");
let http = require("@std/net/http/client");

test.describe("API", () => {
  test.it("should fetch data", async () => {
    let resp = await http.get("https://api.example.com/data");
    test.expect(resp.status).toBe(200);
    test.expect(resp.data).toHaveProperty("items");
  });
});
```

---

## 命令行运行

```bash
# 运行测试文件
gs test.gs

# 运行所有测试
gs test/**/*.test.gs
```

---

## API 参考

### 测试定义

| 函数 | 签名 | 说明 |
|------|------|------|
| `test(name, fn)` | `(string, function) → void` | 定义测试 |
| `test.it(name, fn)` | `(string, function) → void` | `test` 别名 |
| `test.describe(name, fn)` | `(string, function) → void` | 定义测试套件 |
| `test.skip(name, fn)` | `(string, function) → void` | 跳过测试 |
| `test.only(name, fn)` | `(string, function) → void` | 只运行此测试 |

### 钩子函数

| 函数 | 说明 |
|------|------|
| `test.beforeAll(fn)` | 套件开始前 |
| `test.afterAll(fn)` | 套件结束后 |
| `test.beforeEach(fn)` | 每个测试前 |
| `test.afterEach(fn)` | 每个测试后 |

### 断言

| 断言 | 说明 |
|------|------|
| `expect(value).toBe(expected)` | 严格相等 |
| `expect(value).toEqual(expected)` | 深度相等 |
| `expect(value).toBeTruthy()` | 真值 |
| `expect(value).toBeFalsy()` | 假值 |
| `expect(n).toBeGreaterThan(m)` | 大于 |
| `expect(str).toMatch(pattern)` | 匹配正则 |
| `expect(arr).toContain(item)` | 包含元素 |
| `expect(() => fn()).toThrow()` | 抛出异常 |
| `expect(promise).resolves.toBe(v)` | Promise 成功 |

### 配置与运行

| 函数 | 说明 |
|------|------|
| `test.configure(options)` | 配置测试框架 |
| `test.run()` | 运行所有测试 |

---

## 实现状态

✅ **已实现**（2026-06-12）

- 基础测试定义
- 测试套件组织
- 同步/异步测试
- 完整断言库
- 钩子函数
- skip/only 功能
- 测试报告
