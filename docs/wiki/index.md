# GoScript Wiki

> 面向脚本作者、宿主集成者和维护者的知识库入口。这里放更偏“可查、可复制、可扩展”的接口资料，和 `docs/` 下的设计/规范文档互补。

## 快速入口

| 页面 | 内容 | 适用场景 |
|------|------|---------|
| [`native-stdlib.md`](native-stdlib.md) | 原生标准库接口知识库，覆盖 `@std/*` 模块路径、加载方式、接口清单和常见组合 | 编写脚本、查 API、补示例、做回归 |

## 阅读建议

1. 写脚本时，先看 [`native-stdlib.md`](native-stdlib.md) 的“模块索引”和对应模块接口。
2. 需要确认语言内置对象、全局函数、数组/字符串/对象方法时，继续参考 [`../builtins.md`](../builtins.md)。
3. 需要理解模块解析、原生模块注册和 `.gspkg` 关系时，参考 [`../project-analysis.md`](../project-analysis.md) 和 [`../package-module-design.md`](../package-module-design.md)。

## 维护约定

- 原生库接口以 `internal/stdlib/api_docs.go` 和各模块实现为事实来源。
- 新增或调整 `@std/*` 模块时，同步更新 [`native-stdlib.md`](native-stdlib.md) 的模块索引和接口清单。
- 示例优先保持短小、无外部网络依赖、可确定退出；复杂场景放到 `examples/` 或 `scripts/agent/`。
