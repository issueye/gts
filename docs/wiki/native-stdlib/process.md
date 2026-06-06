# @std/process

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/process` | 原生模块路径 |

## 加载

```javascript
let process = require("@std/process");
```

## 接口

| 接口 | 说明 |
|------|------|
| `argv` | 当前脚本参数数组 |
| `argv0` | 进程启动名 |
| `pid` | 当前进程 ID |
| `env` | 环境变量对象 |
| `cwd()` | 返回当前工作目录 |
| `chdir(path)` | 切换工作目录 |
| `execPath()` | 返回可执行文件路径 |
| `getenv(name, fallback?)` | 读取环境变量 |
| `envObject()` | 返回环境变量对象 |
| `uptime()` | 返回进程运行秒数 |
| `hrtime(previous?)` | 返回高精度时间数组 |
| `setenv(name, value)` | 设置环境变量 |
| `unsetenv(name)` | 删除环境变量 |
| `exit(code?)` | 退出进程 |
| `version` | GoScript 版本 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
