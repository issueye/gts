# 文档索引

> 本目录包含 GoScript v0.1 的设计、规范、实现状态和开发计划文档。  
> 新接手建议先读：**`project-analysis.md` → `development-plan.md` → `design.md` → `language-spec.md` → `builtins.md`**。

## 文档清单

| 文档 | 内容 | 适用读者 |
|------|------|---------|
| [`README.md`](../README.md) | 项目总览、快速开始 | 所有人 |
| [`project-analysis.md`](project-analysis.md) | 当前项目状态、运行链路、模块职责、风险与接手建议 | 新接手维护者、贡献者 |
| [`design.md`](design.md) | 架构总览、模块划分、关键算法、权衡 | 贡献者、架构师 |
| [`language-spec.md`](language-spec.md) | 语言规范：词法、类型、表达式、语句、对象 | 脚本作者 |
| [`ai-usage-guide.md`](ai-usage-guide.md) | 面向 AI 的精简使用指南：语法、原生库、注意事项、任务模板 | AI Agent、脚本作者 |
| [`grammar.ebnf`](grammar.ebnf) | 形式化 EBNF 语法 | 工具链作者、Parser 维护者 |
| [`bad-parts-fixed.md`](bad-parts-fixed.md) | **被修复的 JS 缺点一览** | 所有人（强烈推荐阅读） |
| [`builtins.md`](builtins.md) | 内置对象与标准库 API 参考 | 脚本作者 |
| [`async-model.md`](async-model.md) | Promise / async-await / 事件循环 | 高级用户、嵌入式使用 |
| [`package-module-design.md`](package-module-design.md) | 包模块打包、引用解析、依赖锁定与分发设计 | 运行时/工具链维护者 |
| [`roadmap.md`](roadmap.md) | 实施路线图与里程碑 | 贡献者、PM |
| [`examples/`](examples/) | 示例脚本合集 | 脚本作者 |

## 文档状态

- ✅ 基础解释器、CLI、模块加载、部分标准库和示例回归：**已实现**
- ⏳ 类型检查、REPL、公开嵌入 API 和完整模块语义：**仍待补齐**
- ⏳ 深层规范文档：部分仍描述目标形态，当前状态以 [`project-analysis.md`](project-analysis.md) 和 [`development-plan.md`](development-plan.md) 为准

## 反馈与贡献

发现文档错误、含糊或不一致？请在 issue 中提出，标注 `docs` 标签。
