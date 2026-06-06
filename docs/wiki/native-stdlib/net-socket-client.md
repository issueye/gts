# @std/net/socket/client

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/socket/client` | 原生模块路径 |

## 加载

```javascript
let netSocketClient = require("@std/net/socket/client");
```

## 接口

| 接口 | 说明 |
|------|------|
| `connect(host, port) -> socket` | 连接 TCP 服务 |
| `dial(host, port) -> socket` | `connect` 的别名 |
| `socket.write(data)/send(data)` | 写入数据 |
| `socket.read(si

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
