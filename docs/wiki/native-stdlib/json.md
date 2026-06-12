# `@std/json` - JSON 增强功能

> 提供 JSON5 解析、Schema 验证、Pointer 和 Patch 功能。

---

## 基础用法

```javascript
let json = require("@std/json");

// JSON5 解析（支持注释、尾逗号、单引号）
let data = json.parse5(`{
  // 配置
  name: 'app',
  version: '1.0.0',
}`);

// Schema 验证
json.validate(data, {
  type: "object",
  required: ["name"]
});

// JSON Pointer 访问
json.get(data, "/user/name");
json.set(data, "/user/age", 30);
```

---

## JSON5 支持

### 解析 JSON5

```javascript
// 支持注释
let config = json.parse5(`{
  // 应用配置
  "name": "myapp",  // 应用名称
  "version": "1.0.0"
}`);

// 支持尾逗号
let data = json.parse5(`{
  "a": 1,
  "b": 2,
}`);

// 支持单引号
let obj = json.parse5(`{
  'name': 'value',
  'count': 42
}`);

// 支持无引号键名
let simple = json.parse5(`{
  name: 'John',
  age: 30
}`);
```

### 序列化为 JSON5

```javascript
let obj = { name: "John", age: 30 };

// 默认序列化
json.stringify5(obj);
// {name:"John",age:30}

// 格式化输出
json.stringify5(obj, { space: 2 });
// {
//   name: "John",
//   age: 30
// }

// 使用单引号
json.stringify5(obj, { quote: "single" });
// {name:'John',age:30}
```

---

## Schema 验证

### 基础验证

```javascript
let schema = {
  type: "object",
  properties: {
    name: { type: "string" },
    age: { type: "number" }
  },
  required: ["name"]
};

let data = { name: "John", age: 30 };

// 验证数据
let result = json.validate(data, schema);
// { valid: true } 或 { valid: false, errors: [...] }

if (result.valid) {
  console.log("验证通过");
} else {
  console.log("验证失败:", result.errors);
}
```

### 类型验证

```javascript
// 字符串
json.validate("hello", { type: "string" });

// 数字
json.validate(42, { type: "number" });

// 布尔
json.validate(true, { type: "boolean" });

// 数组
json.validate([1, 2, 3], { type: "array" });

// 对象
json.validate({}, { type: "object" });

// null
json.validate(null, { type: "null" });
```

### 属性验证

```javascript
let userSchema = {
  type: "object",
  properties: {
    name: { 
      type: "string",
      minLength: 1,
      maxLength: 50
    },
    age: { 
      type: "number",
      minimum: 0,
      maximum: 120
    },
    email: {
      type: "string",
      pattern: "^[^@]+@[^@]+\\.[^@]+$"
    }
  },
  required: ["name", "email"]
};

let user = {
  name: "John",
  age: 30,
  email: "john@example.com"
};

json.validate(user, userSchema);
```

### 数组验证

```javascript
let arraySchema = {
  type: "array",
  items: { type: "number" },
  minItems: 1,
  maxItems: 10
};

json.validate([1, 2, 3], arraySchema); // 通过
json.validate([], arraySchema);        // 失败：minItems
json.validate([1, "a"], arraySchema);  // 失败：类型
```

---

## JSON Pointer (RFC 6901)

### 访问值

```javascript
let doc = {
  user: {
    name: "John",
    age: 30,
    tags: ["admin", "user"]
  }
};

// 获取值
json.get(doc, "/user/name");      // "John"
json.get(doc, "/user/age");       // 30
json.get(doc, "/user/tags/0");    // "admin"
json.get(doc, "/unknown");        // undefined

// 检查存在
json.has(doc, "/user/name");      // true
json.has(doc, "/user/unknown");   // false
```

### 修改值

```javascript
let doc = { user: { name: "John" } };

// 设置值
json.set(doc, "/user/age", 30);
// { user: { name: "John", age: 30 } }

// 创建嵌套路径
json.set(doc, "/user/address/city", "NYC");
// { user: { name: "John", address: { city: "NYC" } } }

// 删除值
json.remove(doc, "/user/age");
// { user: { name: "John" } }
```

### 路径格式

```javascript
// 根
json.get(doc, "");               // 整个文档

// 对象属性
json.get(doc, "/key");           // doc.key

// 嵌套属性
json.get(doc, "/a/b/c");         // doc.a.b.c

// 数组索引
json.get(doc, "/items/0");       // doc.items[0]

// 特殊字符（~ 和 /）
json.get(doc, "/a~0b");          // doc["a~b"]（~ 转义为 ~0）
json.get(doc, "/a~1b");          // doc["a/b"]（/ 转义为 ~1）
```

---

## JSON Patch (RFC 6902)

### 基础操作

```javascript
let doc = { name: "John", age: 30 };

// add - 添加
json.patch(doc, [
  { op: "add", path: "/email", value: "john@example.com" }
]);
// { name: "John", age: 30, email: "john@example.com" }

// remove - 删除
json.patch(doc, [
  { op: "remove", path: "/age" }
]);
// { name: "John" }

// replace - 替换
json.patch(doc, [
  { op: "replace", path: "/name", value: "Jane" }
]);
// { name: "Jane", age: 30 }
```

### 高级操作

```javascript
let doc = {
  profile: { name: "John" },
  settings: { theme: "dark" }
};

// move - 移动
json.patch(doc, [
  { op: "move", from: "/profile/name", path: "/settings/username" }
]);
// { profile: {}, settings: { theme: "dark", username: "John" } }

// copy - 复制
json.patch(doc, [
  { op: "copy", from: "/settings/theme", path: "/profile/theme" }
]);
// { profile: { theme: "dark" }, settings: { theme: "dark" } }

// test - 测试（验证值）
json.patch(doc, [
  { op: "test", path: "/settings/theme", value: "dark" }
]);
// 如果值匹配则成功，否则抛出异常
```

### 批量操作

```javascript
let doc = { a: 1, b: 2 };

// 多个操作按顺序执行
json.patch(doc, [
  { op: "add", path: "/c", value: 3 },
  { op: "remove", path: "/b" },
  { op: "replace", path: "/a", value: 10 }
]);
// { a: 10, c: 3 }
```

### 生成 Patch

```javascript
let oldDoc = { name: "John", age: 30 };
let newDoc = { name: "Jane", age: 30, email: "jane@example.com" };

// 计算差异
let patch = json.diff(oldDoc, newDoc);
// [
//   { op: "replace", path: "/name", value: "Jane" },
//   { op: "add", path: "/email", value: "jane@example.com" }
// ]

// 应用差异
json.patch(oldDoc, patch);
// oldDoc 现在等于 newDoc
```

---

## 完整示例

### 配置文件处理

```javascript
let json = require("@std/json");
let fs = require("@std/fs");

// 加载 JSON5 配置
let configText = fs.readTextSync("config.json5");
let config = json.parse5(configText);

// 验证配置
let schema = {
  type: "object",
  properties: {
    app: {
      type: "object",
      required: ["name", "port"]
    },
    database: {
      type: "object",
      required: ["url"]
    }
  },
  required: ["app", "database"]
};

let result = json.validate(config, schema);
if (!result.valid) {
  throw new Error("配置无效: " + JSON.stringify(result.errors));
}

// 使用配置
let appName = json.get(config, "/app/name");
let dbUrl = json.get(config, "/database/url");
```

### 动态配置更新

```javascript
let config = { user: { theme: "light", lang: "en" } };

// 应用用户偏好
let userPrefs = [
  { op: "replace", path: "/user/theme", value: "dark" },
  { op: "add", path: "/user/notifications", value: true }
];

json.patch(config, userPrefs);
console.log(config);
// { user: { theme: "dark", lang: "en", notifications: true } }
```

### API 响应验证

```javascript
let responseSchema = {
  type: "object",
  properties: {
    status: { type: "string" },
    data: { type: "array" },
    total: { type: "number", minimum: 0 }
  },
  required: ["status", "data"]
};

let response = JSON.parse(apiResponse);
let validation = json.validate(response, responseSchema);

if (validation.valid) {
  let total = json.get(response, "/total");
  console.log(`获取到 ${total} 条记录`);
} else {
  console.error("API 响应格式错误:", validation.errors);
}
```

---

## API 参考

### JSON5 函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `parse5(text)` | `(string) → any` | 解析 JSON5 字符串 |
| `stringify5(value, options?)` | `(any, object?) → string` | 序列化为 JSON5 |

### Schema 函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `validate(data, schema)` | `(any, object) → {valid, errors?}` | 验证数据 |

### Pointer 函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `get(doc, path)` | `(object, string) → any` | 获取值 |
| `set(doc, path, value)` | `(object, string, any) → void` | 设置值 |
| `has(doc, path)` | `(object, string) → boolean` | 检查存在 |
| `remove(doc, path)` | `(object, string) → void` | 删除值 |

### Patch 函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `patch(doc, operations)` | `(object, array) → void` | 应用 patch |
| `diff(oldDoc, newDoc)` | `(object, object) → array` | 生成 patch |

---

## 最佳实践

### 1. 使用 JSON5 处理配置

```javascript
// ✅ 好：可读性强
let config = json.parse5(`{
  // 服务器配置
  server: {
    host: 'localhost',
    port: 3000,  // 尾逗号
  }
}`);

// ❌ 不好：需要严格 JSON
let config = JSON.parse('{"server":{"host":"localhost","port":3000}}');
```

### 2. 先验证再使用

```javascript
// ✅ 好
let result = json.validate(data, schema);
if (result.valid) {
  processData(data);
} else {
  handleErrors(result.errors);
}

// ❌ 不好：直接使用未验证数据
processData(data);
```

### 3. 使用 Pointer 访问嵌套数据

```javascript
// ✅ 好：安全
let email = json.get(user, "/profile/contact/email");

// ❌ 不好：可能抛出异常
let email = user.profile.contact.email;
```

### 4. 用 diff/patch 做版本控制

```javascript
// 保存变更历史
let history = [];
let currentDoc = loadDocument();

function updateDocument(newDoc) {
  let patch = json.diff(currentDoc, newDoc);
  history.push(patch);
  currentDoc = newDoc;
}

// 回滚
function undo() {
  let patch = history.pop();
  // 反向应用 patch
}
```

---

## 实现状态

✅ **已完成**（2026-06-12）

- ✅ JSON5 解析与序列化
- ✅ Schema 验证（基础）
- ✅ JSON Pointer
- ✅ JSON Patch
- ✅ diff 生成
