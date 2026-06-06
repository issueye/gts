# @std/template

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/template` | 原生模块路径 |

## 加载

```javascript
let template = require("@std/template");
```

## 接口

| 接口 | 说明 |
|------|------|
| `render(source, data?, options?)` | 渲染文本模板 |
| `renderHTML(source, data?, options?)` | 渲染 HTML 模板并转义 |
| `renderFileSync(path, data?, options?)` | 读取文件并渲染模板 |
| `escapeHTML(value)` | HTML 转义 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
