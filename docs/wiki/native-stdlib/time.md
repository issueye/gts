# @std/time

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/time` | 原生模块路径 |

## 加载

```javascript
let time = require("@std/time");
```

## 接口

| 接口 | 说明 |
|------|------|
| `now()` | 返回当前 Date |
| `nowMs()` | 返回当前毫秒时间戳 |
| `unix(seconds, nanoseconds?)` | 从 Unix 秒创建 Date |
| `unixMs(ms)` | 从毫秒时间戳创建 Date |
| `parse(value, layout?, time

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
