# GTP Scheduler Script Events 示例

这个示例展示 GTP 插件主动事件可以在 GoScript 脚本里监听。

运行：

```bash
cd examples/22-gtp-scheduler-script-events
go run ../../cmd/gs run
```

脚本会：

1. 加载 `@plugin/scheduler`。
2. 使用 `scheduler.once("trigger", fn)` 监听定时任务触发事件。
3. 调用 `scheduler.schedule()` 注册一次性任务。
4. 在脚本回调中处理 `event.data.payload`，并写入 `script-event-result.txt`。

事件回调收到的对象形态：

```javascript
{
  id: "evt-task-1-1",
  type: "event",
  module: "@plugin/scheduler",
  event: "trigger",
  data: {
    id: "task-1",
    name: "script-listener-demo",
    payload: { kind: "writeFile", message: "handled in script" },
    fired: 1
  }
}
```
