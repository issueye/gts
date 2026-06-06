# @std/path

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/path` | 原生模块路径 |

## 加载

```javascript
let path = require("@std/path");
```

## 接口

| 接口 | 说明 |
|------|------|
| `join(...parts)` | 拼接路径 |
| `resolve(...parts)` | 解析为绝对路径 |
| `relative(from, to)` | 计算相对路径 |
| `normali

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
