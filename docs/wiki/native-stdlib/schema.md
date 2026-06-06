# @std/schema

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/schema` | 原生模块路径 |

## 加载

```javascript
let schema = require("@std/schema");
```

## 接口

| 接口 | 说明 |
|------|------|
| `validate(schema, value)` | 校验值并返回结果对象 |
| `assert(schema, value)` | 校验失败时抛出错误 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
