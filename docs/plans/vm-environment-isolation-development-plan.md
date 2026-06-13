# GTS VM / 环境隔离能力开发计划

## 背景

`@std/web` 已通过 `webApp.handlerMu` 将进入 GTS 脚本 handler 链的请求串行化。这个修复可靠地保护了当前语言运行时，但代价是脚本层 HTTP handler 无法并行执行。

下一阶段目标不是简单移除锁，而是补齐 VM / 环境隔离能力，让每个请求可以在独立脚本运行态中执行，同时保留 HTTP server 的 Go 层并发能力。

## 当前实现判断

### 现有共享点

- `webApp` 保存 route 列表，每条 route 保存注册时传入的 `object.Object` handler。
- GTS 函数对象 `object.Function` 持有 `Env *object.Environment`，这是词法闭包环境。
- 普通函数调用会创建 `f.Env.NewScope()`，局部调用 scope 是新的，但父链仍指向原闭包环境。
- 模块缓存 `module.Cache` 按 VM 持有 `*object.Environment`，同一个 VM 内 import / require 的模块顶层变量天然共享。
- `VirtualMachine` 持有 object manager、async wait group、global const、importer、evaluator、argv 等运行时状态。
- `Hash`、`Array`、`Map`、`Set`、`Instance` 等对象内部容器大多不是并发安全结构。

因此，现在的并发风险不是 `req/res` 对象本身，而是多个请求同时执行同一批 handler 函数时会共享：

- handler 函数的闭包环境；
- 模块顶层状态；
- 被闭包引用的对象图；
- 同一个 VM 的 async / timer / importer / object manager 状态。

### handler 锁的真实语义

`webApp.handlerMu` 是“脚本入口全局互斥”，它保护了同一 `webApp` 下的所有 GTS handler 链。Go HTTP server 仍可接收连接、读 body、构造请求上下文，但进入 GTS VM 后串行。

这适合作为 P0 兜底策略，不适合作为长期并发模型。

## 隔离目标

1. 每个 HTTP 请求执行 handler 时拥有独立可变运行态。
2. 请求之间不共享普通脚本对象，除非显式使用 Go 层线程安全能力。
3. 模块顶层代码的执行、缓存、闭包绑定具备清晰策略。
4. 现有 CLI、SDK、REPL、非 web 标准库行为不被破坏。
5. 允许按配置从保守串行逐步切换到隔离并行。

## 方案比较

### 方案 A：给所有对象加锁

给 `Environment`、`Hash`、`Array`、`Map`、`Set`、`Instance` 等对象补锁，允许同一 VM 并发执行。

不推荐。原因：

- 需要覆盖解释器几乎所有读写路径，改动面大。
- 只能避免 Go data race，不能解决语言语义上的共享可变状态混乱。
- 异步、模块初始化、闭包捕获、原型/类实例仍会有复杂可见性问题。

### 方案 B：调用 handler 前深拷贝闭包环境

为 `object.Function` 增加 clone 能力：复制 `Function`、`Environment` 父链、普通对象图，然后在副本上执行请求。

可作为过渡，但不应作为最终方案。

优点：

- 接入 `@std/web` 较快。
- 能保护大多数闭包变量和普通对象。

缺点：

- 模块缓存、native module、GoObject、Promise、Channel、Timer、文件句柄等对象难以定义通用深拷贝语义。
- 函数默认参数、类方法绑定、原型链、循环引用对象图需要专门处理。
- 顶层模块副作用是否每请求重新执行不清晰。

### 方案 C：请求级 Runtime / VM 隔离

在 web server 启动时记录可重新加载应用的入口信息；每个请求从 VM 池取一个独立 VM，加载同一应用模块，在该 VM 内找到对应 handler 并执行。

推荐作为长期方向。

优点：

- VM、module cache、environment、object manager 完全隔离，语义清晰。
- 可以复用现有 `VirtualMachinePool` 思路，未来做 warm pool。
- 与 CLI / SDK 的“每次运行一个 VM + module cache”模型一致。

缺点：

- 需要把当前 runner 里的加载能力下沉为可复用 runtime snapshot / loader。
- 每请求冷启动成本较高，需要 warm pool 和编译/解析缓存逐步优化。
- app 注册 route 的过程要可重放，或者要把 route 规范从 handler 对象中抽离。

## 推荐路线

采用“C 为主，B 为辅助诊断工具”的路线：

- P0 保留当前 `handlerMu`，作为默认安全模式。
- P1 建立可复用的运行时实例化能力，不接 web 并行。
- P2 为 `@std/web` 增加 isolated 模式，先做到正确。
- P3 增加 warm VM pool，恢复可接受性能。
- P4 设计显式共享能力，避免用户依赖隐式共享闭包。

补充路线：如果目标是提高“单个 worker 内”的 I/O 并发，而不是请求间隔离，可并行推进 [`event-loop-worker-concurrency-development-plan.md`](event-loop-worker-concurrency-development-plan.md)。两条路线互补：

- `event-loop`：同一个 VM 仍由单 owner goroutine 串行执行脚本片段，但 HTTP/DB/timer/stream 等等待型任务可挂起并由 goroutine 后端并发等待。
- `isolated`：通过多 VM / 每请求 VM 隔离普通脚本状态，适合多核扩展和强隔离。

长期可组合为“多 event-loop worker 池”：每个 worker 内 I/O 高并发，worker 之间用 VM 隔离和显式共享 API 管理状态。

## 目标架构

### 核心抽象

新增运行时隔离包或对象层 API：

- `RuntimeTemplate`
  - 保存入口文件、工作目录、argv、typecheck、resolver root、插件/native module 配置。
  - 可创建独立 `RuntimeInstance`。
- `RuntimeInstance`
  - 拥有独立 `VirtualMachine`、`module.Cache`、`async.Pool`、root env。
  - 负责加载入口、执行 main 或指定 bootstrap。
- `ModuleSnapshot` 或 `ProgramCache`
  - 第一阶段可为空，直接重新 parse/eval。
  - 后续缓存 AST / 解析结果，避免每个 VM 重复 lex/parse。

### web 层改造

`webApp` 不再只保存 handler 对象，还应保存可重建 handler 的 route 描述：

- method；
- path；
- handler 在注册序列中的位置或导出名；
- middleware 类型；
- native middleware options。

短期可先支持两种模式：

- `mode: "shared-serial"`：当前行为，默认。
- `mode: "isolated"`：每个请求取独立 VM，重新加载 app 并执行匹配 route。

长期目标：

- app 启动时完成 route manifest 提取；
- 每个 isolated VM 启动时重放 route 注册；
- 请求执行时使用 isolated VM 内部的 handler 对象，而不是主 VM 的 handler 对象。

## 开发阶段

### P0：并发风险基线和回归测试

状态：当前应立即补齐。

任务：

- 保留 `webApp.handlerMu`。
- 增加 `@std/web` 并发回归测试，构造共享闭包计数器，证明串行模式不会产生竞态。
- 增加文档说明：当前 `@std/web` 的脚本 handler 是串行执行。

验收：

- `go test ./internal/stdlib` 通过。
- race 场景在串行模式稳定。

### P1：运行时实例化能力下沉

任务：

- 从 `cmd/gs` runner 和 `sdk.Runtime` 中抽出通用 loader。
- 提供类似：
  - `runtime.NewTemplate(options)`
  - `template.NewInstance()`
  - `instance.LoadEntry()`
  - `instance.Close()`
- 确保每个 instance 使用独立 `VirtualMachine` 和 `module.Cache`。
- 保持 CLI、SDK 当前 API 不变，内部迁移到新 loader。

验收：

- 现有 `cmd/gs` 和 `sdk` 测试通过。
- 新增测试证明两个 instance 加载同一模块时顶层变量互不影响。

### P2：对象 clone 能力最小集

任务：

- 增加 `object.CloneGraph` 作为调试/过渡能力。
- 支持：
  - primitives 直接复用；
  - `Array`、`Hash`、`Map`、`Set`、`Instance` 深拷贝；
  - `Function` 复制函数壳并重绑定 cloned env；
  - `Class` 复制 methods / fields / statics；
  - 循环引用通过 seen map 处理。
- 对不可安全 clone 的对象返回明确错误：
  - `GoObject`
  - `Promise`
  - `TimerId`
  - `Channel`
  - `WaitGroup`
  - native resource wrapper。

验收：

- 单元测试覆盖闭包、循环对象、类实例、不可 clone 对象报错。
- 该能力先不作为 web 默认并发路径。

### P3：`@std/web` isolated 模式

任务：

- 为 `web.createApp` 增加 options：
  - `createApp({ concurrency: "serial" })`
  - `createApp({ concurrency: "isolated" })`
  - 默认仍为 `serial`。
- 在 isolated 模式中，`listen` 捕获运行时模板信息。
- 每个请求：
  - 从 isolated VM pool 获取 instance；
  - 加载或复用该 instance 的 route registry；
  - 构造当前请求的 req/res；
  - 在该 VM 内执行匹配 handler 链；
  - 请求结束后清理或归还 instance。
- route 注册阶段和请求执行阶段分离，避免每个请求重复监听端口。

验收：

- 并发请求能同时进入脚本 handler。
- 顶层 `let counter = 0` 在 isolated 请求间不共享。
- 单请求内 middleware 链仍共享同一个 req/res。
- serial 模式行为保持不变。

### P4：warm VM pool 与生命周期

任务：

- `webApp` 持有 `RuntimeInstancePool`。
- app 启动时预热 N 个 isolated instance。
- instance 使用次数或空闲时间达到阈值后回收。
- 请求结束必须等待或取消该 VM 内 async work，避免泄漏。
- 增加指标：
  - pool size；
  - cold starts；
  - request VM checkout latency；
  - active script handlers。

验收：

- 并发 50/100 请求不串行阻塞。
- 无 goroutine / terminal session / timer 明显泄漏。
- 性能较 cold isolated 模式显著改善。

### P5：显式共享状态 API

任务：

- 提供线程安全共享状态模块，避免用户依赖隐式闭包共享：
  - `@std/shared.counter`
  - `@std/shared.map`
  - 或 `@std/atomic` / `@std/sync`。
- 明确 isolated 模式下模块顶层状态是每 VM 独立的。
- 文档给出迁移示例：
  - 原闭包 counter；
  - 改为外部 DB / shared atomic / Go native module。

验收：

- 用户能在 isolated web app 中显式维护跨请求状态。
- API 文档说明并发语义。

## 测试计划

### 单元测试

- `Environment` 父链 clone / instance 隔离。
- `module.Cache` 在不同 VM 中不共享。
- `Function.Env` 重绑定后默认参数、闭包变量、类方法正常。
- 不可 clone 对象错误路径。

### web 集成测试

- serial 模式：并发请求进入 handler 的最大并行数为 1。
- isolated 模式：并发请求进入 handler 的最大并行数大于 1。
- isolated 模式：模块顶层变量请求间隔离。
- middleware `next()` 行为不变。
- native middleware `json/text/static/proxy` 行为不变。

### race / 压测

- `go test -race ./internal/stdlib ./internal/object ./sdk ./cmd/gs`。
- 针对 `@std/web` 构造 100 并发请求。
- 检查 async goroutine、timer、stream response 是否在请求结束后清理。

## 兼容策略

- 默认保持 serial，避免破坏现有依赖共享闭包状态的 web app。
- isolated 作为 opt-in。
- 文档明确：
  - serial：同 app 脚本 handler 串行，共享模块/闭包状态。
  - isolated：请求间不共享普通脚本状态，需要显式共享 API。
- 后续版本可考虑将新项目模板默认设为 isolated，但不改变老项目默认。

## 风险清单

- 重新加载 app 时如果 `main()` 直接 `listen()`，isolated instance 不能重复监听端口，需要引入 bootstrap / route manifest 分离。
- native module 可能捕获 VM 内对象或 Go resource，需要逐个审计。
- 插件模块 `@plugin/` / `@host/` 的生命周期需要与 isolated VM 明确绑定。
- async 函数在请求结束后继续运行时，必须定义是否允许后台任务存活。
- web streaming 会拉长 VM checkout 时间，pool 需要避免被长连接耗尽。

## 建议优先级

1. 先补 P0 测试和文档，锁住当前安全边界。
2. 做 P1 runtime loader 下沉，这是所有隔离能力的地基。
3. 做 P3 isolated web 的最小正确版本，哪怕第一版 cold start。
4. 再做 P4 warm pool 性能优化。
5. 最后做 P5 显式共享 API，完善迁移体验。

## 第一轮可执行任务拆分

1. 新增 `internal/runtime` 或 `internal/execution` 包，抽出 CLI/SDK 共用的 source/file/module loader。
2. 新增测试：两个 runtime instance 加载同一脚本，模块顶层计数互不影响。
3. 给 `webApp` 增加 concurrency mode 字段，但默认 serial，不改变行为。
4. 在 `@std/web` 文档/API doc 中标注 serial handler 语义。
5. 写 isolated mode 技术预研 spike：证明同一 app route 可在两个 VM 内独立注册并执行。

## 当前进度

- 已给 `webApp` 增加 `concurrency` 字段和 `createApp({ concurrency: "serial" })` 解析骨架。
- 默认模式仍为 `serial`；`isolated` 在实现前会明确报错。
- 已补 `@std/web` 并发回归测试：并发 HTTP 请求下，最大同时进入脚本 handler 数必须为 1。
- 已补 SDK 层回归测试：两个独立 `Runtime` 加载同一模块文件时，模块顶层状态互不影响。
- 已补齐 native module API doc 登记缺口，避免 `--api_doc all` 在隔离开发过程中持续噪声失败。

## P1 最小切入点建议

下一步不急于整体重构 CLI runner。建议先在 SDK 侧抽出一个内部 `runtimeSession` 小结构，承载当前重复出现的 VM、async pool、module cache、resolver、rootDir 初始化逻辑：

1. 从 `sdk.Runtime.configure(baseDir)` 中提取 `newRuntimeSession(opts, baseDir)`。
2. 让 `RunSource`、`RunFile`、`CallExport` 先使用同一个 session 初始化路径，但保持公开 API 不变。
3. 给 session 增加 `Close/drain`，确保每个 session 的 async work 和 VM 资源有明确生命周期。
4. 等 SDK 侧稳定后，再把 `cmd/gs` runner 迁移到同一内部包。

这个切入点风险较低，因为 SDK 已经是“一个 Runtime 管一个 VM/cache”的清晰边界；先把这里整理成可复用实例化能力，再服务 `@std/web` isolated VM pool。
