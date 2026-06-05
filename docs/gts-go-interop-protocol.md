# GTS 与 Go 交互协议设计

> 目标：让 GTS 脚本能够稳定调用 Go 提供的能力，同时允许 Go 程序把“不适合在 GTS 内封装”的能力包装成接口暴露给 GTS。
>
> 本设计优先贴合当前实现：Go 侧已有 `module.RegisterNative`、`object.Builtin`、`object.GoObject`、`object.Promise` 和 `@std/*` 原生模块机制。后续可在不破坏脚本 API 的前提下扩展到独立 Go 子进程、Socket 或远端服务。

## 1. 设计目标

1. **GTS 调用体验简单**：脚本侧像调用普通模块一样调用 Go 能力。
2. **Go 封装边界清晰**：文件系统、进程、网络、数据库、系统 API、复杂性能代码等由 Go 实现，GTS 只看到稳定接口。
3. **同步/异步统一**：快速调用可以同步返回，耗时调用返回 `Promise`。
4. **类型与错误可预测**：所有跨边界数据都有明确映射，错误统一转换成 GTS `Error`。
5. **可扩展到进程间协议**：进程内 native module 和进程间 RPC 共用同一套方法命名、类型、错误和权限模型。
6. **安全默认收敛**：能力必须显式注册、显式授权，不允许脚本反射任意 Go 对象。

## 2. 总体形态

协议分为两层：

| 层级 | 适用场景 | 当前落地方式 |
|------|----------|--------------|
| **进程内 ABI** | Go 应用嵌入 GTS，或 GTS 标准库直接调用 Go 函数 | `module.RegisterNative("@go/xxx", factory)` 返回 `object.Hash`，函数使用 `object.Builtin` |
| **进程间 GTP** | 需要隔离、长生命周期服务、独立崩溃边界、跨语言工具进程 | JSON Lines 帧协议，通过 stdin/stdout、TCP、Unix/Windows pipe、WebSocket 承载 |

脚本 API 尽量保持一致：

```javascript
const sys = require("@go/system");

let info = sys.info();
let rows = await sys.queryLargeData({ limit: 1000 });
```

当 `@go/system` 是进程内模块时，调用直接进入 Go 函数；当它是远端服务代理时，调用被编码成 GTP 请求。GTS 作者不需要关心承载方式。

## 3. 模块命名

建议保留现有 `@std/*` 给项目内置标准库，新增以下命名空间：

| 命名空间 | 含义 | 示例 |
|----------|------|------|
| `@std/*` | GTS 官方内置标准库 | `@std/fs`、`@std/net/http/client` |
| `@go/*` | 宿主 Go 程序显式注册的能力模块 | `@go/system`、`@go/image` |
| `@host/*` | 应用级宿主能力，偏业务语义 | `@host/workspace`、`@host/auth` |
| `@plugin/*` | 外部插件或扩展能力 | `@plugin/ocr` |

推荐规则：

- Go 侧不得自动暴露任意包、结构体或方法。
- 每个模块必须由宿主显式注册。
- 模块导出必须是普通对象，字段可以是常量、函数、资源构造器或资源句柄方法。
- 模块名应表达能力域，不暴露 Go 包内部路径。

## 4. 进程内 ABI

### 4.1 注册协议

Go 侧注册模块：

```go
module.RegisterNative("@go/system", func(env *object.Environment) (object.Object, error) {
    exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
    setHashMember(exports, "info", &object.Builtin{Name: "system.info", Fn: systemInfo})
    return exports, nil
})
```

GTS 侧导入：

```javascript
const system = require("@go/system");
console.log(system.info().os);
```

ABI 约束：

- `NativeModuleFactory` 只负责构造导出对象，不执行重副作用。
- 每个导出函数签名固定为 `func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object`。
- 参数校验失败返回 `object.NewError(pos, "...")`，不 panic。
- 同步函数直接返回 `object.Object`。
- 异步函数返回 `*object.Promise`，在 Go goroutine 中 resolve/reject。
- 资源对象用 `object.GoObject` 作为内部句柄，但脚本侧只能通过导出的封装方法操作。

### 4.2 类型映射

| GTS 值 | Go 表示 | 说明 |
|--------|---------|------|
| `undefined` | `object.UNDEFINED` | 缺省值，不建议跨进程序列化 |
| `null` | `object.NULL` | 显式空值 |
| `boolean` | `*object.Boolean` | 布尔 |
| `number` | `*object.Number` | 当前统一为 `float64`，整数需额外校验安全范围 |
| `string` | `*object.String` | UTF-8 字符串 |
| `array` | `*object.Array` | 有序列表 |
| `object` | `*object.Hash` | 仅使用字符串键作为跨边界稳定对象 |
| `function` | `*object.Function` 或 `*object.Builtin` | 只允许作为回调传入已声明支持回调的 Go API |
| `Promise` | `*object.Promise` | 异步结果 |
| Go 资源 | `*object.GoObject` | 脚本不可直接解包，只能由绑定方法使用 |
| Error | `*object.Error` | 统一错误对象 |

Go 封装层应提供一组转换辅助函数，避免每个模块手写类型判断：

```go
func ArgString(pos ast.Position, name string, value object.Object) (string, *object.Error)
func ArgNumber(pos ast.Position, name string, value object.Object) (float64, *object.Error)
func ArgObject(pos ast.Position, name string, value object.Object) (*object.Hash, *object.Error)
func ToObject(value any) object.Object
func FromObject(value object.Object) (any, error)
```

### 4.3 错误协议

Go 函数错误必须转换为 GTS `Error`：

```go
return object.NewError(pos, "TypeError: system.info: options must be an object")
```

错误命名建议：

| 错误名 | 场景 |
|--------|------|
| `TypeError` | 参数类型不对 |
| `RangeError` | 数值范围不合法 |
| `ReferenceError` | 引用不存在 |
| `ImportError` | 模块或能力不可用 |
| `PermissionError` | 未授权能力，建议新增到 `object.NewError` 前缀识别 |
| `TimeoutError` | 调用超时，建议新增到 `object.NewError` 前缀识别 |
| `HostError` | Go 宿主内部错误，建议新增到 `object.NewError` 前缀识别 |

错误消息格式：

```text
<module>.<method>: <human readable reason>
```

示例：

```text
system.readSecret: permission denied: capability host.secrets is not enabled
```

### 4.4 异步协议

耗时或阻塞操作必须返回 `Promise`：

```go
func queryLargeData(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
    p := env.ObjectManager().NewPromise()
    env.VM().Go(func() {
        rows, err := query()
        if err != nil {
            p.Reject(object.NewError(pos, "HostError: db.query: %v", err))
            return
        }
        p.Resolve(ToObject(rows))
    })
    return p
}
```

建议异步边界：

- 文件/网络/数据库/进程等可能阻塞的操作返回 `Promise`，除非 API 名明确带 `Sync`。
- `Promise` 必须恰好 settle 一次。
- Go goroutine 内不得直接修改 GTS 环境变量，只通过 resolve/reject 返回值。
- 回调函数必须在 VM 的调度入口执行，避免并发进入解释器。

### 4.5 资源句柄

Go 资源不能直接暴露内部结构。推荐脚本侧看到一个普通对象：

```javascript
let conn = db.open("sqlite", "file:test.db");
let rows = conn.query("select * from users");
conn.close();
```

Go 侧对象结构：

- 内部资源放在 `object.GoObject{Value: *Conn}`。
- 导出方法使用 `Builtin.Extra` 或闭包绑定资源。
- 必须提供 `close()`。
- 重复 `close()` 应幂等。
- 方法调用时检测资源是否已关闭。

资源对象建议包含：

| 字段/方法 | 要求 |
|-----------|------|
| `id` | 可选，调试用字符串，不泄露指针 |
| `closed` | 可选布尔状态 |
| `close()` | 释放资源 |
| `toString()` | 返回稳定描述，如 `[GoResource db.Conn]` |

## 5. 进程间 GTP 帧协议

GTP 是 **GoScript Transport Protocol** 的建议名称。它用于把同一套模块 API 映射到外部 Go 程序。

### 5.1 承载

第一阶段推荐 JSON Lines：

- 每行一个 JSON 对象。
- UTF-8 编码。
- stdout 只输出协议帧；日志必须走 stderr。
- 单帧建议限制默认 8 MiB，超过时使用流或文件句柄。

后续可复用同样帧结构承载于 TCP、pipe、WebSocket。

### 5.2 握手

GTS 启动外部 Go 服务后发送：

```json
{"v":1,"id":"0","type":"hello","runtime":"gts","protocol":"gtp","capabilities":["call","cancel","stream"],"modules":["@go/image"]}
```

Go 服务响应：

```json
{"v":1,"id":"0","type":"ready","service":"image-tools","version":"0.1.0","capabilities":["call","cancel"],"modules":{"@go/image":["resize","metadata"]}}
```

握手失败：

```json
{"v":1,"id":"0","type":"error","error":{"name":"ImportError","message":"unsupported protocol version","code":"GTP_UNSUPPORTED_VERSION"}}
```

### 5.3 请求与响应

调用请求：

```json
{"v":1,"id":"42","type":"call","module":"@go/image","method":"metadata","args":[{"$t":"string","v":"a.png"}],"deadlineMs":5000}
```

成功响应：

```json
{"v":1,"id":"42","type":"result","ok":true,"result":{"$t":"object","v":{"width":{"$t":"number","v":800},"height":{"$t":"number","v":600}}}}
```

失败响应：

```json
{"v":1,"id":"42","type":"result","ok":false,"error":{"name":"HostError","message":"image.metadata: unsupported format","code":"UNSUPPORTED_FORMAT","details":{"path":"a.png"}}}
```

取消请求：

```json
{"v":1,"id":"43","type":"cancel","target":"42","reason":"caller timeout"}
```

通知事件：

```json
{"v":1,"id":"evt-1","type":"event","module":"@go/image","event":"progress","data":{"$t":"object","v":{"percent":{"$t":"number","v":50}}}}
```

### 5.4 GTP 值编码

跨进程不直接使用 Go/GTS 内存对象，统一使用带类型标签的 JSON 值：

| 类型 | 编码 |
|------|------|
| undefined | `{"$t":"undefined"}` |
| null | `{"$t":"null"}` |
| boolean | `{"$t":"boolean","v":true}` |
| number | `{"$t":"number","v":1.5}` |
| string | `{"$t":"string","v":"text"}` |
| array | `{"$t":"array","v":[...]}` |
| object | `{"$t":"object","v":{"key":...}}` |
| bytes | `{"$t":"bytes","encoding":"base64","v":"..."}` |
| resource | `{"$t":"resource","id":"res-1","kind":"db.Conn"}` |
| error | `{"$t":"error","name":"Error","message":"..."}` |

约束：

- 对象键必须是字符串。
- `NaN`、`Infinity` 不应作为普通 JSON number 发送；使用 `{"$t":"number","special":"NaN"}` 这类扩展编码。
- 循环引用不支持，遇到循环应返回 `TypeError`。
- 大字节数据优先走 `bytes` 或流，不塞进普通字符串。

### 5.5 资源协议

外部服务返回资源：

```json
{"$t":"resource","id":"res-7","kind":"sqlite.Conn","methods":["query","exec","close"]}
```

GTS 运行时创建代理对象：

```javascript
conn.query(sql, params);
conn.close();
```

代理调用转换为：

```json
{"v":1,"id":"51","type":"call","resource":"res-7","method":"query","args":[...]}
```

资源释放：

```json
{"v":1,"id":"52","type":"call","resource":"res-7","method":"close","args":[]}
```

规则：

- `close` 必须幂等。
- 服务端可主动发送 `event: resourceClosed`。
- GTS VM 结束时必须释放本 VM 创建的资源。
- 资源 id 只在当前连接内有效。

### 5.6 流协议

流用于大输出、持续事件、子进程 stdout/stderr 等场景。

打开流时返回：

```json
{"$t":"resource","id":"stream-1","kind":"ReadableStream","methods":["read","close"]}
```

读取：

```json
{"v":1,"id":"60","type":"call","resource":"stream-1","method":"read","args":[{"$t":"number","v":65536}]}
```

读取结果：

```json
{"v":1,"id":"60","type":"result","ok":true,"result":{"$t":"object","v":{"done":{"$t":"boolean","v":false},"data":{"$t":"bytes","encoding":"base64","v":"..."}}}}
```

## 6. 权限与安全

能力授权建议以 manifest 或宿主配置声明：

```toml
[permissions]
fs.read = ["${projectRoot}"]
fs.write = ["${projectRoot}/dist"]
process.exec = false
network.hosts = ["api.example.com"]
host.secrets = false
```

协议规则：

- 默认无权限。
- 每个 Go 模块声明自己需要的 capability。
- 导入模块时可检查一次权限，执行敏感方法时仍需再次检查。
- 路径类 API 必须做规范化和根目录限制。
- 外部进程服务必须有启动白名单，不允许脚本传任意命令成为服务。
- 日志和错误不得泄露 secret。

## 7. 版本与兼容

版本分三类：

| 版本 | 说明 |
|------|------|
| `protocolVersion` | GTP 帧协议版本，当前建议 `1` |
| `moduleVersion` | 单个模块 API 版本，如 `@go/image@1` |
| `runtimeVersion` | GTS 运行时版本 |

兼容规则：

- 同一 major 版本内只允许新增字段或新增方法。
- 删除字段、改变语义、改变返回类型必须升 major。
- GTP 帧必须忽略未知字段。
- 方法返回对象可以新增字段，但不能移除既有字段。

## 8. 推荐实施路线

### 阶段 1：进程内 ABI 固化

1. 新增 `internal/interop` 包，集中放参数解析、类型转换、错误构造、权限检查辅助函数。
2. 将新 Go 宿主能力统一注册到 `@go/*` 或 `@host/*`。
3. 为 `object.NewError` 增加 `PermissionError`、`TimeoutError`、`HostError` 前缀识别。
4. 为 native module 增加 API 文档注册约定，沿用现有 `module.RegisterNativeAPIDoc`。

### 阶段 2：宿主注册 facade

提供公开嵌入 API，避免应用直接依赖 internal 细节：

```go
type Host struct {
    Modules map[string]Module
}

type Module struct {
    Name    string
    Version string
    Methods map[string]Method
}

type Method func(ctx Context, args []Value) (Value, error)
```

facade 内部负责转换到 `object.Builtin`。

### 阶段 3：GTP 外部服务

1. 实现 JSON Lines transport。
2. 实现 `@go/*` 远端代理模块。
3. 支持 call/result/error/cancel。
4. 支持 resource 代理和 VM 结束清理。

### 阶段 4：流、事件与工具化

1. 支持 stream resource。
2. 支持事件通知。
3. 增加协议调试日志。
4. 增加 schema/签名导出，用于补全、文档和类型检查。

## 9. API 设计准则

Go 封装给 GTS 的接口应遵循：

- GTS 侧参数少而直观，复杂选项用对象。
- 返回普通对象，不返回 Go 内部命名。
- 方法名使用小驼峰。
- 阻塞方法名显式带 `Sync`，否则返回 `Promise`。
- 错误通过 throw/reject 表达，不用 `{ ok: false }` 包业务错误，除非这是领域协议本身。
- 资源类 API 必须有生命周期方法。
- 对外暴露稳定 schema，内部 Go 结构可自由调整。

示例：

```javascript
const image = require("@go/image");

let meta = image.metadata("input.png");
await image.resize("input.png", "out.png", {
  width: 640,
  height: 480,
  fit: "cover",
});
```

不推荐：

```javascript
// 暴露 Go 结构细节、参数位置不清晰、错误不可预测
image.CallGoImageProcessor(1, "input.png", "out.png", 640, 480, true);
```

## 10. 最小可验收标准

第一版协议实现完成时，应至少满足：

1. GTS 可以 `require("@go/demo")` 调用 Go 注册函数。
2. 支持 string/number/boolean/null/array/object 双向转换。
3. Go 参数错误能变成 GTS `TypeError`。
4. Go 内部错误能变成 GTS `HostError`。
5. 一个异步 Go 方法能返回 `Promise` 并被 `await`。
6. 一个 Go 资源对象能被创建、调用方法、关闭。
7. 未授权能力返回 `PermissionError`。
8. 文档能列出模块签名。

