# GoScript

> 一门使用 Go 实现的、语法风格参考 JavaScript 的动态脚本语言。  
> 文件后缀：`.gs` | 当前解释器入口：`cmd/gs`

GoScript 的目标是为 Go 生态提供一个**易嵌入、可扩展、熟悉 JS 风格**的脚本层，同时通过可选类型注解给动态脚本加上文档与早期检查能力。

当前仓库已经具备词法、语法、AST、求值器、CLI、基础内置函数、部分标准库和示例回归测试。部分文档仍描述目标形态；最新开发计划见 [`docs/development-plan.md`](docs/development-plan.md)。

---

## 特性状态

| 维度 | 当前状态 |
|------|----------|
| 执行模型 | 纯 AST 树遍历解释器 |
| CLI | 已支持 `go run ./cmd/gs <file>`、`go run ./cmd/gs run` |
| 资源保护 | CLI 默认 `--timeout 10s`，可用 `--timeout 0` 关闭 |
| 语法风格 | JavaScript 风格子集，词法层拒绝 `==` / `!=`，未声明赋值会报错 |
| 基础运行时 | 数字、字符串、布尔、数组、对象、函数、闭包、类等已有测试覆盖 |
| 模式匹配 | `match` 已有解析和部分求值能力 |
| 异步模型 | `Promise`、`async/await`、timer 有部分实现，语义仍需加固 |
| 模块系统 | `require(path)` 已接入 CLI；基础 named/default/namespace `import/export` 已可运行，完整语义仍待补齐 |
| 类型系统 | 类型注解可解析；`--check-types` 暂未实现类型检查 |
| REPL | 基础版已实现：无参数启动、持久环境、`.help` / `.exit` / `.load` |
| 嵌入 API | 暂未提供稳定公开 facade |

---

## 快速开始

```bash
# 构建 CLI
go build -o gs ./cmd/gs

# 初始化一个新项目
./gs init hello-app
cd hello-app

# 运行根目录 project.toml 指定的入口
./gs run

# 运行单个脚本
./gs examples/01-basics.gs

# 打包成 .gspkg
./gs pack . dist/hello-app.gspkg

# 打包成可直接分发运行的二进制
./gs dist . dist/hello-app

# 不构建，直接运行
go run ./cmd/gs main.gs

# 进入交互式 REPL
go run ./cmd/gs
```

CLI 常用参数：

```bash
# 查看版本
go run ./cmd/gs --version

# 调整脚本最长运行时间，默认 10 秒
go run ./cmd/gs --timeout 2s main.gs

# 关闭超时保护，仅建议调试时使用
go run ./cmd/gs --timeout 0 main.gs

# 限制异步 worker 数量
go run ./cmd/gs --workers 1 examples/01-basics.gs
```

`--check-types` 当前会返回“类型检查器尚未实现”的明确错误，后续会接入 `internal/typechecker`。

---

## 最小脚本

```javascript
// hello.gs
let name = "GoScript";
console.log("Hello, " + name + "!");
```

运行：

```bash
go run ./cmd/gs hello.gs
```

项目入口也可以写成 `main()`。当执行 `gs run` 或直接执行名为 `main.gs` 的文件时，CLI 会在加载文件后自动调用顶层 `main()`：

```javascript
function main() {
  println("Hello, GoScript!");
}
```

---

## 文档导航

| 文档 | 作用 |
|------|------|
| [`docs/development-plan.md`](docs/development-plan.md) | 当前开发计划和真实缺口 |
| [`docs/project-analysis.md`](docs/project-analysis.md) | 项目分析：当前状态、运行链路、模块职责和接手建议 |
| [`docs/design.md`](docs/design.md) | 架构总览、模块划分、关键算法与权衡 |
| [`docs/language-spec.md`](docs/language-spec.md) | 语言规范：词法、类型、表达式、语句、对象模型 |
| [`docs/grammar.ebnf`](docs/grammar.ebnf) | 形式化 EBNF 语法 |
| [`docs/bad-parts-fixed.md`](docs/bad-parts-fixed.md) | **GoScript 修复的 JS 缺点一览**（==、+、switch 等） |
| [`docs/builtins.md`](docs/builtins.md) | 内置对象、内置函数、标准库 |
| [`docs/async-model.md`](docs/async-model.md) | `Promise` / `async/await` / 事件循环 |
| [`docs/roadmap.md`](docs/roadmap.md) | 早期路线图，当前执行以 development-plan 为准 |
| [`docs/examples/`](docs/examples/) | 示例脚本合集 |
| [`examples/README.md`](examples/README.md) | 教学示例和自动验证状态 |

---

## 当前可稳定验证的示例

```bash
go test ./cmd/gs
```

当前自动回归清单保持很小，只收录短时间、无网络、确定退出的脚本。详见 [`examples/README.md`](examples/README.md)。

---

## 仓库结构

```
gts/
├── cmd/gs/                # CLI 入口
├── internal/
│   ├── lexer/             # 词法分析
│   ├── parser/            # 语法分析（Pratt 表达式 + 递归下降语句）
│   ├── ast/               # AST 节点定义
│   ├── object/            # 运行时值与对象系统
│   ├── evaluator/         # 树遍历求值器
│   ├── async/             # 异步 worker 池
│   ├── module/            # 模块缓存与原生模块注册
│   ├── stdlib/            # 原生标准库模块
│   ├── bundle/            # 简易打包器实验
│   └── proj/              # project.toml 读取
├── docs/                  # 设计与规范文档
├── examples/              # 教学示例
├── vscode-extension/      # VS Code 语法高亮扩展
├── main.gs                # 根目录示例入口
├── project.toml           # 项目配置示例
└── go.mod
```

---

## 开发计划

当前优先级：

1. 文档与实现对齐。
2. 完善模块运行时，补齐 `import/export` 的完整语义。
3. 补强运行时语义：错误对象、console 方法、C 风格 `for` 边角。
4. 实现 `internal/typechecker` 并接入 `--check-types`。
5. 逐步扩大稳定示例回归清单。

完整计划见 [`docs/development-plan.md`](docs/development-plan.md)。

---

## 测试

```bash
go test ./...
```

---

## 许可证

MIT License
