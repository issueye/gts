# @std/signal

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/signal` | 原生模块路径 |

## 加载

```javascript
let signal = require("@std/signal");
```

## 接口

| 接口 | 说明 |
|------|------|
| `supported()` | 返回支持的信号名 |
| `wait(signalsOrOptions?, timeoutMs?)` | 等待信号 |
| `notify(signalsOrOptions?) -> watcher` | 创建信号监听器 |
| `send(pid, signal?)` | 向进程发送信号 |
| `watcher.wait(timeoutMs?)` | 等待下一次信号 |
| `watcher.stop()` | 停止监听 |
| `SIGINT/SIGTERM/...` | 常用信号常量 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
