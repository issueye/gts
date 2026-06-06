# @std/stream

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/stream` | 原生模块路径 |

## 加载

```javascript
let stream = require("@std/stream");
```

## 接口

| 接口 | 说明 |
|------|------|
| `fromString(text) -> stream` | 从字符串创建可读流 |
| `stream.read(si

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
