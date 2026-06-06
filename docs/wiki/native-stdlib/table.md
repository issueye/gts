# @std/table

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/table` | 原生模块路径 |

## 加载

```javascript
let table = require("@std/table");
```

## 接口

| 接口 | 说明 |
|------|------|
| `layout(rows, options?)` | 按终端宽度布局表格，返回 `lines` / `widths` |
| `options.width` | 表格总宽度 |
| `options.minColWidth` | 最小列宽 |
| `options.wrap` | 是否按列宽换行 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
