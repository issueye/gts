# @std/markdown

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/markdown` | 原生模块路径 |

## 加载

```javascript
let markdown = require("@std/markdown");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(source)` | 解析 Markdown，返回 block / inline AST 和 diagnostics |
| `renderTerminal(source, options?)` | 渲染 Markdown 为终端行，支持 `options.width` |
| `createStream(options?)` | 创建流式 Markdown 预览器，支持 `append` / `snapshot` / `finali

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
