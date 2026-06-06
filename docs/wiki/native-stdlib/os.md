# @std/os

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/os` | 原生模块路径 |

## 加载

```javascript
let os = require("@std/os");
```

## 接口

| 接口 | 说明 |
|------|------|
| `platform` | 当前操作系统平台 |
| `arch` | 当前 CPU 架构 |
| `eol` | 当前平台换行符 |
| `type()` | 返回操作系统类型 |
| `release()` | 返回系统版本 |
| `homedir()` | 返回用户主目录 |
| `tmpdir()` | 返回临时目录 |
| `hostname()` | 返回主机名 |
| `cpus()` | 返回 CPU 信息数组 |
| `userInfo()` | 返回当前用户信息 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
