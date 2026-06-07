# GTP 定时任务插件

`plugins/scheduler` 是一个基于 GTP JSON Lines 的独立 Go 项目插件，模块名为 `@plugin/scheduler`。插件只依赖公开的 `github.com/issueye/goscript/sdk/gtp` 协议 SDK，不依赖解释器、解析器或 `internal` 运行时实现。

## 启动

```bash
cd plugins/scheduler
go run .
```

插件通过 stdin/stdout 收发 GTP 帧，stderr 只用于错误日志。

## GTS 项目自动唤醒

在项目根目录的普通配置文件 `config.toml` 中配置插件：

```toml
[plugins.scheduler]
command = "go"
args = ["run", "."]
cwd = "../../plugins/scheduler"
modules = ["@plugin/scheduler"]
capabilities = ["call", "event"]
```

运行 `gs run` 时，GTS 会读取 `config.toml` 的 `[plugins]` 配置，自动启动插件进程，发送 `hello` 握手帧并等待 `ready`。脚本中可以直接加载插件模块：

```javascript
const scheduler = require("@plugin/scheduler");
let tasks = scheduler.list();
println(String(tasks.length));
```

项目结束时，GTS 会关闭插件 stdin/stdout 并清理插件进程。

## 握手

请求：

```json
{"v":1,"id":"hello-1","type":"hello","runtime":"gts","protocol":"gtp","capabilities":["call","event"],"modules":["@plugin/scheduler"]}
```

响应：

```json
{"v":1,"id":"hello-1","type":"ready","service":"scheduler","capabilities":["call","event"],"modules":{"@plugin/scheduler":["schedule","cancel","list","clear"]}}
```

## 方法

### `schedule(options)`

创建定时任务。

`options` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 可选任务名 |
| `delayMs` | number | 首次触发延迟 |
| `intervalMs` | number | 可选重复间隔；大于 0 时为重复任务 |
| `repeat` | number | 可选重复次数；`-1` 表示无限重复 |
| `payload` | any | 触发事件携带的数据 |

请求：

```json
{"v":1,"id":"1","type":"call","module":"@plugin/scheduler","method":"schedule","args":[{"$t":"object","v":{"name":{"$t":"string","v":"demo"},"delayMs":{"$t":"number","v":100},"payload":{"$t":"object","v":{"message":{"$t":"string","v":"hello"}}}}]}
```

成功响应返回任务对象。

任务触发时发送事件：

```json
{"v":1,"id":"evt-task-1-1","type":"event","module":"@plugin/scheduler","event":"trigger","data":{"$t":"object","v":{"id":{"$t":"string","v":"task-1"},"fired":{"$t":"number","v":1}}}}
```

### `cancel(id)`

取消任务。

```json
{"v":1,"id":"2","type":"call","module":"@plugin/scheduler","method":"cancel","args":[{"$t":"string","v":"task-1"}]}
```

### `list()`

列出当前等待中的任务。

```json
{"v":1,"id":"3","type":"call","module":"@plugin/scheduler","method":"list","args":[]}
```

### `clear()`

取消并清空全部等待中的任务。

```json
{"v":1,"id":"4","type":"call","module":"@plugin/scheduler","method":"clear","args":[]}
```

## 示例客户端

```bash
go run ./examples/19-gtp-scheduler-client
```

## 宿主处理插件主动事件

`examples/21-gtp-scheduler-event-handler` 展示了更接近真实宿主的用法：

1. 宿主启动 scheduler 插件。
2. 宿主调用 `schedule()` 注册任务。
3. scheduler 到点后发送 `event: trigger`，并带回任务 `payload`。
4. 宿主监听插件事件，读取 `payload.kind`，再执行自己的处理逻辑。

```bash
go run ./examples/21-gtp-scheduler-event-handler
```

这个示例中，scheduler 不执行任务内容，只负责计时和推送；真正的执行发生在宿主侧。

## 脚本监听插件事件

插件模块也会向脚本暴露事件监听方法：

```javascript
const scheduler = require("@plugin/scheduler");

scheduler.once("trigger", function(event) {
  let task = event.data;
  println(task.payload.message);
});

scheduler.schedule({
  name: "script-listener-demo",
  delayMs: 100,
  payload: { message: "handled in script" }
});
```

事件监听是按当前模块隔离的：`@plugin/a` 和 `@plugin/b` 即使来自同一个插件进程，并且都发送 `trigger` 事件，也只会触发各自模块对象上注册的监听器。

监听方法：

| 方法 | 说明 |
|------|------|
| `on(event, listener)` | 持久监听；会保持脚本运行，直到调用 `off` 或插件关闭 |
| `once(event, listener)` | 一次性监听；第一次匹配事件触发后自动释放 |
| `off(event, listener)` | 移除一个持久或一次性监听 |
| `listenerCount(event)` | 返回当前模块对应事件的监听数量 |

回调收到的 `event` 对象包含：

| 字段 | 说明 |
|------|------|
| `id` | GTP event frame id |
| `type` | 固定为 `event` |
| `module` | 发送事件的模块名 |
| `event` | 事件名 |
| `data` | 插件发送的事件数据 |

可运行示例：

```bash
cd examples/22-gtp-scheduler-script-events
go run ../../cmd/gs run
```
