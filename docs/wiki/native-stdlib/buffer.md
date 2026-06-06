# @std/buffer

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/buffer` | 原生模块路径 |

## 加载

```javascript
let buffer = require("@std/buffer");
```

## 接口

| 接口 | 说明 |
|------|------|
| `from(value, encoding?) -> Buffer` | 从字符串、数组或 Buffer 创建 Buffer |
| `alloc(si

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
