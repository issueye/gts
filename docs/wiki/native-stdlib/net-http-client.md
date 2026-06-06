# @std/net/http/client

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/http/client` | 原生模块路径 |

## 加载

```javascript
let netHttpClient = require("@std/net/http/client");
```

## 接口

| 接口 | 说明 |
|------|------|
| `get(urlOrOptions, options?)` | 发起 GET 请求 |
| `post(urlOrOptions, body?, options?)` | 发起 POST 请求 |
| `request(options)` | 发起通用 HTTP 请求 |
| `fetch(options)` | `request` 的别名 |
| `stream(options)` | 发起请求并返回流式响应 |
| `response.status/statusText/body/headers/ok/contentLength` | HTTP 响应字段 |
| `response.close()` | 关闭流式响应体 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
