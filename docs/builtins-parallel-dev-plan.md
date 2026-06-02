# builtins 剩余能力并行开发计划

> 目标：继续按 `docs/builtins.md` 对齐剩余未实现或未完全实现的语言层能力。
> Provider/agent 逻辑仍由脚本层完成，L3 只负责通用运行时与标准库抽象。

## 并行分工

| Worker | 范围 | 主要文件 | 验收点 |
|--------|------|----------|--------|
| A | 异步与 Promise | `internal/evaluator/async.go`、`internal/evaluator/evaluator_test.go` | `Promise.race/allSettled`、`clearTimeout/clearInterval`、`queueMicrotask` 可用 |
| B | console | `internal/evaluator/console.go`、`internal/evaluator/builtins.go`、`internal/evaluator/evaluator_test.go` | 文档列出的 console 方法基础可调用 |
| C | Map / Set | `internal/object/object.go`、`internal/evaluator/map_set.go`、`internal/evaluator/evaluator.go`、`internal/evaluator/evaluator_test.go` | `new Map()`、`new Set()` 基础增删查和 `size` 可用 |
| D | Date / RegExp | `internal/object/object.go`、`internal/evaluator/date_regexp.go`、`internal/evaluator/string_methods.go`、`internal/evaluator/evaluator.go`、`internal/evaluator/evaluator_test.go` | Date 常用 getter/setter 与 RegExp `test/exec` 基础可用 |

## 集成原则

- 每个 worker 尽量使用独立新文件实现主体逻辑，只在注册和属性分发处做小幅接入。
- 不回退其他 worker 或用户已有改动。
- 所有新增能力都要有解释器层测试。
- 主线程统一处理冲突、更新中文覆盖报告，并运行 `go test ./...`。

## 第一阶段交付

1. 完成低耦合 API 的基础实现。
2. 更新 `docs/builtins-test-report.md`，把已补齐项从“待补齐”移动到“已覆盖/基础版”。
3. 全量测试通过。

## 后续增强

- `console.table` 先做可读文本输出，后续再增强格式对齐。
- `Date` 先使用本地时间模型，后续可补 UTC 系列方法。
- `RegExp` 先做 Go regexp 可支持的基础模式，后续再补 JS 风格 flag 和全局匹配语义。
- `Map/Set` 先做基础方法与 `size`，后续补迭代协议。
