# GTS Event Loop Worker 并发开发计划

## 背景

当前 `@std/web` 为保护同一个 VM、闭包环境和模块顶层状态，已经将脚本 handler 串行化。VM / 环境隔离计划可以通过多 VM 或每请求 VM 提高整体并发，但单个 worker 内仍存在一个问题：

- 如果 handler 在等待 HTTP、DB、文件、timer、stream 等 I/O，整个脚本 handler 链仍占着执行权。
- 单 worker 无法高效承载大量“等待中”的请求。

本计划提出另一条并发路线：**事件循环 + 单 VM owner + goroutine async backend**。

目标不是让多个 goroutine 同时执行同一个 VM，而是让单 worker 内的阻塞 I/O 交给 goroutine 池，完成后把 continuation 投递回 VM owner event loop。这样可以提升单 worker 的 I/O 并发，同时保持脚本对象图单线程一致性。

## 核心原则

1. 同一个 `VirtualMachine` 只能有一个 owner event loop goroutine 执行脚本代码。
2. 任何 `Environment`、`Hash`、`Array`、`Map`、`Set`、`Instance`、`Function.Env` 的读写都必须发生在 owner event loop 上。
3. I/O goroutine 不能直接读写可变 `object.Object` 图。
4. I/O goroutine 返回结果时，应返回 Go primitive / byte buffer / 明确可转移数据；由 event loop 转成 GTS object。
5. Promise resolve/reject、timer callback、web async handler continuation 都必须投递回 event loop。
6. CPU 密集脚本不在本方案内并行化；CPU 并行需要多 VM worker 池或进程池。

## 目标与非目标

### 目标

- 单 worker 内允许大量请求处于等待 I/O 状态。
- VM 内脚本片段仍串行执行，避免语言层 data race。
- `@std/web` 支持 async handler：handler 返回 Promise 时，请求保持打开，Promise settle 后继续响应。
- timers、HTTP client、stream 等逐步迁移到 event loop 调度。
- 为未来多 worker 池提供统一 worker 抽象。

### 非目标

- 不让多个 goroutine 同时执行同一个 VM。
- 不给所有对象加锁来实现共享内存并行。
- 第一阶段不实现完整 JS async/await continuation 重写。
- 第一阶段不保证 CPU 密集任务可抢占。

## 当前实现差距

当前已有基础：

- `VirtualMachine.Go(fn)` 可接入 spawner。
- `Promise` 有 pending/fulfilled/rejected 状态和 `Wait()`。
- `setTimeout`、`queueMicrotask`、`Promise.all` 等已使用 goroutine。
- `@std/web` 可通过 handler 锁保护脚本入口。

主要差距：

- goroutine 可以直接调用 `callTimerFunction` / `EvalNode`，没有 VM owner 约束。
- `Promise.Wait()` 是阻塞等待，不是 event loop continuation。
- `sleep(ms)` 直接 `time.Sleep`，会阻塞当前脚本执行。
- async I/O 完成后没有统一投递回 VM owner event loop。
- `@std/web` 不识别 handler 返回 Promise。

## 目标架构

```text
worker
  event loop goroutine
    owns VM
    owns Environment/Object graph
    runs script tasks and continuations

  async backend goroutine pool
    runs blocking I/O
    does not touch mutable GTS objects
    sends completion events back to event loop

  request table
    tracks active HTTP requests
    maps Promise/continuation to response writer
```

## 核心类型草案

```go
type EventLoopWorker struct {
    vm      *object.VirtualMachine
    ready   chan LoopTask
    ioPool  *async.Pool
    closing chan struct{}
}

type LoopTask struct {
    Name string
    Run  func(*object.VirtualMachine) object.Object
}

type AsyncOp struct {
    Name   string
    Run    func(context.Context) (AsyncResult, error) // goroutine pool
    Resume func(AsyncResult, error) object.Object     // event loop
}

type AsyncResult struct {
    Value any
}
```

约束：

- `LoopTask.Run` 可以访问 VM 和 GTS object。
- `AsyncOp.Run` 不可以访问 VM 和 GTS object。
- `AsyncOp.Resume` 回到 event loop 执行，再把 Go 结果转换为 GTS object。

## 开发阶段

### P0：事件循环骨架，不改变外部语义

任务：

- 新增 `internal/runtime` 或 `internal/eventloop` 包。
- 实现 `EventLoopWorker`：
  - `Post(task)`
  - `Run()`
  - `Stop()`
  - `RunSync(task)` 用于测试。
- `VirtualMachine` 增加 owner/scheduler 概念：
  - `SetScheduler(scheduler)`
  - `Post(fn)`
  - 保留 `Go(fn)` 兼容旧逻辑。
- 第一阶段不迁移 evaluator，只建立可测试的调度边界。

验收：

- 单元测试证明所有 posted task 按 FIFO 在同一 goroutine 执行。
- 单元测试证明 async backend completion 回到 event loop 后执行。
- 现有测试不受影响。

### P1：Promise settle 投递回 event loop

任务：

- 扩展 `object.Promise`：
  - 支持注册 fulfillment/rejection handler。
  - settle 后通过 VM scheduler enqueue continuation，而不是任意 goroutine 直接执行。
- `Promise.then` 的回调必须回到 event loop。
- `queueMicrotask(fn)` 改为向 event loop microtask queue 投递。

验收：

- Promise `.then` 顺序稳定。
- `queueMicrotask` 在当前 task 后、下一个 macro task 前执行。
- `Promise.all/race/allSettled` 不在 goroutine 中直接构造/修改 GTS object。

### P2：非阻塞 timers / sleepAsync

任务：

- 新增非阻塞 timer primitive：
  - `setTimeout` 到期后投递 callback 到 event loop。
  - `queueMicrotask` 进入 microtask queue。
  - 新增 `sleepAsync(ms) -> Promise` 或让 `@std/timers.sleep(ms)` 支持 Promise 模式。
- 保留旧 `sleep(ms)` 阻塞语义，避免破坏现有脚本；在 web async handler 中建议使用 async sleep。

验收：

- 多个 timer callback 不并发进入 VM。
- timer callback 与普通 task 顺序可预测。
- 旧 `sleep(ms)` 测试保持通过。

### P3：`@std/web` async handler

任务：

- `callWebHandlers` 识别 handler 返回 `*object.Promise`。
- Promise pending 时：
  - 不立即结束请求；
  - 保存 request context；
  - handler chain 的后续执行作为 continuation 入 event loop。
- Promise fulfilled：
  - 若 response 未写完，继续执行 next handler 或结束响应。
- Promise rejected：
  - 返回 500 或进入未来错误 middleware。
- HTTP 请求断开时取消对应 pending async op。

验收：

- async handler 可以并发挂起多个请求。
- 同一 worker 内最大“正在执行脚本片段”仍为 1。
- 100 个 sleepAsync 请求总耗时接近单个 sleep，而不是 100 倍 sleep。
- 断连请求不会泄漏 pending continuation。

### P4：HTTP client async backend

任务：

- 为 `@std/net/http/client` 增加 Promise API：
  - `requestAsync(options) -> Promise`
  - `getAsync(url, options?) -> Promise`
  - `postAsync(url, body?, options?) -> Promise`
- I/O goroutine 执行 `http.Client.Do` 和 body read/stream pump。
- 返回 event loop 后构造 response object。
- 对 streaming response，设计独立 stream actor，避免 I/O goroutine 直接写 GTS object。

验收：

- 多个 outbound HTTP 请求可并发等待。
- completion 顺序按实际完成时间入队，但 VM 执行仍串行。
- response object 构造只发生在 event loop。

### P5：stream / SSE / proxy async 化

任务：

- `res.stream` 支持异步 pump：
  - I/O goroutine 负责读 upstream；
  - 写 response 需要按 request actor 顺序执行。
- `web.proxy` 可使用 async HTTP client。
- SSE 长连接不占用 VM 执行权。

验收：

- 代理 50 条流式请求时，单 worker 不被单个流阻塞。
- response chunk 顺序正确。
- 客户端断连能取消 upstream。

### P6：公平性、取消、观测

任务：

- event loop 增加 microtask / macrotask 分层。
- 每轮限制最大 microtask drain，避免微任务饿死 I/O。
- request context cancellation 贯穿 AsyncOp。
- 增加指标：
  - ready queue length；
  - pending async ops；
  - active requests；
  - event loop tick duration；
  - longest task duration。

验收：

- 长 Promise 链不会永久饿死 timer/I/O。
- 慢 task 可被日志/指标定位。
- 取消路径无 goroutine 泄漏。

## `@std/web` 模式建议

建议未来支持三种模式：

```javascript
web.createApp({ concurrency: "serial" })
web.createApp({ concurrency: "event-loop" })
web.createApp({ concurrency: "isolated" })
```

- `serial`：当前安全模式，整个 handler 链串行执行。
- `event-loop`：单 VM owner event loop，I/O 可挂起，脚本片段串行。
- `isolated`：多 VM / 每请求 VM 隔离，请求间普通脚本状态不共享。

组合路线：

- 单 worker 高 I/O 并发：`event-loop`。
- 多核扩展：多个 `event-loop` worker 组成 worker pool。
- 强隔离：`isolated` 或多进程 worker。

## 测试计划

### event loop 单元测试

- FIFO task 顺序。
- microtask 优先级。
- macro task 不被无限 microtask 饿死。
- async backend 不在 event loop goroutine 外执行 GTS object mutation。

### Promise 测试

- `Promise.resolve().then(...)` 顺序。
- `Promise.all/race/allSettled` 回调均在 event loop。
- reject path 不直接 panic 到 goroutine。

### web 集成测试

- 100 个 async sleep handler 请求并发完成。
- 记录 `activeScript`，任何时刻不超过 1。
- 记录 `activeRequests`，可大于 1。
- 断连取消 pending request。

### race 测试

- `go test -race ./internal/eventloop ./internal/evaluator ./internal/stdlib`
- 专门压测 async web handler、HTTP client async、stream proxy。

## 风险与难点

- 当前 evaluator 是树遍历执行，真正 `await` continuation 需要 AST 执行状态保存；第一版可以先支持“handler 返回 Promise”而不实现完整 `await`。
- 现有 `Promise.Wait()` 使用广泛，迁移时需要保留同步等待路径，避免 CLI/测试行为突变。
- 任何 goroutine 直接创建并填充 `object.Hash/Array` 都可能破坏 owner 约束，需要逐步审计。
- streaming response 不能简单把所有 chunk 都排进 VM event loop，否则高吞吐流会拖慢脚本任务；需要 request actor 或 Go 层 stream pump。
- CPU 密集 handler 仍会阻塞 event loop，需要多 worker 或显式 `yield()` 才能改善。

## 推荐第一轮任务

1. 新增 `internal/eventloop` 包，实现单 owner event loop 和 async backend completion 投递。
2. 给 `VirtualMachine` 增加可选 scheduler，不改现有 `Go(fn)` 行为。
3. 写测试证明 event loop 内 GTS task 串行、async completion 可并发等待。
4. 新增 `@std/timers.sleepAsync(ms) -> Promise`，第一版只用于验证事件循环。
5. 在 `@std/web` spike 中支持返回 Promise 的 handler，并用 `sleepAsync` 证明单 worker 内请求等待可并发。

## 当前进度

- 已新增 `internal/eventloop.Worker` 骨架。
- 已实现 `Post` / `PostFunc` / `Run` / `RunSync` / `StartAsync`。
- 已验证 event loop task 按 FIFO 在同一个 owner goroutine 执行。
- 已验证 async backend 可在非 owner goroutine 等待，完成后回到 owner goroutine resume。
- 已给 `VirtualMachine` 增加可选 `SetScheduler` / `Post` 挂点；未设置 scheduler 时 `Post` 退化为旧 `Go` 行为。
- 已新增 `sleepAsync(ms) -> Promise`，并接入全局 builtin 与 `@std/timers.sleepAsync`。
- `sleepAsync` 的 timer 到期后通过 `vm.Post` settle Promise；如果 VM 接入 event loop scheduler，settle 会回到 owner loop。
- 尚未迁移 Promise chaining、`queueMicrotask`、`setTimeout`、`@std/web` handler 或 HTTP client；现有阻塞 `sleep(ms)` 语义保持不变。

## 与 VM 隔离路线的关系

两条路线互补：

- event-loop 提高单 worker 内 I/O 并发，保留同一个 VM 的共享模块/闭包语义。
- isolated / 多 VM 提高 CPU 并行和状态隔离能力。

最终推荐架构是：

```text
worker pool
  worker 1: event loop + VM
  worker 2: event loop + VM
  worker N: event loop + VM
```

每个 worker 内脚本状态单线程一致；worker 之间通过显式共享 API、DB、消息队列或 native thread-safe store 共享状态。
