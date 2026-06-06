# @std/terminal

> 宿主终端探测、会话生命周期、raw mode、resize 事件和 TUI 渲染辅助。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/terminal` | 原生终端模块 |

## 加载

```javascript
let terminal = require("@std/terminal");
```

## 基础接口

| 接口 | 说明 |
|------|------|
| `isTTY(fd?)` | 判断 `stdin`、`stdout` 或 `stderr` 是否终端，默认 `stdout` |
| `size(fd?)` | 返回 `{ cols, rows }`，默认读取 `stdout` 尺寸 |
| `capabilities()` | 返回 `{ clearScrollback, alternateScreen, resizeEvents, virtualTerminal, rawMode }` |
| `read(size?)` | 从 stdin 读取文本 |
| `write(text)` / `writeln(text?)` | 写入 stdout |
| `setRawMode(enabled?)` | 开启 raw mode，返回带 `restore()` 的对象 |

## 会话

```javascript
let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  mouse: false,
  alternateScreen: true,
  hideCursor: true,
  restoreOnExit: true,
  restoreOnError: true,
  resizeDebounceMs: 50,
  onInput: function(data) {},
  onResize: function(size) {},
  onError: function(error, session) {},
});
```

会话会记录已启用的 raw mode、Bracketed Paste、鼠标、备用屏和隐藏光标状态，并在 `session.stop()`、VM 结束或 callback 错误时按配置恢复。`session.restore()` 可重复调用。

| 会话接口 | 说明 |
|----------|------|
| `session.write(text)` / `session.writeln(text?)` | 写入终端 |
| `session.size()` | 返回当前尺寸 |
| `session.setRawMode(enabled)` | 切换 raw mode |
| `session.restore()` | 恢复会话托管的终端模式 |
| `session.stop()` | 停止事件循环并按 `restoreOnExit` 恢复 |
| `session.drainInput(maxMs?, idleMs?)` | 清理输入事件队列 |
| `session.clear(options?)` | 清理屏幕，可显式清 scrollback |
| `session.renderFrame(frame, options?)` | 裁剪并渲染一帧 TUI |
| `session.redraw(fn)` | 合并回调内的 screen 操作后一次写入 |

## 清屏策略

```javascript
session.clear({ screen: true, scrollback: false });
session.clear({ screen: true, scrollback: true });
```

默认只清理可见屏幕，不清滚动历史。`scrollback: true` 会追加 `CSI 3J`，必须由调用方显式选择。

## 帧渲染

```javascript
session.renderFrame(lines.join("\n"), {
  rows: size.rows,
  cols: size.cols,
  clip: true,
  diff: true,
  full: false,
});
```

`renderFrame` 会裁剪超过 viewport 的行和列，full render 时先清可见屏幕并移动到左上角。开启 `diff` 后，会话保存上一帧，只重写变化行；resize 后应用可传 `full: true` 强制完整重绘。

## 原子 redraw

```javascript
session.redraw(function(screen) {
  screen.clear({ screen: true });
  screen.moveTo(1, 1);
  screen.write(frame);
});
```

回调中的 `screen.clear()`、`screen.moveTo()`、`screen.write()` 会先合并成一个输出，再一次写入终端，减少清屏和重绘之间的中间态。

## Resize

`terminal.start({ onResize, resizeDebounceMs })` 会轮询终端尺寸并防抖，窗口拖拽过程中合并中间尺寸，稳定后触发回调：

```javascript
function onResize(size) {
  // size.cols / size.rows / size.stable
}
```

## 维护来源

- `internal/stdlib/api_docs.go`
- `internal/stdlib/terminal.go`
- 相关示例：`scripts/agent/smoke_terminal.gs`
