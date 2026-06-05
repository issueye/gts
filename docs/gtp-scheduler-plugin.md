# GTP 定时任务插件

`cmd/gtp-scheduler` 是一个基于 GTP JSON Lines 的独立进程插件，模块名为 `@plugin/scheduler`。

## 启动

```bash
go run ./cmd/gtp-scheduler
```

插件通过 stdin/stdout 收发 GTP 帧，stderr 只用于错误日志。

## GTS 项目自动唤醒

在项目 `project.toml` 中配置插件：

```toml
[plugins.scheduler]
command = "go"
args = ["run", "./cmd/gtp-scheduler"]
cwd = "../.."
modules = ["@plugin/scheduler"]
capabilities = ["call", "event"]
```

运行 `gs run` 时，GTS 会读取 `[plugins]` 配置，自动启动插件进程，发送 `hello` 握手帧并等待 `ready`。脚本中可以直接加载插件模块：

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
