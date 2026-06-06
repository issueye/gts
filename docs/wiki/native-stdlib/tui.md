# @std/tui

> 提供给脚本的轻量原生 TUI 框架：脚本负责 `init/update/view`，原生层负责终端事件、全屏生命周期、resize、diff 渲染和恢复。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/tui` | 脚本侧 TUI 状态机与布局辅助 |

## 加载

```javascript
let tui = require("@std/tui");
```

## App 状态机

```javascript
let app = tui.createApp({
  init: function(size) {
    return { count: 0, cols: size.cols };
  },
  update: function(state, msg) {
    if (msg.type === "text") {
      state.count = state.count + 1;
    }
    if (msg.type === "key" && msg.key === "ctrl+c") {
      return { state: state, quit: true };
    }
    return state;
  },
  view: function(state, size) {
    return tui.box("count=" + String(state.count), {
      title: "Counter",
      width: size.cols,
    });
  },
});

app.run();
```

`update(state, msg)` 可以直接返回新 state，也可以返回 `{ state, quit: true }`。`quit: true` 会让 `run()` 退出并恢复终端。

## App 方法

| 接口 | 说明 |
|------|------|
| `createApp(spec)` | 创建 TUI app，`spec` 支持 `init`、`update`、`view`、`state` |
| `app.dispatch(msg)` | 同步派发消息，返回最新 state；适合测试或非交互脚本 |
| `app.render(size?)` | 调用 `view(state, size)` 并返回帧文本 |
| `app.run(options?)` | 进入全屏事件循环，处理输入、resize、tick 并渲染 |
| `app.stop()` | 请求停止运行并恢复终端 |
| `app.state()` | 返回当前 state |

## 消息

| 构造器 | 结果 |
|--------|------|
| `key(name, raw?)` | `{ type: "key", key, raw? }` |
| `text(value)` | `{ type: "text", text: value, raw: value }` |
| `resize(cols, rows)` | `{ type: "resize", cols, rows, stable: true }` |
| `tick()` | `{ type: "tick", timeMs }` |

`run()` 会自动把终端输入解析为 `key`、`text`、`mouse`、`resize`、`tick` 或 `raw` 消息。

## 运行选项

```javascript
app.run({
  raw: true,
  alternateScreen: true,
  hideCursor: true,
  mouse: false,
  bracketedPaste: false,
  diff: true,
  clip: true,
  full: false,
  tickMs: 0,
  resizeDebounceMs: 50,
});
```

默认启用 raw mode、备用屏、隐藏光标、diff 渲染和裁剪；`stop()`、退出或 callback 错误时会恢复终端状态。

## 布局辅助

| 接口 | 说明 |
|------|------|
| `box(content, options?)` | 渲染面板，支持 `title`、`width`、`height`、`padding`、`border` |
| `input(options)` | 渲染单行输入框，支持 `title`、`value`、`cursor`、`placeholder`、`prompt`、`width`、`focused`、`meta` |
| `row(parts...)` | 水平拼接多块多行文本 |
| `column(parts...)` | 垂直拼接多块文本 |
| `pad(content, padding?)` | 给内容加内边距 |
| `statusBar({ left?, center?, right? }, width)` | 渲染单行状态栏 |
| `style(text, options?)` | 返回 ANSI 样式文本，参数同 `@std/terminal.style` |
| `width(value)` / `truncate(value, width)` / `stripAnsi(value)` | 显示宽度工具 |

## 维护来源

- `internal/stdlib/tui.go`
- `internal/stdlib/api_docs.go`
- 相关示例：`examples/23-native-tui.gs`
