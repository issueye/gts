# GTP IM 机器人插件

`plugins/im-bot` 是一个独立 Go 项目 GTP 插件，模块名为 `@plugin/im-bot`。它只依赖公开的 `github.com/issueye/goscript/sdk/gtp` 协议 SDK，不依赖解释器、解析器或 `internal` 运行时实现；也不依赖 `cc-connect` 运行，而是参考各 IM 平台的接入方式，按当前项目的 GTP 插件机制封装成统一脚本 API。

## 支持的适配器

| 平台 | 接入方式 | 适配器名 |
|------|----------|----------|
| 飞书/Lark | OpenAPI：tenant access token + `im/v1/messages` 发消息 | `feishu` / `lark` |
| QQ/NapCat | OneBot v11 HTTP API：`send_private_msg` / `send_group_msg` | `onebot` / `qq` |
| QQ 官方机器人 | QQ Bot API v2：AppAccessToken + `/v2/groups|users/.../messages` | `qqbot` |
| 微信个人号 | ilink HTTP 网关：`sendmessage`，依赖会话 `context_token` | `weixin` |

当前版本先实现发送侧。接收侧建议后续按平台长连接/回调转换为 GTP `event`：

- 飞书：WebSocket 长连接或 HTTP webhook。
- OneBot：正向 WebSocket 或 HTTP post event。
- QQBot：官方 Gateway WebSocket。
- Weixin：ilink `getupdates` 长轮询。

## 普通配置文件

在项目根目录的 `config.toml` 中配置插件：

```toml
[plugins.im_bot]
command = "go"
args = ["run", "."]
cwd = "../../plugins/im-bot"
modules = ["@plugin/im-bot"]
capabilities = ["call"]
```

## 脚本 API

```javascript
const bots = require("@plugin/im-bot");

bots.configure({
  name: "feishu-main",
  platform: "feishu",
  appId: "cli_xxx",
  appSecret: "secret"
});

bots.send({
  adapter: "feishu-main",
  to: "oc_xxx",
  toType: "chat_id",
  text: "GTS 脚本消息"
});
```

## 配置示例

飞书：

```javascript
bots.configure({
  name: "fs",
  platform: "feishu",
  appId: "cli_xxx",
  appSecret: "xxx",
  baseUrl: "https://open.feishu.cn"
});
```

OneBot/NapCat：

```javascript
bots.configure({
  name: "qq",
  platform: "onebot",
  baseUrl: "http://127.0.0.1:3000",
  token: "optional"
});

bots.send({ adapter: "qq", toType: "group", to: "123456", text: "群消息" });
```

QQ 官方机器人：

```javascript
bots.configure({
  name: "qqbot",
  platform: "qqbot",
  appId: "1020...",
  appSecret: "xxx",
  sandbox: false
});

bots.send({ adapter: "qqbot", toType: "group", to: "group_openid", text: "消息" });
```

微信个人号 ilink：

```javascript
bots.configure({
  name: "wx",
  platform: "weixin",
  token: "Bearer token without prefix",
  baseUrl: "https://ilinkai.weixin.qq.com"
});

bots.send({
  adapter: "wx",
  to: "user@im.wechat",
  text: "微信消息",
  extra: { contextToken: "context_token_from_inbound_message" }
});
```

## 方法

| 方法 | 说明 |
|------|------|
| `configure(options)` | 注册或覆盖一个适配器 |
| `list()` | 列出已配置适配器，不返回密钥明文 |
| `send(options)` | 通过指定适配器发送文本消息 |

`send` 字段：

| 字段 | 说明 |
|------|------|
| `adapter` | `configure.name` |
| `to` | 接收者 ID；OneBot 可用 `group:群号` 前缀 |
| `toType` | `private` / `group` / `chat_id` / `open_id` 等，按平台解释 |
| `text` | 发送文本 |
| `extra` | 平台扩展参数；微信目前使用 `contextToken` |
