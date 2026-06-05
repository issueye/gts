# GTP Scheduler Client 示例

这个示例启动 `cmd/gtp-scheduler` 作为独立进程，通过 GTP JSON Lines 协议创建一个一次性定时任务，并接收任务触发事件。

运行：

```bash
go run ./examples/19-gtp-scheduler-client
```

期望输出：

```text
ready: service=scheduler ...
scheduled: ...
event after 100ms: trigger ...
```

