# GoScript 终端 Resize 与 TUI 普通屏幕模式语言侧支持建议 2026-06-06

本文档基于 `gs-agent` TUI 在 Windows Terminal / PowerShell 中调整窗口大小后出现界面重复、残留、滚动错位的问题整理。重点说明：为什么单靠应用层 ANSI 重绘不够稳定，以及 GoScript 语言/标准库侧建议补充哪些能力。

## 问题背景

`gs-agent` 的 TUI 当前使用普通屏幕模式运行：

```javascript
runTuiApp({
  alternateScreen: false,
});
```

这样做的好处是退出后用户仍能在主终端滚动历史中看到运行过程。但在普通屏幕模式下，窗口 resize 后终端会对主屏幕缓冲区和 scrollback 做 reflow。应用层随后再执行清屏和重绘时，旧 UI 可能已经被终端推入滚动区域，导致视觉上出现两份界面：

- 上方残留旧的 header / transcript / prompt。
- 下方出现新的完整 TUI。
- 继续输入或 resize 后，残留内容可能进一步错位。

这类问题在备用屏模式下通常不会出现，因为备用屏拥有独立缓冲区，不依赖主终端 scrollback。

## 当前应用层实现

当前 `src/tui/screen.gs` 的局部刷新器主要做：

- 保存上一帧屏幕行。
- 普通输入时只重写变化行。
- 尺寸变化时触发 full render。
- full render 时输出 `clearScreen()` 和 `hideCursor()`。

当前 `clearScreen()` 是：

```javascript
"\x1b[2J\x1b[H"
```

这只能清理当前可见屏幕并移动光标到左上角，不能可靠控制主屏幕 scrollback，也不能阻止 Windows Terminal 在 resize 时先对旧内容做 reflow。

应用层已经做过的缓解：

- 修正 `renderFrame()` 的高度预算，避免写出窗口高度。
- 局部刷新器裁剪超过 viewport 的行，避免写到最后一行后触发滚动。

这些能减少“应用自己写多一行导致滚动”的问题，但不能完全解决普通屏幕模式下 resize reflow 造成的历史残留。

## 结论

如果目标是稳定、产品级 TUI，语言侧需要提供更完整的终端会话能力。应用层可以继续写 ANSI 作为短期补丁，但无法跨终端可靠处理以下问题：

- resize 期间主屏幕缓冲区 reflow。
- scrollback 清理策略。
- 光标位置和 viewport 的原子重置。
- resize 事件防抖和最终尺寸确认。
- 多个 ANSI 操作和重绘之间的竞态。
- Windows 控制台与类 Unix 终端差异。

## 短期建议：默认使用备用屏

对全屏 TUI，建议语言侧和应用侧优先使用备用屏：

```javascript
let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  alternateScreen: true,
  hideCursor: true,
  restoreOnExit: true,
  restoreOnError: true,
});
```

备用屏的优势：

- resize 后不会污染主终端 scrollback。
- 清屏只影响备用屏缓冲区。
- 退出后主终端历史仍保持启动 TUI 前的状态。
- 更符合全屏 TUI 的常见行为。

如果需要保留运行摘要，建议退出备用屏后打印一段最终摘要，而不是让整个 TUI 常驻主屏幕历史。

## P0：终端会话生命周期由标准库托管

建议 `@std/terminal.start()` 完整接管这些模式：

```javascript
let session = terminal.start({
  raw: true,
  bracketedPaste: true,
  mouse: false,
  alternateScreen: true,
  hideCursor: true,
  restoreOnExit: true,
  restoreOnError: true,
});
```

标准库应记录当前 session 开启过哪些模式，并在以下路径统一恢复：

- 正常 `session.stop()`。
- 脚本顶层 runtime error。
- `onInput` / `onResize` callback runtime error。
- timeout。
- Ctrl+C / 中断信号。
- VM 退出。
- panic recover。

建议 `session.restore()` 可重复调用，多次调用不报错。

## P0：提供原子化 redraw API

应用层当前是手写：

```javascript
session.write(clearScreen() + hideCursor());
session.write(frame);
```

建议语言侧提供：

```javascript
session.redraw(function(screen) {
  screen.clear({ scrollback: false });
  screen.moveTo(1, 1);
  screen.write(frame);
});
```

或更简单：

```javascript
session.renderFrame(frame, {
  rows: size.rows,
  cols: size.cols,
  full: true,
  clip: true,
});
```

语义要求：

- 一次 redraw 内部尽量合并输出，减少中间态。
- 自动裁剪超过 viewport 的行。
- 必要时避免在最后一行写入会触发滚动的换行。
- resize 后第一帧强制 full render。
- full render 之前重置光标和清理可见区域。

## P0：普通屏幕模式需要 scrollback 策略

如果应用明确选择普通屏幕模式：

```javascript
terminal.start({
  alternateScreen: false,
});
```

建议提供显式 scrollback 策略：

```javascript
session.clear({
  screen: true,
  scrollback: false,
});

session.clear({
  screen: true,
  scrollback: true,
});
```

说明：

- `screen: true` 对应清可见屏幕。
- `scrollback: true` 可使用 `CSI 3J` 等能力清滚动历史，但必须由调用方显式选择。
- 默认不应清 scrollback，避免意外删除用户终端历史。
- 如果终端不支持清 scrollback，标准库应返回能力信息或降级。

建议提供能力检测：

```javascript
let caps = terminal.capabilities();

// 示例字段
caps.clearScrollback;
caps.alternateScreen;
caps.resizeEvents;
caps.virtualTerminal;
```

## P0：resize 事件需要防抖和最终尺寸

当前 resize loop 轮询尺寸变化后立即触发 `onResize`。窗口拖拽过程中会产生多个中间尺寸，TUI 会多次清屏重绘，更容易和终端 reflow 交错。

建议支持：

```javascript
terminal.start({
  onResize: function(size) {},
  resizeDebounceMs: 50,
});
```

或提供事件类型：

```javascript
{
  type: "resize",
  cols: 120,
  rows: 32,
  stable: true
}
```

最小目标：

- 拖拽窗口时不要对每个瞬时尺寸都立即 full redraw。
- 最终尺寸稳定后至少触发一次准确 resize。
- resize 回调期间如再次 resize，应合并到下一轮，不打断当前 redraw。

## P1：标准库提供屏幕缓冲抽象

建议 `@std/terminal` 提供一个轻量屏幕对象，封装常见 TUI 输出：

```javascript
let screen = terminal.createScreen({
  session: session,
  alternateScreen: true,
});

screen.render(lines, {
  rows: size.rows,
  cols: size.cols,
  diff: true,
  clip: true,
});

screen.reset();
screen.clear();
```

屏幕对象负责：

- 行裁剪。
- 列裁剪。
- ANSI 感知宽度。
- diff render。
- resize 后 full render。
- 不写出 viewport。
- 不在普通屏幕中意外触发滚动。

这样应用层不需要每个项目都自己维护 `previous rows`、`clearLine()`、`moveTo()` 和 resize full render。

## P1：文本宽度和 ANSI 处理继续标准化

TUI 稳定渲染还依赖 ANSI 和 Unicode 宽度处理。建议标准库继续提供：

```javascript
text.width(value);
text.stripAnsi(value);
text.truncateWidth(value, width);
text.padRightWidth(value, width);
text.wrapWidth(value, width);
```

要求：

- CJK 宽字符正确。
- emoji 和组合字符尽量正确。
- ANSI escape 不计入显示宽度。
- 截断时不要破坏 ANSI reset。

这部分已经有 `@std/text` 能力雏形，建议作为 TUI screen 渲染的底层依赖。

## P1：错误恢复应覆盖异步终端回调

`onInput` / `onResize` 是最容易发生 TUI 错误的地方。建议 callback runtime error 默认触发：

1. `session.restore()`。
2. `session.stop()`。
3. 调用 `onError(error, session)`。
4. 无 `onError` 时打印明确错误。

示例：

```javascript
terminal.start({
  raw: true,
  alternateScreen: true,
  hideCursor: true,
  onResize: function(size) {
    throw new Error("resize render failed");
  },
  onError: function(error, session) {
    session.restore();
  },
});
```

不应出现隐藏光标、鼠标模式残留、备用屏不退出等状态。

## 建议 API 汇总

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

session.clear({ screen: true, scrollback: false });
session.redraw(function(screen) {
  screen.moveTo(1, 1);
  screen.write(lines);
});
session.renderFrame(lines, {
  rows: 30,
  cols: 120,
  diff: true,
  clip: true,
  full: false,
});
session.restore();
session.stop();

let caps = terminal.capabilities();
```

## 建议测试场景

### 备用屏 resize

1. 启动 TUI，进入备用屏。
2. 连续拖拽窗口大小。
3. 每次最终尺寸稳定后界面完整重绘。
4. 不出现重复界面、残留边框、隐藏光标。
5. 退出后返回主屏幕。

### 普通屏幕 resize 不清 scrollback

1. 启动 `alternateScreen: false` 的 TUI。
2. resize 后 full render。
3. 当前可见区域不出现重复 UI。
4. scrollback 可能保留历史，但不应把旧 UI 混进当前 viewport。

### 普通屏幕 resize 清 scrollback

1. 启动 `alternateScreen: false`。
2. 调用 `session.clear({ screen: true, scrollback: true })`。
3. resize 后当前 viewport 干净。
4. 如果终端不支持，返回明确降级信息。

### callback 异常恢复

1. `onResize` 中主动抛错。
2. 标准库自动恢复 raw mode、cursor、mouse、alternate screen。
3. 进程退出后终端可正常输入。

### 行数越界保护

1. 向 `renderFrame()` 传入超过 `rows` 的内容。
2. 标准库自动裁剪。
3. 不触发终端滚动。

## 对 `gs-agent` 的落地建议

短期应用侧：

- 将 `src/tui/app.gs` 中 `alternateScreen: false` 改为 `true`，优先保证 resize 稳定。
- 退出备用屏后打印最终摘要，如事件数、answer 文件路径、log 文件路径。

中期语言侧：

- `@std/terminal` 增加 `clear({ screen, scrollback })`。
- `terminal.start()` 支持 `hideCursor`、`alternateScreen`、`restoreOnExit`、`restoreOnError` 的完整托管。
- 增加 resize debounce。
- 增加 `session.renderFrame()` 或 `terminal.createScreen()`。

长期框架侧：

- `src/tui/screen.gs` 从应用层迁移为标准库或官方 TUI helper。
- 应用只负责状态和组件渲染，不直接拼接底层 ANSI。

## 优先级

1. 默认推荐备用屏，全屏 TUI 先稳定。
2. 标准库托管终端生命周期和异常恢复。
3. 提供 `clear({ screen, scrollback })`。
4. 提供 resize debounce 和原子 redraw。
5. 提供 screen buffer / renderFrame 抽象。
6. 完善文本宽度、ANSI、键鼠事件标准化。

## 一句话总结

普通屏幕模式下的 resize 稳定性不是单个 TUI 应用能完全保证的。应用层可以减少越界写入和重绘错误，但要可靠处理 Windows Terminal 的主屏幕缓冲区、scrollback、resize reflow 和异常恢复，应该由 GoScript 的 `@std/terminal` 提供统一会话、清屏、scrollback、resize 和屏幕渲染能力。
