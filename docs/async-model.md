# GoScript 异步模型：Promise / async-await / 事件循环

> 本文档解释 GoScript 解释器中**异步执行**的工作方式，包括：
> - 为什么树遍历解释器也能"挂起"和"恢复"；
> - `Promise` 的状态机与微任务调度；
> - `async/await` 的实现策略（continuation 续延）；
> - 与宿主 Go 的事件循环桥接。

---

## 1. 总览

```
                      脚本入口
                         │
                         ▼
                ┌────────────────────┐
                │ 同步代码（Top-level）│
                │  —— 完全在主线程   │
                │  —— 调用栈 = 求值栈│
                └────────────────────┘
                         │ 第一条 await
                         ▼
                ┌────────────────────┐
                │ 创建 Promise，挂起  │
                │ 注册 continuation  │
                └────────────────────┘
                         │
                         ▼
            ┌──────────────────────────┐
            │     事件循环开始运转      │
            │  1. 微任务队列（Promise）│
            │  2. 宏任务（timer/io）   │
            └──────────────────────────┘
                         │  所有任务结束
                         ▼
                     进程退出
```

---

## 2. 动机

JS 的 `async/await` 本质是**在 `await` 处暂停函数执行**，等待 `Promise` 决议后再继续。
这看似"协程"，但在 V8 内部是基于 **microtask continuation** 实现的——并不真的挂起 OS 线程。

GoScript 是树遍历解释器，没有字节码栈帧的天然边界，但只要把"await 之后的剩余工作"打包为一个**闭包**，就可以在微任务里再调它。技术上的关键点：

1. **闭包捕获环境**——`env` 已经被 `*Function` 持有，可以暂停后复用。
2. **控制流对象**——`BREAK`/`CONTINUE`/`RETURN` 通过 `panic/recover` 机制实现，await 续延也走相同通道。
3. **微任务**——`Promise.then` 注册的回调由事件循环统一调度。

---

## 3. 运行时数据

```go
package async

type State int

const (
    PENDING   State = iota
    FULFILLED
    REJECTED
)

type Promise struct {
    mu        sync.Mutex
    state     State
    value     object.Object
    reason    object.Object
    handlers  []Handler        // then 注册的回调，决议时入队为微任务
    catches   []Handler        // catch 注册的回调
    id        int64
    rt        *Runtime
}

type Handler struct {
    onFulfilled func(object.Object) object.Object
    onRejected  func(object.Object) object.Object
}

type Task struct {
    fn   func() object.Object
    env  *object.Environment
    rt   *Runtime
}

type Runtime struct {
    micro        chan Task    // 微任务队列
    macro        chan Task    // 宏任务队列（timer/io 包装）
    pendingIO    int64        // 未完成 I/O 计数
    pendingTimer int64
    shutdown     chan struct{}
}
```

---

## 4. Promise 状态机

### 4.1 创建

```javascript
let p = new Promise((resolve, reject) => {
  // 同步执行
  if (ok) resolve(42);
  else reject(new Error("oops"));
});
```

```go
func newPromise(rt *Runtime, executor object.Object) *Promise {
    p := &Promise{state: PENDING, rt: rt}
    resolve := func(v object.Object) { p.resolve(v) }
    reject  := func(v object.Object) { p.reject(v)  }
    // 同步执行 executor，捕获 panic
    func() {
        defer func() {
            if r := recover(); r != nil {
                if err, ok := r.(*object.Error); ok {
                    p.reject(err)
                } else {
                    p.reject(object.NewError("panic: %v", r))
                }
            }
        }()
        executor.(*object.Function).Call(rt.GlobalEnv(), resolve, reject)
    }()
    return p
}
```

### 4.2 决议

`resolve(v)` / `reject(r)` 只能**第一次**调用生效：

```go
func (p *Promise) resolve(v object.Object) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.state != PENDING { return }
    // 如果 v 是 Promise，则"展开"（thenable）
    if pp, ok := v.(*Promise); ok {
        pp.then(p.resolve, p.reject)
        return
    }
    p.state = FULFILLED
    p.value = v
    p.flush()
}

func (p *Promise) reject(r object.Object) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.state != PENDING { return }
    p.state = REJECTED
    p.reason = r
    p.flush()
}

func (p *Promise) flush() {
    for _, h := range p.handlers {
        p.rt.enqueueMicro(h)  // 入微任务队列
    }
    for _, h := range p.catches {
        p.rt.enqueueMicro(h)
    }
    p.handlers = nil
    p.catches = nil
}
```

### 4.3 链式

```javascript
p.then(v => v + 1)
 .then(v => v * 2)
 .catch(e => -1);
```

```go
func (p *Promise) then(onF, onR object.Object) *Promise {
    next := newPromise(p.rt, emptyExecutor)
    p.mu.Lock()
    if p.state == PENDING {
        p.handlers = append(p.handlers, Handler{
            onFulfilled: wrap(next, onF),
            onRejected:  wrap(next, onR),
        })
    } else {
        p.rt.enqueueMicro(Task{fn: func() object.Object {
            if p.state == FULFILLED && onF != nil {
                return wrap(next, onF)(p.value)
            } else if p.state == REJECTED && onR != nil {
                return wrap(next, onR)(p.reason)
            } else {
                return p // 透传
            }
        }})
    }
    p.mu.Unlock()
    return next
}
```

---

## 5. await 的实现

### 5.1 朴素策略（v0.1）

```javascript
async function f() {
  let x = compute();
  let y = await fetch(x);
  return process(y);
}
```

解释器遇到 `await expr`：

1. 求值 `expr`，得到 `*Promise` `p`。
2. 若 `p` 已决 → 同步取值，继续。
3. 若 `p` 待定 → 把"await 后**剩余语句序列**"打包成 continuation，调用 `p.then(cont, contErr)`。
4. `p.then` 返回**新 Promise** `next`；原 `async` 函数也返回 `next`。

### 5.2 关键：续延（Continuation）构造

在 Resolver 阶段，识别 `await` 节点并标注它在所属 `BlockStmt` 中的位置。

求值时，遇到 `AwaitExpr`：

```go
func evalAwait(node *ast.AwaitExpr, env *object.Environment) object.Object {
    v := Eval(node.Value, env)
    if !isPromise(v) {
        return v
    }
    p := v.(*object.Promise)
    if p.IsResolved() {
        if p.State() == FULFILLED { return p.Value() }
        panic(p.Reason())  // 同步路径下抛错
    }

    // 构造续延
    afterAwait := buildContinuation(node, env)
    returnedPromise := object.NewPromise(...)
    p.Then(afterAwait, errHandler(returnedPromise, env))
    return returnedPromise
}
```

`buildContinuation`：

```go
// "续延" = "从 await 之后到 BlockStmt 末尾的所有语句 + 收尾 RETURN"
func buildContinuation(await *ast.AwaitExpr, env *object.Environment) func(object.Object) object.Object {
    return func(result object.Object) object.Object {
        // 1. 把 result 绑定到 await 的"匿名临时变量"
        // 2. 从 await 的父 BlockStmt 切片执行剩余语句
        // 3. 若遇到 return/throw，正确包装
    }
}
```

> 实现细节中，需要在 `BlockStmt` 求值时维护"当前执行到第几条语句"，遇 await 时把"剩余切片"作为 closure 引用。
> 这是一段相对精巧的代码，集中在 `internal/evaluator/await_continuation.go`。

### 5.3 多 await 与控制流

```javascript
async function f() {
  let a = await p1;
  if (a < 0) {
    let b = await p2;
    return b;
  }
  return a * 2;
}
```

续延 closure 在每次被调用时**重新建立新一轮执行**——如果它内部还有 `await`，就再挂起一次。
这种递归续延的方式天然支持：

- 循环中的 `await`（每次循环体都是一个新续延）
- `try/catch` 跨 await（错误以 `panic(*Error)` 在 closure 中抛，由外层 `try` 捕获）
- `return` 跨 await（捕获 `*ReturnValue` 后包装到新 `Promise` 的 `resolve`）

---

## 6. 事件循环

### 6.1 主循环

```go
func (rt *Runtime) RunLoop() {
    for {
        // 1. 排空微任务
        rt.drainMicro()
        if rt.idle() {
            return
        }
        // 2. 等待一个宏任务或 IO 完成
        select {
        case t := <-rt.macro:
            safeRun(t)
        case <-rt.shutdown:
            return
        case <-time.After(rt.idleTimeout):
            return
        }
        // 3. 再排空微任务
        rt.drainMicro()
    }
}
```

### 6.2 任务来源

| 任务 | 来源 | 类别 |
|------|------|------|
| `Promise.then` 回调 | Promise 决议时 | 微 |
| `queueMicrotask(fn)` | 用户显式 | 微 |
| `setTimeout(fn, ms)` | `time.AfterFunc` | 宏 |
| `setInterval(fn, ms)` | 重复注册 | 宏 |
| `fetch(...)` 完成 | `http.Do` 回调 | 宏 |
| 自定义 `EventEmitter.emit` | 用户事件总线 | 宏（可配置） |

### 6.3 公平性

- 同一 `Promise` 链上的多个 `.then` 顺序入队，按注册顺序执行。
- 不同 Promise 之间按"先到先服务"（FIFO）。
- 微任务优先于宏任务，避免宏任务饥饿。
- **宏任务 I/O 多路复用**：使用 Go 的 `net.Listener` + `http.Client.Do` 异步接口，I/O 完成时把回调包装为宏任务入队。

---

## 7. 与 Go 宿主集成

### 7.1 启动流程

```go
// cmd/gs/main.go
func runFile(path string) {
    src, _ := os.ReadFile(path)
    l := lexer.New(path, string(src))
    p := parser.New(l)
    prog := p.ParseProgram()
    if errs := p.Errors(); len(errs) > 0 {
        for _, e := range errs { fmt.Fprintln(os.Stderr, e) }
        os.Exit(1)
    }
    rt := async.NewRuntime()
    e := evaluator.New(rt)
    rt.AttachEvaluator(e)
    result := e.Eval(prog, e.GlobalEnv())
    if err, ok := result.(*object.Error); ok {
        fmt.Fprintln(os.Stderr, err.Inspect())
        os.Exit(1)
    }
    rt.RunLoop()
}
```

### 7.2 setTimeout

```go
func builtinSetTimeout(env *object.Env, args ...object.Object) object.Object {
    fn := args[0]
    ms := int(args[1].(*object.Number).Value)
    var extra []object.Object
    if len(args) > 2 { extra = args[2:] }
    id := env.Runtime().SetTimeout(func() {
        safeCall(env, fn, extra)
    }, time.Duration(ms)*time.Millisecond)
    return &object.Number{Value: float64(id)}
}
```

### 7.3 fetch（示例：基于 net/http）

```go
func builtinFetch(env *object.Env, args ...object.Object) object.Object {
    url := args[0].(*object.String).Value
    p := object.NewPromise(env.Runtime(), emptyExecutor)
    go func() {
        resp, err := http.Get(url)
        if err != nil {
            p.Reject(object.NewError("fetch: %v", err))
            return
        }
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        p.Resolve(&object.Instance{
            Class: responseClass,
            Props: map[string]object.Object{
                "status": &object.Number{Value: float64(resp.StatusCode)},
                "ok":     &object.Boolean{Value: resp.StatusCode < 400},
                "text": &object.Builtin{Fn: func(...object.Object) object.Object {
                    return &object.String{Value: string(body)}
                }},
                "json": &object.Builtin{Fn: func(...object.Object) object.Object {
                    return parseJSON(string(body))
                }},
            },
        })
    }()
    return p
}
```

### 7.4 退出条件

事件循环在以下条件同时满足时退出：

- 微任务队列空
- 宏任务队列空
- `pendingIO == 0` 且 `pendingTimer == 0`

> 用户的 `process.exit(0)` 直接终止整个 Go 进程，跳过退出条件。

---

## 8. 错误处理边界

| 错误类型 | 行为 |
|---------|------|
| `async` 函数中同步 `throw` | `Promise` 被 `reject` |
| `async` 函数中 `await` 的 `Promise` 失败 | 续延中 `panic` 异常 → 转 `reject` |
| `await` 之后未捕获错误 | 通过 `try/catch` 链传递；最外层由 `Runtime.OnUnhandledRejection` 捕获 |
| 宏任务同步 `throw` | 记录并报告，**不**中断事件循环 |
| 宏任务 I/O 错误 | 进入 `reject` 路径 |

```go
func (rt *Runtime) OnUnhandledRejection(p *Promise) {
    msg := fmt.Sprintf("UnhandledPromiseRejection: %s", p.Reason().Inspect())
    fmt.Fprintln(os.Stderr, msg)
    if rt.StrictMode() {
        rt.Shutdown()
    }
}
```

---

## 9. 与 ECMAScript 行为的差异

| 项 | ES | GoScript | 备注 |
|----|----|---------|------|
| 微任务 vs 宏任务顺序 | 同 | 同 | |
| 顶层 `await` | 仅 modules 允许 | v0.1 **不**允许 | 在 module 顶层未实现 |
| `Promise.all` 失败行为 | 取首个 reject | 同 | |
| 未处理拒绝 | Host 决定 | 默认打印，配置 strict 后退出 | |
| `Promise.race` 空数组 | 永久 pending | 同 | |

---

## 10. 性能与限制

- **不要**在紧密循环里频繁 `await` 微小 Promise：每次 await 至少一次微任务调度 + 一次闭包调用。
- **避免** `await await await` 的串行化，对独立任务使用 `Promise.all`。
- **长任务**应分批 `await` 出去，把控制权交回事件循环，避免饿死宏任务。

---

## 11. 调试与可观测性

```javascript
console.trace("checkpoint");
```

- 异步栈追踪：解释器在续延创建时记录"前一帧"信息，组成逻辑调用栈。
- 死循环 / 死锁检测：可选 `Runtime.SetMaxTicks(N)`，超过 N 抛出 `RangeError`。

---

> 至此异步子系统设计完成。如需了解语法层面，参见 [`docs/language-spec.md`](language-spec.md) 第 10 节；
> 运行时类型参见 [`docs/builtins.md`](builtins.md) 第 12 节。
