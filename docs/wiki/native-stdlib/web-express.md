# @std/web / @std/express

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/web` | 原生模块路径 |
| `@std/express` | `@std/web` 的同接口别名 |

## 加载

```javascript
let webExpress = require("@std/web");
let webExpressAlias = require("@std/express");
```

## 接口

`@std/express` 是 `@std/web` 的同接口别名。

| 接口 | 说明 |
|------|------|
| `createApp() -> app` | 创建 Web 应用实例 |
| `json() -> middleware` | 解析 JSON 请求体并写入 `req.body` |
| `text() -> middleware` | 将原始文本请求体写入 `req.body` |
| `static(root) -> middleware` | 从 `root` 目录提供静态文件服务 |
| `proxy(targetOrOptions) -> middleware` | 代理请求到目标地址或代理配置 |
| `forward(targetOrOptions) -> middleware` | `proxy` 的别名，用于转发请求 |
| `app.get(path, handler, ...handlers)` | 注册 GET 路由 |
| `app.post(path, handler, ...handlers)` | 注册 POST 路由 |
| `app.put(path, handler, ...handlers)` | 注册 PUT 路由 |
| `app.patch(path, handler, ...handlers)` | 注册 PATCH 路由 |
| `app.delete(path, handler, ...handlers)` | 注册 DELETE 路由 |
| `app.all(path, handler, ...handlers)` | 注册匹配所有 HTTP 方法的路由 |
| `app.use(handler, ...handlers)` | 注册全局中间件 |
| `app.use(path, handler, ...handlers)` | 在指定路径前缀注册中间件 |
| `app.listen(port?) -> server` | 监听端口并返回服务器对象；不传端口时使用随机端口 |
| `server.close()` | 关闭服务器 |
| `handler(req, res, next)` | 路由处理函数签名 |
| `middleware(req, res, next)` | 中间件函数签名 |
| `res.status(code)` | 设置响应状态码并返回 `res` |
| `res.setHeader(name, value)` | 设置响应头并返回 `res` |
| `res.send(body)` | 发送文本响应 |
| `res.json(value)` | 发送 JSON 响应 |
| `res.redirect(url)` | 使用默认状态码跳转到 URL |
| `res.redirect(status, url)` | 使用指定状态码跳转到 URL |
| `res.end(body?)` | 结束响应，可选发送响应体 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
