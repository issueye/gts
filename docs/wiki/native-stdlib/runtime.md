# @std/runtime

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/runtime` | 原生模块路径 |

## 加载

```javascript
let runtime = require("@std/runtime");
```

## 接口

| 接口 | 说明 |
|------|------|
| `runScript(path, options?)` | 在独立 VM 中执行外部 GoScript 文件并返回 exports |
| `callScript(path, exportName, args?, options?)` | 执行外部脚本并调用指定导出函数 |
| `runTool(path, input?, options?)` | `callScript(path, "run", [input], options)` 的便捷封装 |
| `options.cwd` | 动态脚本执行工作目录，默认继承当前进程 cwd |
| `options.argv` | 设置外部脚本 `process.argv` |
| `options.autoMain` | `runScript` 后自动调用 `main()` |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
