# Go SDK Host 示例

这个示例演示 Go 程序如何通过 `sdk` 包注册 `@go/demo` 模块，然后运行一段 GTS 脚本调用 Go 方法；同时也演示 Go 直接调用 GTS 文件导出的函数。

运行：

```bash
go run ./examples/18-go-sdk-host
```

期望输出会包含：

```text
GTS result: map[string]interface {}{... "total":42 ...}
GTS export result: map[string]interface {}{... "count":3 ...}
```
