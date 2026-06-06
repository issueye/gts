# @std/sse

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/sse` | 原生模块路径 |

## 加载

```javascript
let sse = require("@std/sse");
```

## 接口

| 接口 | 说明 |
|------|------|
| `reader(stream) -> reader` | 从文本流创建 SSE 读取器 |
| `parse(text)` | 解析 SSE 文本块 |
| `reader.next()` | 读取下一个 SSE 事件 |
| `reader.readAll()` | 读取所有 SSE 事件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
