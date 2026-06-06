# @std/mime

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/mime` | 原生模块路径 |

## 加载

```javascript
let mime = require("@std/mime");
```

## 接口

| 接口 | 说明 |
|------|------|
| `typeByExtension(extension)` | 根据扩展名返回 MIME 类型 |
| `extensionByType(type)` | 根据 MIME 类型返回扩展名 |
| `parseMediaType(value)` | 解析媒体类型和参数 |
| `formatMediaType(type, params?)` | 格式化媒体类型 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
