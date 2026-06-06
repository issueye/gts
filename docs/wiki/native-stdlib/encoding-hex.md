# @std/encoding/hex

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/encoding/hex` | 原生模块路径 |

## 加载

```javascript
let encodingHex = require("@std/encoding/hex");
```

## 接口

| 接口 | 说明 |
|------|------|
| `encode(value)` | 十六进制编码 |
| `decode(text, options?)` | 十六进制解码，`options.asBuffer` 可返回 Buffer |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
