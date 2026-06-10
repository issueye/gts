# 原生标准库接口知识库

> 本页是原生库知识库总入口。每一个库都有独立知识单元，详细接口请进入对应页面。

## 使用方式

```javascript
let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");

let file = path.join(os.tmpdir(), "hello.txt");
fs.writeTextSync(file, "hello");
console.log(fs.readTextSync(file));
```

原生模块路径使用 `@std/` 前缀。解析器会把 `@std/`、`@go/`、`@host/`、`@plugin/` 视为原生模块命名空间，其中当前仓库内置实现集中在 `@std/*`。

## 知识单元

### 文件、路径、系统

| 库 | 知识单元 |
|------|---------|
| `@std/fs` | [fs.md](native-stdlib/fs.md) |
| `@std/path` | [path.md](native-stdlib/path.md) |
| `@std/os` | [os.md](native-stdlib/os.md) |
| `@std/process` | [process.md](native-stdlib/process.md) |
| `@std/runtime` | [runtime.md](native-stdlib/runtime.md) |

### 进程、终端、时间

| 库 | 知识单元 |
|------|---------|
| `@std/exec` | [exec.md](native-stdlib/exec.md) |
| `@std/cli` | [cli.md](native-stdlib/cli.md) |
| `@std/pty` | [pty.md](native-stdlib/pty.md) |
| `@std/terminal` | [terminal.md](native-stdlib/terminal.md) |
| `@std/tui` | [tui.md](native-stdlib/tui.md) |
| `@std/signal` | [signal.md](native-stdlib/signal.md) |
| `@std/timers` | [timers.md](native-stdlib/timers.md) |
| `@std/time` | [time.md](native-stdlib/time.md) |
| `@std/log` | [log.md](native-stdlib/log.md) |

### 网络、Web

| 库 | 知识单元 |
|------|---------|
| `@std/net/http/client` | [net-http-client.md](native-stdlib/net-http-client.md) |
| `@std/net/http/server` | [net-http-server.md](native-stdlib/net-http-server.md) |
| `@std/net/socket/client` | [net-socket-client.md](native-stdlib/net-socket-client.md) |
| `@std/net/socket/server` | [net-socket-server.md](native-stdlib/net-socket-server.md) |
| `@std/net/ws/client` | [net-ws-client.md](native-stdlib/net-ws-client.md) |
| `@std/net/ws/server` | [net-ws-server.md](native-stdlib/net-ws-server.md) |
| `@std/net/ip` | [net-ip.md](native-stdlib/net-ip.md) |
| `@std/web` / `@std/express` | [web-express.md](native-stdlib/web-express.md) |

### 数据、编码、压缩

| 库 | 知识单元 |
|------|---------|
| `@std/buffer` | [buffer.md](native-stdlib/buffer.md) |
| `@std/encoding/base64` | [encoding-base64.md](native-stdlib/encoding-base64.md) |
| `@std/encoding/hex` | [encoding-hex.md](native-stdlib/encoding-hex.md) |
| `@std/encoding/csv` | [encoding-csv.md](native-stdlib/encoding-csv.md) |
| `@std/compress/gzip` | [compress-gzip.md](native-stdlib/compress-gzip.md) |
| `@std/archive/zip` | [archive-zip.md](native-stdlib/archive-zip.md) |

### 文本、标记、格式

| 库 | 知识单元 |
|------|---------|
| `@std/text` | [text.md](native-stdlib/text.md) |
| `@std/markdown` | [markdown.md](native-stdlib/markdown.md) |
| `@std/highlight` | [highlight.md](native-stdlib/highlight.md) |
| `@std/table` | [table.md](native-stdlib/table.md) |
| `@std/template` | [template.md](native-stdlib/template.md) |
| `@std/toml` | [toml.md](native-stdlib/toml.md) |
| `@std/yaml` | [yaml.md](native-stdlib/yaml.md) |
| `@std/xml` | [xml.md](native-stdlib/xml.md) |
| `@std/mime` | [mime.md](native-stdlib/mime.md) |
| `@std/mail` | [mail.md](native-stdlib/mail.md) |
| `@std/url` | [url.md](native-stdlib/url.md) |

### 安全、校验、数据访问

| 库 | 知识单元 |
|------|---------|
| `@std/crypto` | [crypto.md](native-stdlib/crypto.md) |
| `@std/hash` | [hash.md](native-stdlib/hash.md) |
| `@std/schema` | [schema.md](native-stdlib/schema.md) |
| `@std/db` | [db.md](native-stdlib/db.md) |

### 事件与流

| 库 | 知识单元 |
|------|---------|
| `@std/events` | [events.md](native-stdlib/events.md) |
| `@std/stream` | [stream.md](native-stdlib/stream.md) |
| `@std/sse` | [sse.md](native-stdlib/sse.md) |

## 常见组合

| 目标 | 推荐组合 |
|------|---------|
| 读写配置文件 | `@std/fs` + `@std/path` + `@std/toml` / `@std/yaml` / 内置 `JSON` |
| 命令行工具 | `@std/cli` + `@std/process` + `@std/fs` + `@std/path` + `@std/terminal` |
| 文件校验与打包 | `@std/fs` + `@std/crypto` / `@std/hash` + `@std/archive/zip` |
| 本地 HTTP 服务 | `@std/web` 或 `@std/net/http/server` |
| HTTP 客户端与流式读取 | `@std/net/http/client` + `@std/stream` + `@std/sse` |
| 子进程或交互式命令 | `@std/exec` 或 `@std/pty` |
| 终端 UI 输出 | 简单输出用 `@std/terminal` + `@std/text`；全屏状态机用 `@std/tui` |
| 数据校验入参 | `@std/schema` + 内置 `JSON` |

## 扩展原生库时怎么补文档

1. 在 Go 侧通过 `module.RegisterNative(path, factory)` 注册模块。
2. 在 `internal/stdlib/api_docs.go` 通过 `module.RegisterNativeAPIDoc(path, signatures)` 登记可打印签名。
3. 在本页加入模块分类，并在 `docs/wiki/native-stdlib/` 下维护对应知识单元。
4. 如果接口有典型组合场景，补一个短示例到 `examples/` 或 `docs/examples/`。
