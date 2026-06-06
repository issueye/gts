# @std/yaml

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/yaml` | 原生模块路径 |

## 加载

```javascript
let yaml = require("@std/yaml");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(text)` | 解析 YAML 文本 |
| `stringify(value)` | 序列化为 YAML 文本 |
| `readFileSync(path)` | 读取并解析 YAML 文件 |
| `writeFileSync(path, value)` | 序列化并写入 YAML 文件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
