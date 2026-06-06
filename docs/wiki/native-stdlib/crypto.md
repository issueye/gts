# @std/crypto

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/crypto` | 原生模块路径 |

## 加载

```javascript
let crypto = require("@std/crypto");
```

## 接口

| 接口 | 说明 |
|------|------|
| `randomUUID()` | 生成 UUID v4 |
| `sha1(value)` | 计算 SHA-1 十六进制摘要 |
| `sha256(value)` | 计算 SHA-256 十六进制摘要 |
| `sha512(value)` | 计算 SHA-512 十六进制摘要 |
| `hmac(algorithm, key, value)` | 计算 HMAC 摘要，`algorithm` 支持 `sha1` / `sha256` / `sha512` |
| `pbkdf2(password, salt, iterations, keyLength, algorithm?, options?)` | 派生密钥 |
| `randomBytes(si

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
