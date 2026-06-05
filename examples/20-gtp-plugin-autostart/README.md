# GTP 插件自动唤醒示例

这个示例展示 `gs run` 启动项目时读取 `project.toml` 的 `[plugins.scheduler]` 配置，并自动启动 `cmd/gtp-scheduler` 插件进程。

运行：

```bash
cd examples/20-gtp-plugin-autostart
go run ../../cmd/gs run
```

期望输出：

```text
GTP scheduler plugin auto-started
```

