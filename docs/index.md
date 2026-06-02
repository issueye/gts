# 文档索引

> 本目录包含 GoScript v0.1 的设计与规范文档。  
> 阅读顺序建议：**`design.md` → `language-spec.md` → `async-model.md` → `builtins.md` → `grammar.ebnf`**。

## 文档清单

| 文档 | 内容 | 适用读者 |
|------|------|---------|
| [`README.md`](../README.md) | 项目总览、快速开始 | 所有人 |
| [`design.md`](design.md) | 架构总览、模块划分、关键算法、权衡 | 贡献者、架构师 |
| [`language-spec.md`](language-spec.md) | 语言规范：词法、类型、表达式、语句、对象 | 脚本作者 |
| [`grammar.ebnf`](grammar.ebnf) | 形式化 EBNF 语法 | 工具链作者、Parser 维护者 |
| [`bad-parts-fixed.md`](bad-parts-fixed.md) | **被修复的 JS 缺点一览** | 所有人（强烈推荐阅读） |
| [`builtins.md`](builtins.md) | 内置对象与标准库 API 参考 | 脚本作者 |
| [`async-model.md`](async-model.md) | Promise / async-await / 事件循环 | 高级用户、嵌入式使用 |
| [`roadmap.md`](roadmap.md) | 实施路线图与里程碑 | 贡献者、PM |
| [`examples/`](examples/) | 示例脚本合集 | 脚本作者 |

## 文档状态

- ✅ 设计与规范：**已完成**
- ⏳ 实施：尚未开始，按 [`roadmap.md`](roadmap.md) 推进
- ⏳ 用户教程：实施时同步撰写

## 反馈与贡献

发现文档错误、含糊或不一致？请在 issue 中提出，标注 `docs` 标签。
