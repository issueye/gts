# @std/log

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/log` | 原生模块路径 |

## 加载

```javascript
let log = require("@std/log");
```

## 接口

| 接口 | 说明 |
|------|------|
| `createFileLogger(path, options?) -> logger` | 创建文件日志器 |
| `logger.debug(...values)` | 写入 debug 日志 |
| `logger.info(...values)` | 写入 info 日志 |
| `logger.warn(...values)` | 写入 warn 日志 |
| `logger.error(...values)` | 写入 error 日志 |
| `logger.log(...values)` | 写入 info 日志 |
| `logger.close()` | 关闭日志文件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
