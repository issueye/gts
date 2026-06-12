# `@std/env` - 环境变量与配置管理

> 提供 .env 文件加载、类型安全访问和环境变量验证功能。

---

## 基础用法

```javascript
let env = require("@std/env");

// 加载 .env 文件
env.load();

// 获取环境变量
let port = env.getInt("PORT", 3000);
let debug = env.getBool("DEBUG", false);
let apiKey = env.get("API_KEY");
```

---

## 加载 .env 文件

### 基础加载

```javascript
// 加载 .env
env.load();

// 加载指定文件
env.load(".env.local");

// 覆盖已有变量
env.load(".env.local", { override: true });
```

### 多文件加载

```javascript
// 按优先级加载（后面的覆盖前面的）
env.loadMultiple([
  ".env",           // 默认配置
  ".env.local"      // 本地覆盖
]);
```

---

## 类型安全访问

### 字符串

```javascript
env.get("KEY");                    // string | undefined
env.get("KEY", "default");         // string
env.getString("KEY", "default");   // 别名
```

### 数字

```javascript
env.getInt("PORT", 3000);          // 整数
env.getFloat("RATE", 0.5);         // 浮点数
env.getNumber("VALUE", 10);        // 数字
```

### 布尔值

```javascript
env.getBool("DEBUG", false);       // boolean
// 支持：true/false, 1/0, yes/no, on/off（大小写不敏感）
```

### 数组

```javascript
// 逗号分隔（默认）
env.getArray("HOSTS");             // string[]
// "a,b,c" -> ["a", "b", "c"]

// 自定义分隔符
env.getArray("PATHS", ":");
// "path1:path2" -> ["path1", "path2"]
```

### JSON

```javascript
// 解析 JSON 字符串
env.getJson("CONFIG");
// '{"key":"value"}' -> {key: "value"}
```

---

## 验证

### 检查存在

```javascript
if (env.has("API_KEY")) {
  // 存在
}
```

### 必需变量

```javascript
// 抛出异常如果不存在
env.require(["DATABASE_URL", "API_KEY"]);

// 等价于
if (!env.has("DATABASE_URL")) {
  throw new Error("Missing required env: DATABASE_URL");
}
```

---

## 导出与转换

### 导出为对象

```javascript
// 所有环境变量
let all = env.toObject();

// 过滤前缀
let appConfig = env.toObject({ prefix: "APP_" });
// APP_NAME=test, APP_PORT=3000
// -> { APP_NAME: "test", APP_PORT: "3000" }

// 去除前缀
let clean = env.toObject({ 
  prefix: "APP_", 
  stripPrefix: true 
});
// -> { NAME: "test", PORT: "3000" }
```

---

## 运行时设置

```javascript
// 设置环境变量
env.set("KEY", "value");

// 删除环境变量
env.unset("KEY");
```

---

## 解析（不加载）

```javascript
let content = `
KEY=value
# 注释
DATABASE_URL="postgres://localhost/db"
MULTILINE="line1
line2"
`;

let parsed = env.parse(content);
// { KEY: "value", DATABASE_URL: "postgres://localhost/db", ... }
```

---

## .env 文件格式

### 基础格式

```bash
# 注释
KEY=value
ANOTHER_KEY=another value
```

### 引号

```bash
# 单引号
KEY='value'

# 双引号
KEY="value"

# 多行值
MULTILINE="line1
line2
line3"
```

### 变量展开

```bash
# 引用已定义的变量
BASE_URL=http://localhost
API_URL=${BASE_URL}/api
```

### 特殊字符

```bash
# 空格
KEY=value with spaces

# 等号
KEY=value=with=equals

# 转义
KEY="value with \"quotes\""
```

---

## 完整示例

### .env 文件

```bash
# 应用配置
APP_NAME=MyApp
APP_PORT=3000
APP_DEBUG=true

# 数据库
DATABASE_URL=postgres://localhost/mydb
DATABASE_POOL_SIZE=10

# API 配置
API_KEY=secret_key_123
API_HOSTS=api1.example.com,api2.example.com

# 功能开关
FEATURE_FLAGS={"newUI":true,"betaFeature":false}
```

### 使用示例

```javascript
let env = require("@std/env");

// 加载配置
env.load();

// 验证必需变量
env.require(["DATABASE_URL", "API_KEY"]);

// 读取配置
let config = {
  app: {
    name: env.get("APP_NAME", "DefaultApp"),
    port: env.getInt("APP_PORT", 8080),
    debug: env.getBool("APP_DEBUG", false)
  },
  database: {
    url: env.get("DATABASE_URL"),
    poolSize: env.getInt("DATABASE_POOL_SIZE", 5)
  },
  api: {
    key: env.get("API_KEY"),
    hosts: env.getArray("API_HOSTS")
  },
  features: env.getJson("FEATURE_FLAGS")
};

console.log(config);
```

---

## 最佳实践

### 1. 分层配置

```javascript
// 生产环境不覆盖
env.load(".env");            // 默认值
env.load(".env.local");      // 本地覆盖（不提交到 git）

// 开发环境覆盖
env.load(".env.development", { override: true });
```

### 2. 类型转换

```javascript
// ❌ 不好
let port = parseInt(env.get("PORT"));

// ✅ 好
let port = env.getInt("PORT", 3000);
```

### 3. 提供默认值

```javascript
// ❌ 不好
let timeout = env.getInt("TIMEOUT");  // 可能是 undefined

// ✅ 好
let timeout = env.getInt("TIMEOUT", 5000);
```

### 4. 验证关键配置

```javascript
// 启动时验证
env.require([
  "DATABASE_URL",
  "API_KEY",
  "SESSION_SECRET"
]);
```

### 5. 使用前缀组织

```bash
# .env
APP_NAME=myapp
APP_PORT=3000

DB_HOST=localhost
DB_PORT=5432
```

```javascript
let appConfig = env.toObject({ prefix: "APP_", stripPrefix: true });
let dbConfig = env.toObject({ prefix: "DB_", stripPrefix: true });
```

---

## 与其他模块配合

### 文件系统

```javascript
let env = require("@std/env");
let fs = require("@std/fs");
let path = require("@std/path");

// 加载配置文件
let configPath = path.join(env.get("CONFIG_DIR", "."), "config.toml");
if (fs.existsSync(configPath)) {
  // 加载配置
}
```

### 数据验证

```javascript
let env = require("@std/env");
let v = require("@std/validation");

// 加载环境变量
env.load();

// 验证配置
let configSchema = v.object({
  PORT: v.number().int().min(1).max(65535),
  DATABASE_URL: v.string().url(),
  API_KEY: v.string().min(20)
});

let config = {
  PORT: env.getInt("PORT"),
  DATABASE_URL: env.get("DATABASE_URL"),
  API_KEY: env.get("API_KEY")
};

configSchema.parse(config);
```

---

## API 参考

### 加载函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `load(path?, options?)` | `(string?, object?) → void` | 加载 .env 文件 |
| `loadMultiple(paths)` | `(string[]) → void` | 加载多个文件 |

### 访问函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `get(key, default?)` | `(string, string?) → string?` | 获取字符串 |
| `getString(key, default?)` | `(string, string?) → string?` | 别名 |
| `getInt(key, default?)` | `(string, number?) → number?` | 获取整数 |
| `getFloat(key, default?)` | `(string, number?) → number?` | 获取浮点数 |
| `getBool(key, default?)` | `(string, boolean?) → boolean?` | 获取布尔值 |
| `getArray(key, separator?)` | `(string, string?) → string[]?` | 获取数组 |
| `getJson(key)` | `(string) → any` | 获取 JSON |

### 验证函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `has(key)` | `(string) → boolean` | 检查是否存在 |
| `require(keys)` | `(string[]) → void` | 验证必需变量 |

### 工具函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `set(key, value)` | `(string, string) → void` | 设置变量 |
| `unset(key)` | `(string) → void` | 删除变量 |
| `toObject(options?)` | `(object?) → object` | 导出为对象 |
| `parse(content)` | `(string) → object` | 解析内容 |

---

## 实现状态

⏳ **开发中**（预计 2026-06-13 完成）

- ⏳ .env 文件解析
- ⏳ 类型转换函数
- ⏳ 验证功能
- ⏳ 多文件加载
- ⏳ 变量展开
