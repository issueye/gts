# @std/events

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/events` | 原生模块路径 |

## 加载

```javascript
let events = require("@std/events");
```

## 接口

| 接口 | 说明 |
|------|------|
| `EventEmitter() -> emitter` | 创建事件发射器 |
| `emitter.on(event, listener)` | 监听事件 |
| `emitter.once(event, listener)` | 监听一次事件 |
| `emitter.off(event, listener)` | 移除监听器 |
| `emitter.emit(event, ...args)` | 触发事件 |
| `emitter.listeners(event)` | 返回事件监听器数组 |
| `emitter.listenerCount(event)` | 返回监听器数量 |
| `emitter.removeAllListeners(event?)` | 移除全部或指定事件监听器 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
