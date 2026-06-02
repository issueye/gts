# 实施路线图

> 状态：v0.1 设计阶段 ✅  
> 当前阶段目标：在 `cmd/gs/main.go` 中跑通一个最小可用的解释器（同步子集 + 闭包 + 类型注解可选检查）。

---

## 里程碑

| ID | 名称 | 范围 | 验收 |
|----|------|------|------|
| M0 | 骨架 | `go.mod`、`cmd/gs/main.go`、目录结构 | `go build` 成功 |
| M1 | 词法 | 全部 token、数字/字符串/模板/Unicode 标识符、**不含 ==/!=**、新增 `..`/`..=`/`match` | 词法测试 100% 通过 |
| M2 | 语法 | Pratt + 递归下降；表达式 + 语句 + **match 模式匹配**、**不含 switch/case/default** | 解析器测试 100% 通过 |
| M3 | 求值器 | 基本类型、运算符、控制流（含 match）、严格 ===、严格 + | `fib.gs` `match.gs` 通过 |
| M4 | 函数 / 闭包 | 一等公民、环境指针、箭头函数 | `counter.gs` 通过 |
| M5 | 数组 / 对象 | 字面量、内置方法 | `hello.gs` 通过 |
| M6 | 类 | 继承、`super`、`this` 绑定、字段初始化 | `class.gs` 通过 |
| M7 | 异常 | `try/catch/finally`、Error 子类、堆栈 | `errors.gs` 通过 |
| M8 | Resolver | 作用域分析、闭包标注、this 静态绑定 | Resolver 测试通过 |
| M9 | 异步 | Promise、async/await、续延、事件循环 | `async.gs` `fetch.gs` 通过 |
| M10 | 类型注解 | `--check-types`、结构匹配 | `types.gs` 通过 |
| M11 | 模块 | `import/export`、`require`、文件级缓存 | 多文件项目跑通 |
| M12 | 标准库 | console / Math / JSON / Object / Array / String / RegExp / Date / Map / Set | 内置 API 测试通过 |
| M13 | 嵌入 API | `evaluator.New()` 友好接口、文档示例 | Go 嵌入示例跑通 |
| M14 | REPL | `gs` 不带参数进入交互模式 | REPL 测试通过 |
| M15 | 工具 | 错误信息美化、源码映射（远期） | — |

---

## 当前阶段：M0

**目标**：建立可被 Go 构建的空壳 CLI。

**已完成**：
- 项目结构与目录划分
- 完整设计与规范文档
- `cmd/gs/main.go` 接受参数，识别 `--check-types` 等 flag
- `go build` 成功

**下一步**：进入 M1，开始 `internal/lexer` 实现。

---

## 后续里程碑细目

### M1 词法（约 1~2 周）

- 词法单元定义（`internal/lexer/token.go`）
- DFA 状态机（`internal/lexer/lexer.go`）
- 字符串 / 模板 / Unicode 标识符
- 错误收集与多错聚合
- 表驱动测试

### M2 语法（约 2~3 周）

- `internal/ast` 节点定义
- `internal/parser` Pratt + 递归下降
- EBNF 形式与实现一一对应
- 快照测试

### M3 求值器（约 2~3 周）

- `internal/object` 值与对象
- `internal/evaluator` 树遍历
- `internal/errors` 错误类型
- 控制流对象（`BREAK/CONTINUE/RETURN`）

### M4~M7（约 4~5 周）

- 函数、闭包、数组、对象、类、异常

### M8 Resolver（约 1~2 周）

- 静态作用域分析
- 闭包变量标注
- `this` 静态绑定

### M9 异步（约 3~4 周，复杂度最高）

- `internal/async` 运行时
- Promise 状态机
- 续延 closure 构建
- 事件循环
- `setTimeout` / `fetch` 桥接

### M10 类型注解（约 1~2 周）

- `internal/typechecker`
- 结构类型匹配
- 联合类型解析
- 错误信息

### M11 模块（约 1~2 周）

- 文件加载器
- 模块缓存
- `import` / `export` 求值

### M12~M15（约 4~5 周）

- 标准库补全
- 嵌入 API
- REPL
- 工具打磨

---

## 风险与应对

| 风险 | 应对 |
|------|------|
| 续延在控制流深处构建复杂 | 用统一的 `continuation builder`，单元测试覆盖各种嵌套 |
| Resolver 与 Evaluator 双向耦合 | 严格分层：Resolver 只标注，不执行 |
| 异步栈追踪 | 从一开始就为每个续延打"前一帧"引用 |
| 性能不达标 | v0.1 接受；若成为问题，后续加字节码层 |

---

## 协作建议

- 每个里程碑开一个 git 分支
- CI 跑 `go test ./...` 与示例脚本 `gs examples/*.gs`
- PR 描述引用对应里程碑
- 文档与代码同步更新
