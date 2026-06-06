# @std/toml

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/toml` | 原生模块路径 |

## 加载

```javascript
let toml = require("@std/toml");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(text)` | 解析 TOML 文本 |
| `stringify(value)` | 序列化为 TOML 文本 |
| `readFileSync(path)` | 读取并解析 TOML 文件 |
| `writeFileSync(path, value)` | 序列化并写入 TOML 文件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
