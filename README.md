# GoScript

> 一门使用 Go 实现的、语法风格参考 JavaScript 的动态脚本语言。  
> 文件后缀：`.gs`　|　解释器命令：`gs`

GoScript 的目标是为 Go 生态提供一个**易嵌入、可扩展、熟悉 JS 风格**的脚本层，
同时通过**可选类型注解**给动态脚本加上文档与早期检查能力。
GoScript 刻意**移除了 JavaScript 中公认的"Bad Parts"**（`==` 松散比较、`switch` 的 fall-through、隐式全局、`typeof null === "object"` 等）。

---

## ✨ 特性一览

| 维度 | 决策 |
|------|------|
| 执行模型 | 纯 AST 树遍历解释器（Tree-walking Interpreter） |
| 类型系统 | 动态类型 + 可选类型注解（运行时校验） |
| 语法风格 | 贴近 JavaScript（ES2020 子集），移除了 Bad Parts |
| 相等性 | 只允许 `===` / `!==`（`==` / `!=` 语法错误） |
| 模式匹配 | `match` 替代 `switch`（无 fall-through，表达式可用） |
| 异步模型 | `Promise` + `async/await`，基于微任务事件循环 |
| 模块系统 | 文件级 `import` / `export` |
| 面向对象 | 基于原型的 `class` / `extends` / `super` |
| 错误处理 | 异常机制（`throw` / `try-catch-finally`） |
| 宿主语言 | Go 1.22+ |
| 嵌入能力 | 暴露 `Evaluator` 类型，可在 Go 程序中驱动脚本 |

---

## 🚀 快速开始

```bash
# 1. 克隆并构建
git clone <your-repo>/goscript.git
cd goscript
go build -o gs ./cmd/gs

# 2. REPL 模式
$ ./gs
gs> let x = 1 + 2;
gs> x
3
gs> .exit

# 3. 解释执行脚本
$ ./gs examples/hello.gs
Hello, GoScript!
```

一个最小脚本 `hello.gs`：

```javascript
// hello.gs
let name: string = "GoScript";
console.log("Hello, " + name + "!");
```

---

## 📚 文档导航

| 文档 | 作用 |
|------|------|
| [`docs/design.md`](docs/design.md) | 架构总览、模块划分、关键算法与权衡 |
| [`docs/language-spec.md`](docs/language-spec.md) | 语言规范：词法、类型、表达式、语句、对象模型 |
| [`docs/grammar.ebnf`](docs/grammar.ebnf) | 形式化 EBNF 语法 |
| [`docs/bad-parts-fixed.md`](docs/bad-parts-fixed.md) | **GoScript 修复的 JS 缺点一览**（==、+、switch 等） |
| [`docs/builtins.md`](docs/builtins.md) | 内置对象、内置函数、标准库 |
| [`docs/async-model.md`](docs/async-model.md) | `Promise` / `async/await` / 事件循环 |
| [`docs/roadmap.md`](docs/roadmap.md) | 实施里程碑 |
| [`docs/examples/`](docs/examples/) | 示例脚本合集 |

---

## 🧭 一段示例

```javascript
// 异步 + 类型注解 + 类 + 模块导入 + match 模式匹配
import { fetch } from "net";

class User {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  greet(): string {
    return `Hi, I'm ${this.name}`;
  }
}

async function main() {
  let u: User = new User("Alice");
  console.log(u.greet());

  let resp = await fetch("https://api.example.com/api");
  let code: number = resp.status;

  // match 表达式（替代 switch，无 fall-through）
  let label: string = match code {
    200 => "OK",
    301 => "Moved",
    404 => "Not Found",
    500..599 => "Server Error",
    _ => "Unknown",
  };
  console.log(`HTTP ${code}: ${label}`);

  // 严格相等 (===)，没有 == 的隐式转换
  if (code === 200) console.log("success");
}

main();
```

---

## 🏗️ 仓库结构

```
gts/
├── cmd/gs/                # CLI 入口（gs 命令）
├── internal/
│   ├── lexer/             # 词法分析
│   ├── parser/            # 语法分析（Pratt 表达式 + 递归下降语句）
│   ├── ast/               # AST 节点定义
│   ├── object/            # 运行时值与对象系统
│   ├── resolver/          # 作用域与名字解析
│   ├── typechecker/       # 可选类型注解校验
│   ├── evaluator/         # 树遍历求值器
│   ├── async/             # 事件循环、Promise、continuation
│   ├── stdlib/            # 内置对象与函数
│   └── errors/            # 统一错误类型
├── docs/                  # 设计与规范文档
├── test/                  # 测试与脚本用例
└── go.mod
```

---

## 🛣️ 路线图

- [x] 设计阶段：语言规范、EBNF、架构、Bad Parts 修复
- [ ] 实现阶段：词法 / 语法 / 求值器 / match 模式匹配 / 异步运行时
- [ ] 标准库：console / Math / JSON / Object / Array / String
- [ ] 嵌入 API：作为 Go 库被其它程序调用
- [ ] 工具链：REPL、语法高亮、Language Server（远期）

---

## 📝 许可证

MIT License
