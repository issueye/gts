# GTP SDK

`sdk/gtp` 是 GoScript Transport Protocol 的公开 Go SDK，供外部插件项目使用。

插件应导入：

```go
import "github.com/issueye/goscript/sdk/gtp"
```

不要从插件导入 `github.com/issueye/goscript/internal/...`。`internal` 包保留给解释器、宿主和运行时桥接层使用。

## 包内容

| API | 说明 |
|-----|------|
| `Frame` | GTP 帧结构，覆盖 `hello/ready/call/result/event` 等类型 |
| `Value` | 跨进程值编码，支持 `undefined/null/boolean/number/string/bytes/array/object/resource/error` |
| `NewEncoder` / `NewDecoder` | JSON Lines 编解码器 |
| `EncodeFrame` / `DecodeFrame` | 单帧 JSON 编解码 |
| `OKResult` / `ErrorResult` | 构造调用结果帧 |
| `TypeError` / `HostError` / `NotFoundError` | 常用错误构造 |
| `RequiredObjectArg` / `StringField` / `NumberField` / `Field` | 参数和值读取 helper |

## 独立插件项目

插件建议放在仓库 `plugins/<name>` 下，每个插件有自己的 `go.mod`：

```go
module github.com/issueye/goscript/plugins/example

go 1.25.10

require github.com/issueye/goscript v0.0.0

replace github.com/issueye/goscript => ../..
```

插件入口从 stdin/stdout 读写 GTP JSON Lines，日志输出到 stderr。
