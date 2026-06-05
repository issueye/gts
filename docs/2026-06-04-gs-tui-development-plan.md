# GS TUI 开发计划

## 背景

`gs-agent` 当前已经具备真实 provider、工具注册、会话 JSONL 和流式测试能力。下一步如果要做成可长期使用的 agent，需要提供交互式 TUI，让用户可以在终端内编辑任务、运行 agent、查看模型输出、审计工具调用和读取最终答案。

参考项目是 `E:\codes\github\pi\packages\tui`。`pi` 的 TUI 不是简单打印文本，而是一个完整的终端 UI 框架，核心能力包括 raw mode、输入事件、窗口 resize、ANSI 差量渲染、Bracketed Paste、Kitty 键盘协议、组件系统和可见宽度计算。

本文档用于拆分 `gs` 语言/标准库需要补齐的底层能力，以及 `gs-agent` 可以在脚本层推进的 TUI 实现步骤。

## 目标

第一版目标是实现一个可用于真实 agent 工作流的本地 TUI：

- 在终端内编辑和保存 `workspace/task.txt`。
- 启动一次 agent run，并实时观察状态。
- 展示模型消息、工具调用、工具结果、错误和最终答案。
- 显示 provider、model、maxTurns、工具数量等配置摘要。
- 对 apiKey、Authorization 等敏感字段做脱敏。
- 支持窗口大小变化后重新布局。
- 支持 Ctrl+C 或退出键安全恢复终端状态。

第一版不追求完整替代 `pi` 的 TUI 框架，不做图片协议、鼠标拖拽、多会话数据库、远程协作和复杂主题系统。

## 参考 pi 的能力拆解

`pi` 的 TUI 主要依赖以下能力：

- `process.stdin.setRawMode(true)` 进入 raw mode。
- `process.stdin.on("data", fn)` 接收持续输入事件。
- `process.stdout.write(text)` 直接写 ANSI 序列。
- `process.stdout.on("resize", fn)` 响应窗口尺寸变化。
- `setTimeout`、`setInterval` 做渲染节流、协议协商和 spinner。
- `StdinBuffer` 缓冲被拆开的 escape sequence。
- `keys.ts` 解析普通按键、组合键、Kitty 键盘协议和 key release。
- 差量渲染，减少整屏闪烁。
- ANSI-aware 和 Unicode-aware 的 visible width 计算。
- Windows 下启用 Virtual Terminal Input，避免组合键丢失。

对 `gs` 来说，组件系统、布局、差量渲染、按键解析大部分可以先在脚本层实现；真正必须由语言/标准库提供的是终端原始输入、终端状态恢复、resize 事件和跨平台控制台能力。

## 当前 gs 基础

`gts` 当前已有这些基础，适合作为 TUI 的起点：

- `@std/terminal`
  - `isTTY(name?)`
  - `size(name?)`
  - `read(size?)`
  - `write(text)`
  - `writeln(text?)`
  - `setRawMode(enabled?)`
- `@std/pty`
  - 支持 PTY/ConPTY spawn、write、resize、wait、kill、close。
- `@std/timers`
  - 支持 `setTimeout`、`clearTimeout`、`setInterval`、`clearInterval`、`sleep`。
- `@std/stream`
  - 支持 read、readText、readLine、readAll。

主要缺口是：`terminal.read()` 是同步读取风格，TUI 需要非阻塞输入事件；同时还缺少 resize 事件、终端启动会话对象和更完整的 raw mode 恢复策略。

## 语言和标准库开发计划

### 阶段一：最小可用终端会话

目标：让 gs 脚本可以写一个不会卡死的交互式 TUI。

建议新增 `@std/terminal.start(options)`：

```javascript
let terminal = require("@std/terminal");

let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  onInput: function(data) {
    // data 是原始输入文本，可能包含 ANSI escape sequence
  },
  onResize: function(size) {
    // size.cols / size.rows
  },
});

session.write("\x1b[?25l");
session.stop();
```

建议会话对象提供：

- `write(text)`：写 stdout，不自动换行。
- `writeln(text?)`：写 stdout，自动换行。
- `size()`：返回 `{ cols, rows }`。
- `setRawMode(enabled)`：切换 raw mode。
- `stop()`：恢复 raw mode、光标、Bracketed Paste 等终端状态。
- `drainInput(maxMs?, idleMs?)`：退出前清理残留输入，避免 escape sequence 泄漏到 shell。

验收标准：

- 在 raw mode 下按键能连续触发 `onInput`。
- `setInterval` 渲染不会被输入读取阻塞。
- Ctrl+C 可以由脚本捕获并调用 `session.stop()`。
- 脚本异常退出时尽量恢复终端模式。

### 阶段二：终端事件和控制能力

目标：减少 TUI 层直接拼低级控制逻辑。

建议新增：

- `terminal.onInput(fn)` / `terminal.offInput(fn)`
- `terminal.onResize(fn)` / `terminal.offResize(fn)`
- `terminal.hideCursor()` / `terminal.showCursor()`
- `terminal.clearScreen()`
- `terminal.clearLine()`
- `terminal.clearFromCursor()`
- `terminal.moveTo(row, col)`
- `terminal.moveBy(rows, cols)`
- `terminal.setTitle(title)`
- `terminal.enableBracketedPaste()` / `terminal.disableBracketedPaste()`

这些 API 可以先作为 `terminal.start()` 会话方法提供，后续再决定是否暴露为模块级函数。

验收标准：

- 终端 resize 后 TUI 可以收到事件并重新布局。
- 隐藏光标、显示光标、清屏、移动光标在 Windows Terminal、PowerShell、CMD、Linux/macOS 终端中尽量一致。
- 退出后 shell 中不会残留隐藏光标或 bracketed paste 状态。

### 阶段三：跨平台输入增强

目标：让常用组合键在 Windows 和类 Unix 终端中表现一致。

建议补充：

- Windows 下启用 Virtual Terminal Input。
- raw mode 进入后保留并恢复原始 console mode。
- 支持 Shift+Tab、Ctrl+方向键、Alt+字符等常见输入。
- 提供平台信息，便于脚本层做兼容判断。

验收标准：

- Windows Terminal + PowerShell 下 Shift+Tab 不退化为普通 Tab。
- Ctrl+C 在 raw mode 下不会直接破坏终端状态。
- 连续快速输入不会丢字符。

### 阶段四：测试工具和虚拟终端

目标：让 TUI 可以做自动化回归，而不是只能人工看。

建议提供：

- PTY 测试入口：启动 gs TUI 后写入按键序列，读取屏幕输出。
- 虚拟终端快照工具：把 ANSI 输出还原成屏幕 buffer。
- 标准测试样例：输入、resize、退出、异常恢复。

验收标准：

- 可以自动测试：启动 TUI、输入任务、保存、退出。
- 可以自动测试：resize 后布局没有抛错。
- 可以自动测试：异常后 raw mode 被恢复。

## gs-agent TUI 开发计划

### 阶段一：抽出可复用 agent 装配

目标：TUI 不复制 `runAgentApp()` 的 provider、tools、session 装配逻辑。

建议改造：

- `loadAgentApp(root)`：读取配置、创建 provider、tools、workspace、session。
- `runAgentTask(options)`：传入任务文本、onEvent、是否写 answer 文件。
- `runAgentApp()`：保留现有命令行一次性运行行为。

验收标准：

- `main.gs` 行为不变。
- `smoke-test.gs`、`provider-test.gs`、`stream-test.gs` 继续通过。
- TUI 可以复用同一套配置和工具注册。

### 阶段二：脚本层最小 TUI 框架

目标：先做一个能运行的 TUI 小内核。

建议新增目录：

```text
src/tui/
  ansi.gs
  terminal-session.gs
  input-buffer.gs
  keys.gs
  width.gs
  renderer.gs
  layout.gs
  components.gs
```

职责拆分：

- `ansi.gs`：封装清屏、移动光标、隐藏光标、样式重置。
- `terminal-session.gs`：包装 `@std/terminal`，负责 start/stop。
- `input-buffer.gs`：处理被拆开的 escape sequence。
- `keys.gs`：把原始输入解析为 `ctrl+c`、`tab`、`up` 等 key id。
- `width.gs`：计算中文、emoji、ANSI 样式后的可见宽度。
- `renderer.gs`：根据上一帧和当前帧做差量输出。
- `layout.gs`：根据 cols/rows 计算状态栏、任务区、时间线、详情区。
- `components.gs`：Text、Input、Timeline、Details、StatusBar。

验收标准：

- 能进入 TUI 并退出。
- 能显示四区布局。
- resize 后能重新渲染。
- 输入框支持基础编辑和多行任务。

### 阶段三：接入真实 agent run

目标：TUI 运行真实 agent，而不是 demo。

功能：

- Ctrl+S 保存任务。
- Ctrl+R 启动 agent run。
- 运行时追加 session 事件到时间线。
- 详情区显示选中事件的完整摘要。
- 运行完成后显示 `.agent/answer.md`。
- provider 请求失败时显示错误摘要。

验收标准：

- 使用 DeepSeek Anthropic 兼容配置完成一次真实请求。
- 只读工具任务可以看到 `read_file` / `tool_result`。
- apiKey 不出现在状态栏、详情区、日志或错误提示中。
- 最终答案和 `.agent/answer.md` 一致。

### 阶段四：可用性增强

目标：从“能跑”变成“愿意长期用”。

功能：

- 运行中 spinner 和状态栏。
- 时间线 Up/Down 选择事件。
- 详情区 PageUp/PageDown 滚动。
- 任务区 dirty 标记。
- 退出前未保存提醒。
- 最近 session 加载。
- 工具结果过大时折叠展示。

验收标准：

- 大 session 不会撑爆界面。
- 工具结果截断只影响展示，不影响 JSONL 原始记录。
- 用户可以通过键盘完成一次完整工作流。

### 阶段五：高级输入协议

目标：向 `pi` 的输入体验靠近。

功能：

- Bracketed Paste 支持。
- Kitty keyboard protocol 协商。
- key release 过滤。
- Ctrl/Shift/Alt 组合键增强。
- IME 光标定位标记。

验收标准：

- 粘贴多行任务不会被当成一串散乱按键。
- 中文输入法候选框位置尽量正确。
- 支持 Shift+Tab 反向切换焦点。

## 推荐最小版本范围

第一版可以只实现以下能力：

- 终端 raw mode。
- 非阻塞输入事件。
- stdout 直接写 ANSI。
- resize 事件。
- 简单 key parser。
- 全屏重绘，不做复杂差量渲染。
- 四区布局。
- 真实 agent run。

差量渲染、Kitty 协议、图片协议、鼠标事件可以后置。这样能更快验证 `gs` 语言运行时是否足够承载真实交互。

## 风险和需要观察的问题

- 如果 `terminal.read()` 继续是阻塞模型，TUI 会和 timer、流式输出互相卡住。
- 如果异常退出不能恢复 raw mode，会影响用户 shell，必须优先处理。
- 如果 String 的长度语义和终端列宽不一致，中文布局会错位。
- 如果没有 resize 事件，用户改变窗口大小时只能轮询，体验和性能都较差。
- Windows 组合键差异会影响焦点切换和快捷键体验。
- agent run 如果是长同步调用，TUI 渲染可能被阻塞，需要确认异步任务和回调调度语义。

## 语言侧建议接口清单

优先级从高到低：

```javascript
terminal.start(options)
session.stop()
session.write(text)
session.size()
session.drainInput(maxMs, idleMs)
terminal.onInput(fn)
terminal.onResize(fn)
terminal.hideCursor()
terminal.showCursor()
terminal.clearScreen()
terminal.moveTo(row, col)
terminal.enableBracketedPaste()
terminal.disableBracketedPaste()
```

如果只能先做一个接口，优先做：

```javascript
terminal.start({
  raw: true,
  onInput: fn,
  onResize: fn,
});
```

这个接口一旦稳定，`gs-agent` 就可以开始写真实 TUI。

## 验收路线

### 语言 smoke

- `terminal.start()` 后按键触发回调。
- `setInterval()` 在等待输入时继续刷新计数。
- resize 触发回调。
- Ctrl+C 调用 stop 后 shell 状态正常。

### TUI smoke

- 启动 `tui.gs`。
- 输入一段任务。
- 保存任务。
- 运行 fake provider。
- 时间线出现消息和工具事件。
- 退出后终端恢复。

### 真实接口 smoke

- 使用 `agent.local.toml` 的 DeepSeek Anthropic 兼容配置。
- 运行只读任务，例如读取 README 并总结。
- 时间线显示 `read_file`。
- 最终答案写入 `.agent/answer.md`。
- 页面不泄露 apiKey。

## 第一批落地任务

1. 在 `gts` 中实现 `terminal.start(options)` 的最小版本。
2. 为 `terminal.start()` 添加 raw mode 自动恢复。
3. 增加 resize 事件。
4. 在 `gs-agent` 中抽出 `runAgentTask(options)`。
5. 新增 `src/tui/ansi.gs`、`src/tui/keys.gs`、`src/tui/renderer.gs`。
6. 新增 `tui.gs`，先做全屏重绘版。
7. 使用 fake provider 做 TUI smoke。
8. 再接入 DeepSeek 做真实接口测试。

