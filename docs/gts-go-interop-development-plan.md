# GTS 与 Go 交互开发计划

> 本计划承接 [`gts-go-interop-protocol.md`](gts-go-interop-protocol.md)，目标是先提供可用的 Go SDK，再逐步实现独立进程 GTP。

## 1. 当前目标

第一阶段先完成进程内互操作：

1. GTS 能解析 `@go/*`、`@host/*`、`@plugin/*` 原生模块。
2. Go 程序能通过公开 SDK 注册模块和方法。
3. Go 程序能通过 SDK 运行源码、文件和项目。
4. SDK 提供基础值转换、参数读取、错误构造。
5. 有测试证明 GTS 可以调用 Go 注册的方法。

## 2. 已完成

- 新增 `sdk` 公开包。
- 新增 `sdk.Runtime`，支持 `RunSource`、`RunFile`、`RunProject`。
- 新增 `Runtime.CallExport`，支持 Go 直接调用 GTS 模块导出函数。
- 新增 `sdk.RegisterModule`，支持注册 `@go/*`、`@host/*`、`@plugin/*` 模块。
- 新增 `Runtime.RegisterModule`，支持运行时局部模块注册，避免污染全局原生模块注册表。
- 新增 `sdk.Value`、`ToValue`、`FromValue`、`AsString`、`AsNumber`、`AsBool`、`AsObject`、`AsArray` 等辅助 API。
- 新增 `sdk.AnyMethod` / `Module.MethodsAny`，支持 Go 方法返回普通 Go 值并自动转换为 GTS 值。
- 新增 `sdk.Args` / `NewArgs`，支持带名称的参数读取和类型错误。
- 扩展 native module resolver，使 `@go/*`、`@host/*`、`@plugin/*` 与 `@std/*` 一样走 Go 原生模块注册表。

## 3. Go SDK 使用示例

```go
package main

import (
    "fmt"
    "time"

    "github.com/issueye/goscript/sdk"
)

func main() {
    _ = sdk.RegisterModule(sdk.Module{
        Name: "@go/math",
        Methods: map[string]sdk.Method{
            "add": func(ctx sdk.CallContext, args []sdk.Value) (sdk.Value, error) {
                a, _ := sdk.AsNumber(args[0])
                b, _ := sdk.AsNumber(args[1])
                return sdk.Number(a + b), nil
            },
        },
    })

    rt := sdk.NewRuntime(sdk.Options{Timeout: 5 * time.Second})
    defer rt.Close()

    result, err := rt.RunSource(`
        const math = require("@go/math");
        math.add(2, 3);
    `, "main.gs")
    if err != nil {
        panic(err)
    }
    fmt.Println(sdk.FromValue(result))
}
```

## 4. 下一阶段任务

### 4.1 SDK 稳定化

- 增加 SDK 文档和更多示例。

### 4.2 错误模型

- 在 `object.NewError` 中识别 `PermissionError`、`TimeoutError`、`HostError`。
- SDK 中提供 `TypeError`、`RangeError`、`HostError`、`PermissionError` 便捷构造函数。
- 错误对象保留 Go 错误链的可读信息，但不泄露敏感数据。

### 4.3 权限能力

- 新增 host capability 配置。
- 每个模块可声明需要的权限。
- SDK 在模块导入和方法调用时执行权限检查。

### 4.4 GTP 进程间协议

- 已新增 `internal/gtp` GTP 帧和值编码基础包。
- 已新增 GTP JSON Lines 编解码与性能基线测试。
- 已新增 GTP 插件宿主管理器，`gs run` 会读取 `config.toml` 的 `[plugins]` 配置并自动唤醒插件。
- 已支持 `require("@plugin/...")` 代理调用已唤醒插件的方法。
- 实现 JSON Lines transport 的真实进程/pipe 版本。
- 实现 hello/ready/call/result/cancel 基础帧。
- 实现外部服务代理模块。
- 实现 resource 代理和 VM 结束清理。
- 增加 stream resource。

## 5. 验收标准

短期验收：

1. `go test ./sdk` 通过。
2. 外部 Go 程序可以 import `github.com/issueye/goscript/sdk`。
3. GTS 脚本可以 `require("@go/demo")` 调用 Go 方法。
4. `import demo from "@go/demo"` 对原生模块可用。
5. Go 方法错误能变成 GTS 运行时错误。

长期验收：

1. 同一脚本 API 可在进程内 ABI 和进程间 GTP 之间切换。
2. 权限默认关闭，宿主显式授权。
3. 资源句柄有明确生命周期。
4. SDK API 在 v1 前保持最小破坏。
