# `@std/jwt` - JWT 令牌

> 提供 JWT (JSON Web Token) 生成和验证功能。

## 基础用法

```javascript
let jwt = require("@std/jwt");

// 生成 token
let token = jwt.sign({
  userId: 123,
  role: "admin",
  exp: Date.now() / 1000 + 3600  // 1 小时后过期
}, "secret-key");

// 验证 token
if (jwt.verify(token, "secret-key")) {
  let payload = jwt.decode(token);
  console.log(payload.userId);
}
```

## API

### jwt.sign(payload, secret)

生成 JWT token。

**参数**：
- `payload` - 负载数据（对象）
- `secret` - 密钥

**返回**：JWT token 字符串

### jwt.verify(token, secret)

验证 JWT token。

**参数**：
- `token` - JWT token
- `secret` - 密钥

**返回**：boolean - 是否有效

### jwt.decode(token)

解码 JWT token（不验证）。

**参数**：
- `token` - JWT token

**返回**：负载数据对象

## 示例

```javascript
let jwt = require("@std/jwt");

// 生成 token
let token = jwt.sign({
  userId: 123,
  username: "admin",
  exp: Date.now() / 1000 + 3600
}, "my-secret");

console.log("Token:", token);

// 验证
if (jwt.verify(token, "my-secret")) {
  console.log("Valid token");
  let data = jwt.decode(token);
  console.log("User ID:", data.userId);
}
```
