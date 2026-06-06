# @std/timers

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/timers` | 原生模块路径 |

## 加载

```javascript
let timers = require("@std/timers");
```

## 接口

| 接口 | 说明 |
|------|------|
| `setTimeout(fn, delayMs, ...args)` | 延迟执行函数 |
| `clearTimeout(id)` | 取消 timeout |
| `setInterval(fn, delayMs, ...args)` | 周期执行函数 |
| `clearInterval(id)` | 取消 interval |
| `queueMicrotask(fn)` | 排入微任务 |
| `sleep(ms)` | 阻塞等待指定毫秒 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
