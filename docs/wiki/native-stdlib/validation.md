# `@std/validation` - 数据验证

> 提供链式 API 的数据验证功能，支持字符串、数字、数组、对象等类型的验证。

---

## 基础用法

```javascript
let v = require("@std/validation");

// 字符串验证
let result = v.string().min(3).max(50).validate("hello");
console.log(result.valid); // true

// 数字验证
let schema = v.number().min(0).max(100);
let validated = schema.parse(50); // 返回 50，失败则抛出异常
```

---

## 验证器类型

### 字符串验证器

```javascript
let schema = v.string()
  .min(3)                    // 最小长度
  .max(50)                   // 最大长度
  .email()                   // 邮箱格式
  .url()                     // URL 格式
  .uuid()                    // UUID 格式
  .matches(/^[a-z]+$/)       // 正则匹配
  .required();               // 必需

// 使用
let result = schema.validate("test@example.com");
// { valid: true, value: "test@example.com" }
```

### 数字验证器

```javascript
let schema = v.number()
  .min(0)                    // 最小值
  .max(100)                  // 最大值
  .int()                     // 必须是整数
  .positive()                // 必须为正数
  .required();

// 使用
let result = schema.validate(42);
// { valid: true, value: 42 }
```

### 布尔验证器

```javascript
let schema = v.boolean().required();

let result = schema.validate(true);
// { valid: true, value: true }
```

### 数组验证器

```javascript
let schema = v.array()
  .min(1)                    // 最小长度
  .max(10)                   // 最大长度
  .required();

let result = schema.validate([1, 2, 3]);
// { valid: true, value: [1, 2, 3] }
```

### 对象验证器

```javascript
let schema = v.object().required();

let result = schema.validate({name: "John", age: 30});
// { valid: true, value: {name: "John", age: 30} }
```

---

## 链式 API

所有验证器都支持链式调用：

```javascript
let userSchema = v.string()
  .min(3)
  .max(20)
  .matches(/^[a-zA-Z0-9_]+$/)
  .required();

// 每个方法返回验证器本身，支持连续调用
```

---

## 验证方法

### validate()

返回验证结果对象：

```javascript
let schema = v.string().min(3);
let result = schema.validate("ab");

// 失败时
// {
//   valid: false,
//   value: "ab",
//   error: "String length must be at least 3"
// }

// 成功时
// {
//   valid: true,
//   value: "hello"
// }
```

### parse()

验证并返回值，失败时抛出异常：

```javascript
let schema = v.number().min(0).max(100);

try {
  let value = schema.parse(150);
} catch (e) {
  console.log(e); // Error: Number must be at most 100
}

// 成功时直接返回值
let value = schema.parse(50); // 50
```

---

## 可选与必需

### required()

标记字段为必需：

```javascript
let schema = v.string().required();

schema.validate(undefined);
// { valid: false, error: "Value is required" }
```

### optional()

标记字段为可选：

```javascript
let schema = v.string().min(3).optional();

schema.validate(undefined);
// { valid: true, value: undefined }

schema.validate("ab");
// { valid: false, error: "String length must be at least 3" }
```

---

## 字符串验证器详解

### 长度验证

```javascript
v.string().min(3);          // 最小长度 3
v.string().max(50);         // 最大长度 50
v.string().min(3).max(50);  // 长度范围 3-50
```

### 格式验证

```javascript
// 邮箱
v.string().email().validate("test@example.com");

// URL
v.string().url().validate("https://example.com");

// UUID
v.string().uuid().validate("123e4567-e89b-12d3-a456-426614174000");

// 正则匹配
v.string().matches(/^[0-9]+$/).validate("12345");
```

---

## 数字验证器详解

### 范围验证

```javascript
v.number().min(0);          // >= 0
v.number().max(100);        // <= 100
v.number().min(0).max(100); // 0 到 100
```

### 类型验证

```javascript
// 整数
v.number().int().validate(42);     // ✓
v.number().int().validate(3.14);   // ✗

// 正数
v.number().positive().validate(5);  // ✓
v.number().positive().validate(-1); // ✗
```

---

## 完整示例

### 用户注册验证

```javascript
let v = require("@std/validation");

// 定义验证规则
let usernameSchema = v.string()
  .min(3)
  .max(20)
  .matches(/^[a-zA-Z0-9_]+$/)
  .required();

let emailSchema = v.string()
  .email()
  .required();

let ageSchema = v.number()
  .int()
  .min(0)
  .max(150)
  .optional();

// 验证数据
function validateUser(data) {
  let username = usernameSchema.validate(data.username);
  if (!username.valid) {
    return { error: "Username: " + username.error };
  }

  let email = emailSchema.validate(data.email);
  if (!email.valid) {
    return { error: "Email: " + email.error };
  }

  if (data.age !== undefined) {
    let age = ageSchema.validate(data.age);
    if (!age.valid) {
      return { error: "Age: " + age.error };
    }
  }

  return { valid: true };
}

// 测试
console.log(validateUser({
  username: "john_doe",
  email: "john@example.com",
  age: 25
})); // { valid: true }

console.log(validateUser({
  username: "ab",
  email: "john@example.com"
})); // { error: "Username: String length must be at least 3" }
```

### API 参数验证

```javascript
let v = require("@std/validation");

function createProduct(params) {
  // 验证产品名称
  let name = v.string().min(1).max(100).required().parse(params.name);
  
  // 验证价格
  let price = v.number().min(0).positive().required().parse(params.price);
  
  // 验证库存（可选）
  let stock = v.number().int().min(0).optional().parse(params.stock);
  
  return {
    name: name,
    price: price,
    stock: stock !== undefined ? stock : 0
  };
}

// 使用
try {
  let product = createProduct({
    name: "Widget",
    price: 29.99,
    stock: 100
  });
  console.log("Product created:", product);
} catch (e) {
  console.log("Validation error:", e);
}
```

---

## 最佳实践

### 1. 复用验证规则

```javascript
// 定义可复用的规则
let passwordSchema = v.string()
  .min(8)
  .max(50)
  .matches(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/)
  .required();

// 在多处使用
let signupPassword = passwordSchema.parse(input.password);
let changePassword = passwordSchema.parse(input.newPassword);
```

### 2. 提前验证

```javascript
// ❌ 不好 - 数据传递后才验证
function saveUser(data) {
  // ... 已经处理了部分数据
  let email = v.string().email().parse(data.email);
}

// ✅ 好 - 函数入口处验证
function saveUser(data) {
  let email = v.string().email().parse(data.email);
  // ... 处理已验证的数据
}
```

### 3. 清晰的错误消息

```javascript
// 验证并提供上下文
let result = v.number().min(0).validate(value);
if (!result.valid) {
  console.log("Price validation failed:", result.error);
}
```

### 4. 使用 parse() 简化代码

```javascript
// ❌ 冗长
let result = schema.validate(value);
if (!result.valid) {
  throw new Error(result.error);
}
let validated = result.value;

// ✅ 简洁
let validated = schema.parse(value);
```

---

## API 参考

### 验证器创建

| 函数 | 说明 |
|------|------|
| `v.string()` | 创建字符串验证器 |
| `v.number()` | 创建数字验证器 |
| `v.boolean()` | 创建布尔验证器 |
| `v.array()` | 创建数组验证器 |
| `v.object()` | 创建对象验证器 |

### 字符串方法

| 方法 | 参数 | 说明 |
|------|------|------|
| `min(length)` | `number` | 最小长度 |
| `max(length)` | `number` | 最大长度 |
| `email()` | - | 邮箱格式 |
| `url()` | - | URL 格式 |
| `uuid()` | - | UUID 格式 |
| `matches(pattern)` | `RegExp\|string` | 正则匹配 |
| `required()` | - | 必需 |
| `optional()` | - | 可选 |

### 数字方法

| 方法 | 参数 | 说明 |
|------|------|------|
| `min(value)` | `number` | 最小值 |
| `max(value)` | `number` | 最大值 |
| `int()` | - | 必须是整数 |
| `positive()` | - | 必须为正数 |
| `required()` | - | 必需 |
| `optional()` | - | 可选 |

### 数组方法

| 方法 | 参数 | 说明 |
|------|------|------|
| `min(length)` | `number` | 最小长度 |
| `max(length)` | `number` | 最大长度 |
| `required()` | - | 必需 |
| `optional()` | - | 可选 |

### 验证方法

| 方法 | 返回值 | 说明 |
|------|--------|------|
| `validate(value)` | `{valid, value, error?}` | 验证并返回结果对象 |
| `parse(value)` | `value` | 验证成功返回值，失败抛出异常 |

---

## 实现状态

✅ **已实现**（2026-06-12）

- 字符串验证器（min, max, email, url, uuid, matches）
- 数字验证器（min, max, int, positive）
- 布尔验证器
- 数组验证器（min, max）
- 对象验证器
- 链式 API
- required/optional 修饰符
- validate/parse 方法
