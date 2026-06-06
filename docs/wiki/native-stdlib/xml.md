# @std/xml

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/xml` | 原生模块路径 |

## 加载

```javascript
let xml = require("@std/xml");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(text)` | 解析 XML 文本 |
| `stringify(node)` | 序列化 XML 节点对象 |
| `readFileSync(path)` | 读取并解析 XML 文件 |
| `writeFileSync(path, node)` | 序列化并写入 XML 文件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
