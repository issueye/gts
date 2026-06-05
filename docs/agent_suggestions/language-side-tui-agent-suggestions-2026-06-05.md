# GoScript Agent TUI 语言侧建议 2026-06-05

本文档基于 `gs-agent` 开发真实 TUI agent 的最新问题整理，重点记录当前语言/标准库侧如果继续增强，最能减少应用层绕路的方向。

## 背景

当前 `gs-agent` 已经具备：

- `--tui` 直接启动入口。
- 可局部刷新的 TUI 渲染器。
- `Input`、`Box`、`Container`、`Spacer`、`Markdown`、`Loading`、`Text` 等基础组件。
- 中文宽度处理、鼠标滚动、内容区域滚动条、日志输出。
- 异常退出时应用层恢复光标、鼠标模式和备用屏。

但在真实 TUI agent 场景下，仍有一些能力更适合放到 GoScript 语言/标准库侧统一处理。

## P0：终端会话恢复必须由标准库兜底

### 现象

TUI 开启 mouse tracking、raw mode、bracketed paste、备用屏后，如果渲染、输入回调或异步事件中发生异常，终端可能残留在异常状态：

- 鼠标事件直接打印为 `\x1b[<...M`。
- 退出后无法正常输入。
- 光标隐藏。
- 备用屏未退出。

当前 `gs-agent` 已在 `src/tui/runtime.gs` 应用层补了 `try/finally` 和回调异常边界，但这仍然依赖每个 TUI 应用自己写对。

### 建议

在 `@std/terminal` 中增加统一生命周期能力：

```javascript
let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  mouse: true,
  alternateScreen: true,
  hideCursor: true,
  restoreOnError: true,
  restoreOnExit: true,
});

session.restore();
```

建议 `session.restore()` 一次性恢复：

- raw mode。
- bracketed paste。
- mouse tracking。
- cursor visible。
- alternate screen。
- 其他后续新增的终端模式。

`restore()` 必须可重复调用，多次调用不应报错。

### 建议测试

```javascript
let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  mouse: true,
  alternateScreen: true,
  hideCursor: true,
});
throw new Error("boom");
```

进程退出前应恢复终端，不应残留鼠标模式或隐藏光标。

## P0：异步终端回调错误应触发恢复

### 现象

`E:\codes\gts\internal\stdlib\terminal.go` 的 `eventLoop()` 当前在 `onInput` / `onResize` 回调返回 runtime error 时，主要行为是打印到 stderr。

真实 TUI 中，很多错误会发生在输入回调里：

- 解析按键异常。
- 渲染异常。
- 状态更新异常。
- 组件代码异常。

如果标准库只打印错误，脚本主循环未必能执行自己的 `finally`，终端恢复就不可靠。

### 建议

`terminal.start()` 增加错误处理策略：

```javascript
terminal.start({
  onInput: function(data) {},
  onResize: function(size) {},
  onError: function(error, session) {
    session.restore();
    session.stop();
  },
});
```

默认策略建议：

- 回调发生 runtime error 时调用 `session.restore()`。
- 停止当前 session。
- 将错误传给 `onError`。
- 如果没有 `onError`，继续输出清晰错误信息。

### 建议测试

```javascript
terminal.start({
  raw: true,
  mouse: true,
  alternateScreen: true,
  onInput: function(data) {
    throw new Error("input failed");
  },
});
```

触发输入后，标准库应恢复终端状态。

## P0：解释器/VM 退出时统一停止活动终端会话

### 现象

应用层异常、解释器 runtime error、未来的 Ctrl+C / signal 中断，都可能绕过脚本层 cleanup。

### 建议

解释器在 VM 退出路径中统一调用：

```go
stdlib.StopTerminalSessionsForVM(vm)
```

建议覆盖：

- 顶层 runtime error。
- async callback runtime error。
- timeout。
- Ctrl+C。
- panic recover。
- 嵌入式 exe 应用退出。

这可以作为终端恢复的最后一道兜底。

## P1：标准库提供终端模式 API

当前项目仍在 `src/tui/ansi.gs` 中手写 escape sequence：

- `enterAlternateScreen()`
- `leaveAlternateScreen()`
- `enableMouse()`
- `disableMouse()`
- `hideCursor()`
- `showCursor()`
- `clearScreen()`
- `moveTo()`

建议 `@std/terminal` 原生提供：

```javascript
session.enterAlternateScreen();
session.leaveAlternateScreen();
session.enableMouse();
session.disableMouse();
session.hideCursor();
session.showCursor();
session.clearScreen();
session.clearLine();
session.moveTo(row, col);
```

价值：

- 应用层不需要关心不同终端差异。
- Windows Terminal 和传统控制台可以由标准库统一兼容。
- 标准库能记录哪些模式已经开启，恢复时更安全。

## P1：文本宽度与 ANSI 感知能力

### 现象

TUI 渲染中文、全角符号、emoji、ANSI 颜色时，必须按“终端显示列宽”计算，而不是按字符串长度。

当前项目在 `src/tui/ansi.gs` 中手写：

- `chars()`
- `charWidth()`
- `visibleWidth()`
- `truncateToWidth()`
- `padRight()`
- `stripAnsi()`

### 建议

新增 `@std/text` 或 `@std/terminal/text`：

```javascript
text.chars(value);
text.width(value);
text.stripAnsi(value);
text.truncateWidth(value, width);
text.padRightWidth(value, width);
text.wrapWidth(value, width);
```

需要处理：

- CJK。
- 全角标点。
- emoji。
- 组合字符。
- ANSI escape。
- 换行。

### 建议测试

```javascript
assert(text.width("你好") === 4);
assert(text.width("\x1b[31m你好\x1b[0m") === 4);
assert(text.truncateWidth("你好a", 4) === "你好");
assert(text.padRightWidth("你", 4) === "你  ");
```

## P1：任务与异步模型需要更适合 TUI

### 现象

当前 TUI runtime 使用短 `sleep` 维持主循环：

```javascript
while (!shouldExit(state)) {
  timers.sleep(50);
}
```

这能工作，但真实 agent 需要同时处理：

- 用户输入。
- 加载动画。
- LLM streaming。
- 工具调用进度。
- 用户中断。
- 日志写入。

### 建议

提供可取消任务 API：

```javascript
let task = tasks.spawn(function(signal) {
  // long running work
});

task.cancel();
task.done();
```

或明确 Promise/async/await 的调度模型，让 TUI 可以自然等待 streaming，同时继续刷新 UI。

### 最小可用目标

- 主线程能继续接收输入。
- 后台任务能推送事件。
- 任务可以取消。
- 异常能回到统一错误处理。

## P1：HTTP Streaming 标准化

真实 agent 的 TUI 体验依赖 token 级增量输出。

建议 HTTP 标准库提供稳定 streaming API：

```javascript
let stream = http.stream(url, options);
for (let chunk of stream) {
  // update tui
}
```

或：

```javascript
http.request({
  stream: true,
  onChunk: function(chunk) {},
  onError: function(error) {},
  onDone: function() {},
});
```

需要支持：

- SSE。
- chunked response。
- 超时。
- 取消。
- 错误事件。
- response headers。

## P2：模块转导语法

当前 `src/tui/framework.gs` 需要显式 import 后再 export，因为还不能写：

```javascript
export { runTuiApp } from "@/tui/runtime";
export * from "@/tui/components";
```

建议支持：

- `export { name } from "module"`
- `export { name as alias } from "module"`
- `export * from "module"`

这对组件库、标准库 facade 和真实项目组织很有帮助。

## P2：键盘和鼠标事件标准化

当前项目在 `src/tui/keys.gs` 中解析基础键盘、bracketed paste 和 SGR mouse。

建议标准库输出统一事件：

```javascript
{
  type: "key",
  id: "ctrl+r",
  text: "",
  ctrl: true,
  alt: false,
  shift: false,
  paste: false
}
```

鼠标事件：

```javascript
{
  type: "mouse",
  action: "wheelDown",
  row: 10,
  col: 20,
  button: 0
}
```

后续可以扩展：

- Home / End / Delete。
- Ctrl+Left / Ctrl+Right。
- Alt 组合键。
- Focus in/out。
- Kitty keyboard protocol。

## P2：日志标准库

Agent 运行需要可靠日志。当前项目已封装 `src/agent/log.gs`，但建议标准库提供基础能力：

```javascript
let logger = log.create({
  dir: ".agent/logs",
  format: "jsonl",
  level: "debug",
});

logger.info("agent.start", { model: "deepseek-v4-flash" });
logger.error("agent.error", { error: String(err) });
```

建议支持：

- 自动创建目录。
- JSONL。
- level。
- scope。
- rotate。
- latest 指针文件。

## 建议优先级

1. 终端会话恢复兜底：直接影响 TUI 是否会破坏用户终端。
2. 异步终端回调错误恢复：覆盖最容易出错的输入/resize 路径。
3. VM 退出统一停止终端会话：作为最终安全网。
4. 终端模式 API：减少应用层 ANSI 细节。
5. 文本宽度 API：决定中文 TUI 的稳定性。
6. 任务/异步模型：决定 agent streaming 和中断体验。
7. HTTP streaming：决定真实模型输出体验。
8. 模块转导、键鼠事件、日志：提升框架化和工程体验。

## 当前应用层绕过位置

- `src/tui/runtime.gs`：应用层维护终端生命周期、异常恢复、主循环。
- `src/tui/ansi.gs`：应用层手写 ANSI、中文宽度、颜色、截断。
- `src/tui/keys.gs`：应用层手写键盘和鼠标协议解析。
- `src/tui/framework.gs`：显式 import/export 聚合，绕过模块转导语法。
- `src/agent/log.gs`：应用层自带 JSONL 日志。

## 期望完成后的效果

如果以上 P0/P1 能在语言侧落地，`gs-agent` 的 TUI runtime 可以明显简化：

- 不再手写大部分终端恢复逻辑。
- 异常退出不会破坏终端。
- 中文和 ANSI 渲染更可靠。
- LLM streaming 时 TUI 仍能流畅响应输入。
- TUI 组件库可以更接近通用框架，而不是每个项目重复造基础设施。
