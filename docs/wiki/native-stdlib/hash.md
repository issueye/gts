# @std/hash

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/hash` | 原生模块路径 |

## 加载

```javascript
let hash = require("@std/hash");
```

## 接口

| 接口 | 说明 |
|------|------|
| `adler32(value)` | 计算 Adler-32 十六进制摘要 |
| `crc32(value)` | 计算 CRC-32 十六进制摘要 |
| `crc64(value)` | 计算 CRC-64 十六进制摘要 |
| `fnv1a(value)` | 计算 FNV-1a 十六进制摘要 |
| `adler32Number(value)` | 返回 Adler-32 数字 |
| `crc32Number(value)` | 返回 CRC-32 数字 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
