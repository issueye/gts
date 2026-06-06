# @std/net/http/server

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/http/server` | 原生模块路径 |

## 加载

```javascript
let netHttpServer = require("@std/net/http/server");
```

## 接口

| 接口 | 说明 |
|------|------|
| `createServer(handler?, port?) -> server` | 创建并启动 HTTP 服务器 |
| `handler(req, res)` | HTTP 请求处理函数 |
| `server.close()` | 关闭服务器 |
| `res.status(code)` | 设置状态码 |
| `res.setHeader(name, value)` | 设置响应头 |
| `res.send(body)` | 发送文本响应 |
| `res.json(value)` | 发送 JSON 响应 |
| `res.end(body?)` | 结束响应 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
