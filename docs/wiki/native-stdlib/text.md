# @std/text

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/text` | 原生模块路径 |

## 加载

```javascript
let text = require("@std/text");
```

## 接口

| 接口 | 说明 |
|------|------|
| `chars(value)` | 返回剥离 ANSI 后的可见字符数组 |
| `runes(value)` | `chars` 的别名 |
| `width(value)` | 计算剥离 ANSI 后的终端显示宽度 |
| `truncateWidth(value, width)` | 按终端显示宽度截断文本 |
| `padRightWidth(value, width)` | 按终端显示宽度右侧补空格 |
| `wrapWidth(value, width)` | 按终端显示宽度换行，返回行数组 |
| `stripAnsi(value)` | 移除 ANSI escape 序列 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
