# @std/encoding/base64

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/encoding/base64` | 原生模块路径 |

## 加载

```javascript
let encodingBase64 = require("@std/encoding/base64");
```

## 接口

| 接口 | 说明 |
|------|------|
| `encode(value, options?)` | Base64 编码 |
| `decode(text, options?)` | Base64 解码，`options.asBuffer` 可返回 Buffer |
| `encodeURL(value, options?)` | URL 安全 Base64 编码 |
| `decodeURL(text, options?)` | URL 安全 Base64 解码 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
