# @std/pty

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/pty` | 原生模块路径 |

## 加载

```javascript
let pty = require("@std/pty");
```

## 接口

| 接口 | 说明 |
|------|------|
| `spawn(command, args?, options?) -> pty` | 启动伪终端进程 |
| `open(command, args?, options?) -> pty` | `spawn` 的别名 |
| `pty.write(data)` | 写入伪终端 |
| `pty.writeln(data?)` | 写入一行 |
| `pty.read(si

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
