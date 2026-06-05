# GoScript 语言侧开发计划

来源：`docs/language-side-suggestions.md`

## 目标

优先补齐真实 agent 和 TUI 应用已经踩到的语言/标准库能力，不追求一次性完整 JS 兼容。计划按“会阻塞产品命令/会影响 TUI 可用性/会改善框架组织”的顺序推进。

## P0：产品命令参数边界

状态：已完成，需持续回归。

- 嵌入包可执行文件只消费已知解释器 flag。
- 首个未知 `--xxx` 及其后参数交给 app argv。
- 普通 `gs --bad-flag` 继续报解释器未知 flag。
- 回归测试保留在 `cmd/gs`。

## P0：数字字面量正确性

状态：本轮开始开发。

- 修正 `0x`、`0b`、`0o` 数字字面量解析，保证参与比较和算术时与十进制 number 一致。
- 覆盖建议断言：
  - `0x2e80 === 11904`
  - `20320 >= 0x2e80`
  - `20320 <= 0xa4cf`

## P0：Unicode/终端文本宽度

状态：本轮开始开发。

- 新增 `@std/text`，先提供 TUI 可用的核心 API：
  - `chars(value)` / `runes(value)`
  - `width(value)`
  - `truncateWidth(value, width)`
  - `padRightWidth(value, width)`
  - `stripAnsi(value)`
- 第一版覆盖 CJK、全角标点、emoji 宽度、组合字符宽度、ANSI escape 剥离。
- 后续可把内部实现替换为更完整的 grapheme cluster/Unicode width 数据表，保持 API 不变。

## P1：模块转导语法

状态：待开发。

- AST 为 `ExportDecl` 增加 `Source` 和 `ExportAll`。
- Parser 支持：
  - `export { name } from "module"`
  - `export { name as alias } from "module"`
  - `export * from "module"`
- Evaluator 从目标模块读取导出并写入当前 `exports`。
- 增加 parser/evaluator/CLI 模块缓存回归测试。

## P1：异步/可取消任务

状态：设计预研。

- 复用现有 VM async 机制，设计 `@std/task`：
  - `spawn(fn)`
  - `cancel(handle)`
  - `sleep(ms)`
- 先明确取消语义和 TUI/HTTP streaming 的交互，再进入实现。

## P1：终端标准库产品化

状态：大部分已满足，需补测试和缺口。

- `@std/terminal` 已有 `clearScreen`、`clearLine`、`moveTo`、`hideCursor`、`showCursor`、bracketed paste、session API。
- 后续补 `enterAlternateScreen` / `leaveAlternateScreen`、mouse、focus，并增加非 TTY 行为测试。

## P1：按键协议

状态：待开发。

- 新增 `@std/terminal/keys` 或并入 `@std/terminal`：
  - 统一 key event 结构。
  - 支持 Home/End/Delete、Ctrl+Left/Right、Alt 组合键。
  - Kitty keyboard protocol 后置。

## P2：日志和 Markdown

状态：部分已满足。

- `@std/log` 已提供文件 logger、JSON、level、rotate 基础能力，后续补应用层示例和更多回归。
- Markdown/ANSI 渲染建议后置为 `@std/markdown` 或 `@std/tui/markdown`，优先服务 agent 最终回答展示。

## 本轮交付范围

- 修复数字字面量解析。
- 新增 `@std/text` 基础模块和 API 文档。
- 增加 P0 测试。
