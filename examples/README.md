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
