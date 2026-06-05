# GoScript 语言侧 Markdown / TUI 支持建议 2026-06-05

本文档只记录为了支撑 `gs-agent` TUI Markdown 渲染，建议 GoScript 语言和标准库侧提供的能力。应用侧具体实现方案见 `docs/plans/2026-06-05-tui-markdown-rendering-plan.md`。

## 背景

`gs-agent` 的 TUI 需要稳定展示：

- agent 最终回答。
- streaming 中的模型输出。
- 工具调用结果。
- `web_fetch` / `web_search` 获取的网页摘要。
- README、计划文档、代码说明等 Markdown 内容。

当前应用层已经手写了部分能力：

- `src/tui/ansi.gs`：ANSI strip、中文宽度、截断、padding。
- `src/tui/components.gs`：轻量 Markdown 行扫描。
- `src/tui/widgets.gs`：宽度感知 wrap。

这些能力继续放在应用层会逐渐变复杂，也容易在不同 TUI 项目里重复实现。因此建议语言侧优先补齐下面能力。

## P0：`@std/text` 终端宽度标准库

### 需求

提供 terminal cell width 感知的文本处理 API：

```javascript
let text = require("@std/text");

text.chars(value);
text.width(value);
text.stripAnsi(value);
text.truncateWidth(value, width);
text.padRightWidth(value, width);
text.wrapWidth(value, width);
```

### 必须支持

- CJK 中文、日文、韩文。
- 全角标点。
- emoji。
- 组合字符。
- ANSI escape sequence。
- 换行。

### 行为建议

```javascript
text.width("你好") === 4;
text.width("\x1b[31m你好\x1b[0m") === 4;
text.truncateWidth("你好a", 4) === "你好";
text.padRightWidth("你", 4) === "你  ";
text.wrapWidth("你好世界", 4); // ["你好", "世界"]
```

### 价值

- 这是 TUI Markdown、表格、列表缩进、代码块、滚动区域的基础。
- 应用层不再重复维护 Unicode 宽度范围。
- 可以统一 Windows Terminal、PowerShell、传统控制台里的显示行为。

## P0：`@std/markdown.parse` Block AST

### 需求

提供标准 Markdown 解析入口：

```javascript
let markdown = require("@std/markdown");

let doc = markdown.parse(source);
```

### 建议 AST

```javascript
{
  type: "document",
  children: [
    {
      type: "heading",
      level: 1,
      children: [{ type: "text", text: "Title" }],
      startLine: 1,
      endLine: 1
    }
  ],
  diagnostics: []
}
```

### P0 Block 类型

- `paragraph`
- `heading`
- `code`
- `list`
- `list_item`
- `blockquote`
- `hr`

### P0 Inline 类型

- `text`
- `strong`
- `em`
- `code`
- `link`
- `softbreak`
- `hardbreak`

### 行为要求

- 非法 Markdown 不应抛错，应尽量返回 AST 和 diagnostics。
- 未闭合 code fence 应作为 code block 返回，并在 diagnostics 中标记。
- 保留 source line 范围，方便 TUI 做滚动定位和 debug。

### 价值

- TUI 可以从“字符串扫描器”升级为结构化渲染器。
- Agent 回答中常见的列表、代码块、引用、链接会稳定很多。
- 后续支持 table、task list、TOC 更自然。

## P1：`@std/markdown.renderTerminal`

### 需求

如果语言侧愿意进一步封装，可提供终端渲染器：

```javascript
markdown.renderTerminal(source, {
  width: 80,
  theme: "dark",
  color: true,
  codeWrap: true,
  links: "inline"
});
```

返回：

```javascript
{
  lines: [],
  width: 80,
  headings: [],
  links: [],
  diagnostics: []
}
```

### 支持范围

- heading 样式。
- paragraph wrap。
- list hanging indent。
- blockquote。
- fenced code block。
- link 展示。
- horizontal rule。

### 价值

- 应用层只负责 viewport、scrollbar、焦点和 screen diff。
- 多个 GoScript TUI 项目可以复用同一套 Markdown 终端渲染。

## P1：Streaming Markdown Parser

### 需求

Agent 模型输出是流式的，Markdown 经常处于半成品状态。建议提供增量接口：

```javascript
let stream = markdown.createStream({
  width: 80,
});

stream.append(delta);
let preview = stream.snapshot();
let final = stream.finalize();
```

### 行为建议

- 未闭合 inline 标记按普通文本显示。
- 未闭合 code fence 按 code block 预览。
- snapshot 不抛错。
- finalize 后返回完整 AST 或 terminal lines。

### 价值

- TUI 可以边接收 token 边刷新，不需要每次完整 parse。
- 可以避免“刚输出 ``` 时整屏闪动”的问题。
- 后续可支持 throttling 和 dirty block 重排。

## P1：代码块轻量高亮

### 需求

提供简单 token 级 highlighter：

```javascript
let highlight = require("@std/highlight");

highlight.terminal(code, {
  lang: "json",
  width: 80,
  theme: "dark"
});
```

### 优先语言

- `json`
- `diff`
- `shell`
- `gs`
- `js`
- `toml`

### 行为建议

- 不需要完整语义分析。
- 解析失败时返回纯文本。
- 输出必须能和 `@std/text.width` 协同。

### 价值

- Agent TUI 中代码块、工具参数、配置片段会更可读。
- 先做轻量规则即可，不需要引入大型高亮体系。

## P1：终端样式 Builder

### 需求

提供统一 ANSI 样式 API：

```javascript
let terminal = require("@std/terminal");

terminal.style("text", {
  bold: true,
  dim: false,
  underline: false,
  inverse: false,
  fg: "cyan",
  bg: ""
});
```

可选链式 API：

```javascript
terminal.text().bold("Title").fg("gray", " muted").toString();
```

### 行为要求

- 支持关闭颜色。
- 支持语义色，例如 `accent`、`muted`、`error`、`success`。
- 样式不影响 `@std/text.width`。
- Windows / 非 TTY / CI 中有降级策略。

### 价值

- 应用层不需要维护 `BOLD`、`RESET`、颜色码和 strip 规则。
- Markdown renderer、TUI 组件、日志输出可共用同一套样式。

## P2：HTML -> Markdown

### 需求

配合 `web_fetch`，提供基础 HTML 到 Markdown 的转换：

```javascript
markdown.fromHTML(html, {
  baseUrl: "https://example.com",
  includeLinks: true,
  maxChars: 20000
});
```

### 支持范围

- 标题。
- 段落。
- 链接。
- 列表。
- 代码块。
- 表格基础转换。
- 去除 script/style/nav。

### 价值

- `web_fetch` 抓到网页后可以直接给 TUI Markdown 渲染。
- Agent 搜索和阅读网页的结果会更干净。

## P2：OSC 8 Hyperlink

### 需求

终端支持时提供可点击链接：

```javascript
terminal.hyperlink("OpenAI", "https://openai.com");
```

### 行为要求

- 支持能力检测或显式开关。
- 不支持时降级为：

```text
OpenAI <https://openai.com>
```

### 价值

- TUI 搜索结果、网页引用、文档链接可以直接打开。

## P2：Table Layout Helper

### 需求

Markdown table 和 TUI 表格都需要宽度分配能力：

```javascript
let table = require("@std/table");

table.layout(rows, {
  width: 100,
  minColWidth: 6,
  wrap: true
});
```

### 价值

- Markdown table、tool result table、配置摘要都能复用。
- 避免每个组件重复写列宽压缩算法。

## 建议优先级

1. `@std/text`：所有 TUI 文本布局的基础。
2. `@std/markdown.parse`：让 Markdown 从字符串扫描升级为结构化渲染。
3. `@std/markdown.renderTerminal`：减少应用层重复劳动。
4. Streaming Markdown：提升 agent 流式体验。
5. 代码块轻量高亮：提升答案和工具结果可读性。
6. 终端样式 Builder：统一颜色和 ANSI 行为。
7. HTML -> Markdown：服务 web 工具。
8. OSC 8 hyperlink / table helper：体验增强。

## 建议测试清单

### `@std/text`

```javascript
assert(text.width("你好") === 4);
assert(text.width("a你b") === 4);
assert(text.width("\x1b[31m红色\x1b[0m") === 4);
assert(text.truncateWidth("你好a", 4) === "你好");
assert(text.wrapWidth("你好世界", 4).length === 2);
```

### `@std/markdown`

```javascript
let doc = markdown.parse("# 标题\n\n- **重点**\n\n```js\nconsole.log(1)\n");
assert(doc.children[0].type === "heading");
assert(doc.children[1].type === "list");
assert(doc.children[2].type === "code");
assert(doc.diagnostics.length >= 1); // 未闭合 code fence
```

### `renderTerminal`

```javascript
let out = markdown.renderTerminal("你好世界", { width: 4 });
assert(out.lines.length === 2);
assert(text.width(out.lines[0]) === 4);
```

## 当前应用层绕过位置

- `src/tui/ansi.gs`：宽度、ANSI、颜色。
- `src/tui/widgets.gs`：wrap、viewport。
- `src/tui/components.gs`：Markdown 轻量扫描。
- `docs/plans/2026-06-05-tui-markdown-rendering-plan.md`：应用侧计划。

语言侧如果先实现 `@std/text` 和 `@std/markdown.parse`，`gs-agent` 就可以明显减少自维护代码，并把 TUI Markdown 做得更稳定。
