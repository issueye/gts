# @std/highlight

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/highlight` | 原生模块路径 |

## 加载

```javascript
let highlight = require("@std/highlight");
```

## 接口

| 接口 | 说明 |
|------|------|
| `terminal(code, options?)` | 轻量代码高亮，返回 `lines` / `text` / `lang` |
| `options.lang` | 语言：`json` / `diff` / `shell` / `gs` / `js` / `toml` |
| `options.width` | 按终端宽度换行 |
| `options.color` | 是否输出 ANSI 颜色 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
