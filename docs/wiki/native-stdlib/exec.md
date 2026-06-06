# @std/exec

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/exec` | 原生模块路径 |

## 加载

```javascript
let exec = require("@std/exec");
```

## 接口

| 接口 | 说明 |
|------|------|
| `run(command, argsOrOptions?)` | 执行命令并返回 `exitCode` / `stdout` / `stderr` |
| `output(command, args?)` | 执行命令并返回 stdout 字符串 |
| `combinedOutput(command, args?)` | 执行命令并返回 stdout + stderr |
| `start(command, args?) -> process` | 启动进程并返回控制对象 |
| `spawn(command, args?, options?) -> process` | 启动进程并暴露 stdin / stdout / stderr |
| `command(command, args?) -> cmd` | 创建可配置命令对象 |
| `process.write(data)` | 写入子进程 stdin |
| `process.writeln(data?)` | 写入一行到 stdin |
| `process.closeStdin()` | 关闭 stdin |
| `process.wait()` | 等待进程结束 |
| `process.kill()` | 终止进程 |
| `cmd.setDir(path)` | 设置工作目录 |
| `cmd.setEnv(env)` | 设置环境变量对象 |
| `cmd.run()/output()/combinedOutput()/start()` | 执行已配置命令 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
