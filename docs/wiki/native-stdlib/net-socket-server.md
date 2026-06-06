# @std/net/socket/server

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/socket/server` | 原生模块路径 |

## 加载

```javascript
let netSocketServer = require("@std/net/socket/server");
```

## 接口

| 接口 | 说明 |
|------|------|
| `listen(port, handler) -> server` | 监听 TCP 端口 |
| `createServer(port, handler) -> server` | `listen` 的别名 |
| `handler(socket)` | 连接处理函数 |
| `server.close()` | 关闭服务器 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
