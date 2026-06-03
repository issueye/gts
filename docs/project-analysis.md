# GoScript 项目分析

> 基于当前仓库状态整理，时间：2026-06-03。  
> 本文面向新接手维护者，用来快速理解项目定位、实现边界、运行链路和后续优先级。

---

## 1. 项目定位

GoScript 是一个使用 Go 实现的动态脚本语言解释器，语法风格参考 JavaScript，但有意识地收窄和修正部分 JS 行为，例如：

- 词法层拒绝 `==` / `!=`，要求使用 `===` / `!==`。
- `+` 不做数字和字符串的隐式拼接，类型不匹配会报错。
- 未声明变量赋值、`const` 重赋值、顶层 `break/continue` 等会返回明确错误。

当前实现重点不是把 JavaScript 完整复刻一遍，而是提供一个可运行、可嵌入、可扩展的 Go 生态脚本层，并逐步补齐模块、标准库、包分发和类型检查能力。

---

## 2. 当前真实状态

### 已具备

| 能力 | 状态 |
|------|------|
| CLI | `cmd/gs` 已支持直接运行脚本、`run` 项目入口、`init`、`pack`、`dist`、`bundle`、版本输出和超时保护 |
| 词法分析 | `internal/lexer` 已实现主流 JS 风格 token，并在词法层禁止部分不期望语法 |
| 语法分析 | `internal/parser` 使用 Pratt 表达式解析 + 递归下降语句解析，支持函数、类、模块、模式匹配、类型注解等语法 |
| AST | `internal/ast` 定义表达式、语句、类型和模式节点 |
| 求值器 | `internal/evaluator` 是纯 AST 树遍历解释器，覆盖变量、表达式、函数、闭包、类、异常、数组/对象方法、模块声明等 |
| 运行时对象 | `internal/object` 提供数字、字符串、布尔、数组、对象、函数、类实例、Promise、Map/Set、Date/RegExp、Error 等对象 |
| 异步执行 | `internal/async` 提供有界 worker pool，VM 负责挂接异步任务等待 |
| 模块系统 | `internal/module` 已实现 native/source/package/.gspkg 的解析、缓存和导出对象管理 |
| 原生标准库 | `internal/stdlib` 已注册多个 `@std/*` 模块，包括 fs/path/os/process/exec/db/http/socket/ws/crypto/buffer/timers 等 |
| 包文件 | `internal/packagefile` 支持 `.gspkg` 打包、读取、嵌套包和追加到可执行文件 |
| 项目配置 | `internal/proj` 读取 `project.toml` 中的 project/package/exports/imports/dependencies/bundle 配置 |
| 示例和回归 | `examples/`、`docs/examples/` 和 `cmd/gs` 测试已覆盖一批稳定示例 |
| 编辑器支持 | `vscode-extension/` 提供 VS Code 语法高亮扩展 |

### 仍未完成或需谨慎看待

| 能力 | 缺口 |
|------|------|
| 类型检查 | 类型注解能解析，但 `--check-types` 当前明确返回“尚未实现” |
| REPL | 尚无交互式 shell |
| 公开嵌入 API | 目前主要通过 internal 包组织，没有稳定 public facade |
| 模块语义 | 基础 `require`、`import/export` 可用，但循环依赖、完整 ES 模块语义、live binding 等仍需明确 |
| 错误模型 | 已有 Error 对象和位置信息，完整 Error 子类、稳定 stack 语义仍需加固 |
| 文档一致性 | 深层规范文档中仍有目标形态描述，开发时应以测试和当前实现为准 |
| 网络/长运行示例 | 一些标准库和 Agent 示例涉及外部环境，不能都作为短时稳定测试 |

---

## 3. 运行链路

### 直接运行脚本

典型命令：

```bash
go run ./cmd/gs examples/01-basics.gs
```

执行路径：

1. `cmd/gs/main.go` 解析 CLI 参数。
2. `runner.runFile` 判断是否需要自动调用 `main()`。
3. `runner.evalFile` 创建新的 `VirtualMachine`、`async.Pool`、`module.Cache` 和 `module.Resolver`。
4. 顶层环境通过 `module.SetupExports` 初始化 `exports` / `module.exports`。
5. `configureModuleLoaders` 注册全局内置函数和模块加载函数。
6. `evalSource` 依次执行 lexer、parser、evaluator。
7. 如果顶层结果或 `main()` 返回 Promise，runner 会等待 Promise 完成。
8. runner 等待 VM 异步任务和 worker pool 收尾，并受 `--timeout` 保护。

### 运行项目

典型命令：

```bash
go run ./cmd/gs run
```

执行路径与直接脚本类似，但入口来自当前目录的 `project.toml`：

```toml
[project]
entry = "main.gs"
```

项目运行时会临时切换工作目录到项目根目录，便于脚本内使用 `process.cwd()`、相对文件路径和本地模块解析。执行结束后会恢复原工作目录。

### 打包与分发

| 命令 | 作用 |
|------|------|
| `gs pack [dir] [out.gspkg]` | 将包含 `project.toml` 的目录打包为 `.gspkg` |
| `gs dist [dir] [out]` | 先生成 `.gspkg`，再追加到当前解释器可执行文件后，形成自包含可执行文件 |
| `gs bundle <entry.gs> [out.gs]` | 生成单文件 bundle，目前和运行时模块系统应分开看待 |

---

## 4. 模块与包解析

模块加载由 CLI runner、`internal/module`、`internal/packagefile` 和 evaluator 共同完成。

### 支持的模块来源

| 来源 | 示例 | 说明 |
|------|------|------|
| 原生模块 | `require("@std/fs")` | 通过 `module.RegisterNative` 注册，由 `internal/stdlib` 初始化 |
| 相对源码 | `require("./lib")` | 解析为 `.gs` 文件并缓存模块环境 |
| Agent 别名 | `@agent/core/agent` | 解析到项目内 `scripts/agent/` |
| package 依赖 | `require("tools")` | 通过 `project.toml` 的 dependencies / exports 找到入口 |
| `.gspkg` | `tools.gspkg` 或嵌套包 | 通过 zip 包读取源码，支持嵌套包引用 |

### 缓存模型

`module.Cache` 以 resolved ID 或绝对路径为 key 保存模块环境。首次加载模块时创建环境、初始化 `exports`，执行完成后返回 `module.GetExports(env)`。后续加载相同模块会直接返回缓存导出，避免重复执行。

### 注意点

- native 模块每次通过 factory 创建对象，不等同于源码模块环境缓存。
- `.gspkg` 模块路径使用 `packagePath!archive/path.gs` 形式表达包内位置。
- `require` 和 `import/export` 已连接到同一套 loader，但完整模块语义仍是后续重点。

---

## 5. 目录职责

```text
gts/
├── cmd/gs/                  CLI 入口、运行器、命令测试
├── internal/lexer/          词法分析
├── internal/parser/         语法分析
├── internal/ast/            AST 节点
├── internal/object/         运行时对象、环境、VM、Promise
├── internal/evaluator/      AST 求值器和内置对象方法
├── internal/async/          有界异步 worker pool
├── internal/module/         模块解析、缓存、native 注册
├── internal/stdlib/         原生标准库模块
├── internal/proj/           project.toml 读取
├── internal/packagefile/    .gspkg 打包/读取/嵌入可执行文件
├── internal/bundle/         简易 bundle 实验
├── docs/                    规范、设计、计划和示例文档
├── examples/                教学和集成示例
├── scripts/agent/           GoScript Agent 相关脚本和 smoke 示例
└── vscode-extension/        VS Code 语法扩展
```

---

## 6. 测试与验证入口

常用验证：

```bash
go test ./...
go test ./cmd/gs
go run ./cmd/gs examples/01-basics.gs
go run ./cmd/gs examples/16-native-stdlib.gs
```

测试分布：

| 目录 | 关注点 |
|------|--------|
| `internal/lexer/*_test.go` | token、词法边界、非法语法 |
| `internal/parser/*_test.go` | 语法结构和 AST |
| `internal/evaluator/*_test.go` | 运行时语义、内置方法 |
| `internal/module/*_test.go` | 模块解析和缓存 |
| `internal/packagefile/*_test.go` | `.gspkg` 打包读取 |
| `cmd/gs/main_test.go` | CLI、项目运行、包分发、稳定示例 |

当前仓库根目录存在 `gs.exe`，可用于本地快速运行；跨平台验证仍应优先使用 `go test` 和 `go run`。

---

## 7. 主要风险

1. **文档和实现存在时间差。** `docs/language-spec.md`、`docs/design.md` 等更偏目标描述，改动前要结合测试和源码确认。
2. **模块系统已经复杂化。** 相对源码、native、package、`.gspkg`、嵌套包和 alias 共用解析逻辑，改 resolver 时要补集成测试。
3. **异步超时不是取消机制。** runner 的 timeout 能让 CLI 返回错误，但不能保证所有底层阻塞操作都能被主动取消。
4. **标准库涉及外部资源。** db、http、socket、pty、terminal、web 等模块需要区分单元测试、smoke 测试和手动测试。
5. **内部包边界尚未沉淀。** 如果后续要给 Go 宿主嵌入，需要先设计公开 API，而不是直接暴露 internal 实现。

---

## 8. 建议优先级

### P0：保持现状可验证

- 运行并保持 `go test ./...` 通过。
- 把新增能力加入短时、确定性的 CLI 回归用例。
- 明确哪些示例是稳定、pending、manual。

### P1：模块语义收敛

- 为 `require`、ES-style `import/export`、package exports 和 `.gspkg` 增加更多端到端用例。
- 明确循环依赖策略。
- 明确默认导出、命名导出、namespace import 和 `module.exports` 的兼容边界。

### P2：运行时语义加固

- 补 Error 子类和 stack 行为。
- 加强 C-style `for`、异常、类继承、Promise 等边界测试。
- 对照 `bad-parts-fixed.md` 确保“修复 JS 缺点”的承诺都有测试。

### P3：类型检查和嵌入 API

- 新建 `internal/typechecker` 或先设计类型检查数据流。
- 为 CLI 的 `--check-types` 接入真实实现。
- 如果要服务 Go 宿主，新增稳定 public package，封装 runner/evaluator/module 细节。

---

## 9. 接手建议

新维护者建议按这个顺序阅读：

1. `README.md`
2. `docs/project-analysis.md`
3. `cmd/gs/main.go`
4. `internal/evaluator/evaluator.go`
5. `internal/module/resolver.go`
6. `internal/packagefile/packagefile.go`
7. `docs/development-plan.md`

修改实现时，优先从一个可执行脚本或测试用例出发，再改 parser/evaluator/module。这个项目的行为承诺应尽量落到可重复执行的示例和测试里。
