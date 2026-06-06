# @std/net/ws/client

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/ws/client` | 原生模块路径 |

## 加载

```javascript
let netWsClient = require("@std/net/ws/client");
```

## 接口

| 接口 | 说明 |
|------|------|
| `connect(url, options?) -> ws` | 连接 WebSocket |
| `ws.send(data)` | 发送数据 |
| `ws.sendText(text)` | 发送文本 |
| `ws.recv()` | 接收一条消息 |
| `ws.close()` | 关闭连接 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
