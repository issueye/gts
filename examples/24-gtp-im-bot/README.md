# GTP IM 机器人插件示例

这个示例展示如何通过 `@plugin/im-bot` 注册 IM 平台适配器。插件自身不调用 cc-connect，只按各平台协议封装发送能力。

运行：

```bash
cd examples/24-gtp-im-bot
go run ../../cmd/gs run
```

发送真实消息前，先在 `main.gs` 中按平台填写凭据和目标会话，再打开 `bots.send(...)`。
