# @std/random

> 密码学安全的随机数生成模块。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/random` | 原生模块路径 |

## 加载

```javascript
let rand = require("@std/random");
```

## 接口

### 基础随机

| 接口 | 说明 |
|------|------|
| `int(min, max)` | 生成 [min, max) 范围的随机整数 |
| `float(min, max)` | 生成 [min, max) 范围的随机浮点数 |
| `bool()` | 生成随机布尔值 |

### 数组操作

| 接口 | 说明 |
|------|------|
| `pick(array)` | 从数组中随机选择一个元素 |
| `sample(array, n)` | 从数组中随机选择 n 个不重复元素 |
| `shuffle(array)` | 随机打乱数组（返回新数组） |

### 随机字符串

| 接口 | 说明 |
|------|------|
| `hex(n)` | 生成 n 字节的十六进制字符串（长度 2n） |
| `base64(n)` | 生成 n 字节的 base64 字符串 |
| `alphanumeric(n)` | 生成 n 个字符的字母数字字符串 |
| `alpha(n)` | 生成 n 个字符的纯字母字符串 |
| `numeric(n)` | 生成 n 个字符的纯数字字符串 |

### UUID

| 接口 | 说明 |
|------|------|
| `uuid()` | 生成 UUID v4 |
| `uuidv4()` | 生成 UUID v4（别名） |

### 随机字节

| 接口 | 说明 |
|------|------|
| `bytes(n)` | 生成 n 字节的随机字节数组 |

## 示例

```javascript
let rand = require("@std/random");

// 基础随机
console.log(rand.int(1, 100));      // 1-99
console.log(rand.float(0.0, 1.0));  // 0.0-1.0
console.log(rand.bool());            // true/false

// 数组操作
let items = ["a", "b", "c", "d", "e"];
console.log(rand.pick(items));       // 随机一个
console.log(rand.sample(items, 3));  // 随机三个
console.log(rand.shuffle(items));    // 打乱数组

// 随机字符串
console.log(rand.hex(16));           // 32 字符十六进制
console.log(rand.base64(16));        // 24 字符 base64
console.log(rand.alphanumeric(10));  // 10 字符 a-zA-Z0-9
console.log(rand.alpha(10));         // 10 字符 a-zA-Z
console.log(rand.numeric(10));       // 10 字符 0-9

// UUID
console.log(rand.uuid());            // UUID v4

// 随机字节
console.log(rand.bytes(16));         // 16 字节数组
```

## 维护来源

- `internal/stdlib/random.go` - 模块实现
- `examples/30-random.gs` - 使用示例
