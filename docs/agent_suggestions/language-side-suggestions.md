# GoScript 语言侧建议

本文档记录 `gs-agent` 和 TUI 框架开发过程中实际遇到的语言/标准库问题，以及建议优先处理的方向。目标不是追求一次性做完整 JS 兼容，而是优先补齐“真实 agent + TUI 应用”需要的稳定能力。

## P0：嵌入式可执行文件参数边界

现象：

- 旧方式需要 `.\dist\gs-agent.exe -- tui` 才能把参数传给应用。
- 改成产品化命令后，希望直接使用 `.\dist\gs-agent.exe --tui`。
- 如果解释器先解析 `--tui`，会报 `flag provided but not defined: -tui`，应用无法收到参数。

建议：

- 对带嵌入包的可执行文件，解释器只消费已知解释器 flag。
- 遇到第一个未知 `--xxx` 参数时，把该参数及其后的所有参数作为 app argv。
- 普通未嵌入的 `gs` 解释器仍保留未知 flag 报错。

当前处理：

- 已在 `E:\codes\gts\cmd\gs\main.go` 增加 `splitDirectEmbeddedAppArgs`。
- 已补充 `cmd/gs` 级测试覆盖直接 app flag 分流。

建议测试：

- `app.exe --tui` 应进入应用 argv。
- `app.exe --timeout 0 --tui` 应保留解释器 timeout，同时把 `--tui` 给应用。
- `gs --bad-flag` 仍应报解释器未知 flag。

## P0：Unicode 字符宽度和字符串索引

现象：

- TUI 输入中文时，如果按字节长度或普通 `.length` 计算，会出现挤压、错位、截断错误。
- 当前项目必须用 `Array.from()` / `for...of` 做字符遍历。
- `charWidth("你")` 一开始返回 1，导致中文换行不正确；排查后发现十六进制范围判断疑似异常。

建议：

- 标准库提供 `@std/text` 或 `@std/terminal` 下的宽度 API：
  - `text.runes(value)` 或 `text.chars(value)`
  - `text.width(value)`
  - `text.truncateWidth(value, width)`
  - `text.padRightWidth(value, width)`
- 统一处理 CJK、全角标点、emoji、组合字符、ANSI escape 剥离。

当前处理：

- `src/tui/ansi.gs` 中手写了 `chars`、`charWidth`、`visibleWidth`、`truncateToWidth`。
- 十六进制范围已改成十进制范围绕过。

建议测试：

- `charWidth("你") === 2`
- `visibleWidth("你好") === 4`
- `truncateWidth("你好a", 4) === "你好"`
- 带 ANSI 样式的文本宽度不应计算 escape 序列。

## P0：十六进制数字字面量比较

现象：

- `codePointAt(0)` 返回 `20320`，但用 `0x2e80 <= code <= 0xa4cf` 风格范围判断时没有按预期命中。
- 改成十进制常量后，`charWidth("你")` 才正确返回 2。

建议：

- 检查 lexer/parser/evaluator 对 `0x...` 数字字面量的解析、类型和比较行为。
- 确保十六进制字面量与十进制 number 在 `>=`、`<=`、算术中一致。

建议测试：

```javascript
assert(0x2e80 === 11904);
assert(20320 >= 0x2e80);
assert(20320 <= 0xa4cf);
```

## P1：模块转导语法

现象：

- GoScript 当前不支持：

```javascript
export { runTuiApp } from "@/tui/runtime";
```

- `src/tui/framework.gs` 只能先显式 import 再 export，代码比较啰嗦。

建议：

- 支持 ES module 常见转导语法：
  - `export { name } from "module"`
  - `export { name as alias } from "module"`
  - `export * from "module"`

当前处理：

- `src/tui/framework.gs` 使用显式 import + export 聚合。

价值：

- 组件库和标准库聚合入口会更自然。
- 有助于形成稳定的 `@std/tui` 或项目级 framework facade。

## P1：异步模型和 TUI 主循环

现象：

- 当前 TUI runtime 依靠 `terminal.start({ onInput, onResize })` 回调和 `timers.sleep(50)` 保持主循环。
- agent run 仍是同步闭环，长请求期间 UI 的细粒度响应能力有限。

建议：

- 明确 GoScript 的异步任务模型：
  - 是否支持 promise/await。
  - 是否支持后台 task/worker。
  - 是否允许主线程 TUI 和模型请求并行推进。
- 标准库提供可取消任务：
  - `task.spawn(fn)`
  - `task.cancel(handle)`
  - `task.sleep(ms)`

短期建议：

- 保持当前同步模型可用。
- 先为 HTTP streaming、TUI tick、用户中断留出 API 设计空间。

## P1：终端标准库继续产品化

现状：

- 当前已有 `terminal.start(options)`、raw input、bracketed paste、resize、session write、setTitle、drainInput。
- 这已经足够支撑第一版 TUI。

建议继续补齐：

- `session.clearScreen()`
- `session.clearLine()`
- `session.moveTo(row, col)`
- `session.hideCursor()` / `session.showCursor()`
- `session.enterAlternateScreen()` / `session.leaveAlternateScreen()`
- `session.enableMouse()` / `session.disableMouse()`
- `session.onFocus()` 可后置

价值：

- 现在这些能力由 `src/tui/ansi.gs` 手写 escape 序列。
- 标准库封装后跨平台行为更可控，Windows Terminal、传统控制台、CI 非 TTY 可以统一处理。

## P1：Bracketed Paste 和按键协议

现象：

- 当前 `src/tui/keys.gs` 只解析基础方向键、PageUp/PageDown、Tab、Ctrl 键和 bracketed paste。
- Claude Code 类 TUI 还需要更完整的键盘协议。

建议：

- 标准库或 `@std/terminal/keys` 提供统一 key event：

```javascript
{
  id: "ctrl+r",
  text: "",
  alt: false,
  ctrl: true,
  shift: false,
  paste: false
}
```

- 后续支持：
  - Home / End
  - Delete
  - Ctrl+Left / Ctrl+Right
  - Alt 组合键
  - Kitty keyboard protocol

## P2：文件系统和日志便利能力

现状：

- 当前项目使用 `fs.appendTextSync` 写 JSONL 日志。
- 可用，但应用层需要自己处理目录、latest 文件、日志 JSON 格式。

建议：

- 增加 `@std/log` 或基础 logger：
  - JSONL 输出
  - scope/level/time
  - 自动创建目录
  - 简单 rotate

价值：

- agent/TUI 长运行任务需要可靠日志。
- 避免每个项目都重复写 logger。

## P2：Markdown/ANSI 渲染基础库

现状：

- `src/tui/components.gs` 里实现了最小 Markdown 渲染：
  - heading
  - list
  - code fence
  - bold
  - inline code

建议：

- 后续考虑 `@std/markdown` 或 `@std/tui/markdown`：
  - Markdown block parser
  - ANSI style renderer
  - 宽度感知换行
  - 代码块语言标识保留

价值：

- agent 最终回答天然是 Markdown。
- TUI 中稳定展示 Markdown 是核心体验。

## P2：类型/语法开发体验

建议：

- 对不支持的语法给出更明确错误。例如 `export { x } from` 当前报 `no prefix parser for FROM`，可以提示“export-from syntax is not supported”。
- 对 `undefined`、缺失字段、对象 `in` 操作保持稳定语义。
- 对脚本错误保留更清晰的源码位置，尤其是打包后 `exe!path:line:col` 已经很有用，建议继续保持。

## 建议优先级

1. 嵌入 exe app 参数边界：直接影响产品命令。
2. Unicode/宽度 API：直接影响 TUI 是否可用。
3. 十六进制字面量比较：属于语言正确性问题。
4. 模块转导语法：影响框架组织和标准库 facade。
5. 异步/可取消任务：影响 streaming、长任务和 TUI 中断。
6. 终端 API 产品化：减少脚本层 ANSI 细节。
7. Markdown/日志标准库：提升 agent 应用开发效率。

## 当前项目中的绕过位置

- `src/tui/ansi.gs`：手写 Unicode 宽度、ANSI strip、padding。
- `src/tui/framework.gs`：显式 import 再 export，绕过 export-from。
- `src/tui/runtime.gs`：用 `timers.sleep(50)` 维持 TUI 主循环。
- `src/agent/log.gs`：项目内自带 JSONL logger。
- `main.gs`：只识别 `--tui` 作为 TUI 入口。
