# GoScript 教学实例

本目录提供从入门到进阶的 GoScript 教学实例，按学习顺序编号，每步聚焦一个核心概念。

## 学习路线

| 编号 | 文件 | 学习内容 |
|------|------|----------|
| 01 | `01-basics.gs` | 变量声明、控制台输出、注释 |
| 02 | `02-operators.gs` | 算术、比较、逻辑运算符 |
| 03 | `03-control-flow.gs` | if/else、while、for 循环 |
| 04 | `04-functions.gs` | 函数声明、参数、返回值 |
| 05 | `05-closures.gs` | 闭包、高阶函数 |
| 06 | `06-arrays-objects.gs` | 数组/对象字面量、内置方法 |
| 07 | `07-types.gs` | 可选类型注解 |
| 08 | `08-match.gs` | 模式匹配 (match) |
| 09 | `09-errors.gs` | 错误处理 try/catch/finally |
| 10 | `10-classes.gs` | 类与继承 |
| 11 | `11-async.gs` | 异步编程 async/await |
| 12 | `12-comprehensive.gs` | 综合实战：图书管理系统 |

## 运行方式

```bash
# 安装 GoScript
go build -o gs ./cmd/gs

# 运行单个实例
./gs examples/01-basics.gs

# 或使用 go run
go run ./cmd/gs examples/01-basics.gs
```

每个 `.gs` 文件都是独立可执行的脚本，建议按编号顺序学习。

## 当前自动验证状态

当前 CLI 回归测试只收录已经稳定、无网络依赖、不会长时间等待异步任务的示例：

| 状态 | 文件 |
|------|------|
| 稳定回归 | `examples/01-basics.gs` |
| 稳定回归 | `docs/examples/hello.gs` |
| 稳定回归 | `docs/examples/fib.gs` |
| 稳定回归 | `docs/examples/counter.gs` |
| 稳定回归 | `docs/examples/modules.gs` |

其余示例目前保留为教学/目标行为示例，暂不纳入自动测试。主要原因是部分示例使用了尚未完整支持的语法、类型检查能力，或包含异步、网络、长时间运行行为。后续每修复一类能力，再把对应示例移动到稳定回归清单。

## 资源保护

CLI 默认带有脚本运行超时保护，避免异步、网络或无限循环示例长期占用内存：

```bash
go run ./cmd/gs --timeout 10s examples/01-basics.gs
```

需要调试长时间运行脚本时可以显式调大超时，或用 `--timeout 0` 关闭保护。自动测试中只允许加入短时间、无网络、可确定退出的脚本。
