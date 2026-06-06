# GTP Scheduler Event Handler 示例

这个示例展示宿主如何处理插件主动推送的消息：

1. 宿主启动 `cmd/gtp-scheduler` 插件。
2. 宿主调用 `@plugin/scheduler.schedule()` 注册一个定时任务。
3. scheduler 到点后只发送 `trigger` 事件和任务 payload。
4. 宿主监听插件事件，并根据 payload 执行自己的处理逻辑。

运行：

```bash
go run ./examples/21-gtp-scheduler-event-handler
```

预期输出包含：

```text
scheduled task:
received scheduler event:
host executor:
```
