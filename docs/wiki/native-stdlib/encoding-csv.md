# @std/encoding/csv

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/encoding/csv` | 原生模块路径 |

## 加载

```javascript
let encodingCsv = require("@std/encoding/csv");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(text, options?)` | 解析 CSV 文本 |
| `stringify(rows, options?)` | 将数组或对象行序列化为 CSV |
| `readFileSync(path, options?)` | 读取并解析 CSV 文件 |
| `writeFileSync(path, rows, options?)` | 序列化并写入 CSV 文件 |
| `options.header` | 是否使用首行表头 |
| `options.comma/comment/fieldsPerRecord/trimLeadingSpace` | CSV 解析写入选项 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
