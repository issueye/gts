# GoScript AI Agent 语言能力补齐计划

> 生成日期：2026-06-02。
> 本文基于参考项目 `E:\codes\github\pi` 与当前 GoScript 仓库 `E:\codes\gts` 的分析，重点回答：如果 **agent 本体用 GoScript 脚本实现**，语言层、运行时和标准库还需要补齐哪些能力。

---

## 1. 目标边界

本计划不以“用 Go 重写一个 Pi”为目标，而是以“让 GoScript 足够强，可以用脚本写出类似 Pi 的 AI coding agent”为目标。

因此分工应为：

| 层级 | 实现位置 | 说明 |
|---|---|---|
| 语言语法和求值语义 | Go 解释器 | 模块、异步、错误、对象、迭代、类型/schema 等基础能力 |
| 原生标准库 | Go 原生模块 | 文件、路径、进程、HTTP、SSE、WebSocket、stream、buffer、crypto、database 等宿主能力 |
| Agent 框架 | GoScript 脚本 | `Agent`、agent loop、provider 适配、tool registry、session、资源发现 |
| Coding tools | 优先 GoScript 脚本，必要时调用原生标准库 | `read/write/edit/bash/grep/find/ls` 等工具 |
| CLI 外壳 | Go + GoScript | Go 负责加载入口脚本和宿主参数，agent 行为由脚本库完成 |

一句话：**Go 层补“语言能做什么”，脚本层写“agent 怎么做”。**

---

## 2. Pi 项目的语言需求抽象

`E:\codes\github\pi` 的核心能力可以抽象为一组脚本语言必须能表达和承载的能力：

| Pi 能力 | 对 GoScript 的语言/运行时要求 |
|---|---|
| 多 provider LLM 调用 | HTTP client、SSE/WebSocket streaming、JSON、headers、超时、重试、API key 读取 |
| 流式消息 | async iterator、`for await`、stream reader、增量 JSON 组装 |
| 工具调用 | 函数一等公民、对象/数组、schema 校验、异常、并发/取消 |
| Agent loop | async/await 稳定、队列、事件分发、可组合模块 |
| 会话持久化 | fs/path/os/process、JSONL 读写、时间/uuid/hash |
| Coding tools | 文件读写、目录遍历、进程执行、正则/搜索、diff/patch 辅助 |
| 数据库访问 | SQLite/PostgreSQL/MySQL/MSSQL 原生驱动、统一 query/exec API、连接生命周期 |
| Skills/Prompt templates | 文件发现、Markdown 文本处理、模板渲染、路径规范化 |
| 扩展机制 | 动态模块加载、稳定 module cache、插件入口约定、脚本隔离/错误边界 |
| TUI/RPC | stdin/stdout stream、JSONL framing、终端输入输出、可取消任务 |

这些需求表明：GoScript 目前最该补的不是先写 agent，而是补齐 **异步 I/O、模块、标准库、错误、schema、stream、文件/进程能力**。

---

## 3. 当前 GoScript 基础

当前仓库已经具备以下基础：

| 领域 | 当前状态 |
|---|---|
| CLI | `cmd/gs` 可执行脚本和项目 `run`，支持超时与 worker 数 |
| 语言核心 | 词法、解析、AST、求值器、函数、闭包、类、对象、数组已有基础实现 |
| 异步 | 已有 `Promise`、`async/await`、timer 和 worker pool，但语义需加固 |
| 模块 | 已有 `require`、module cache、native module registry 和基础 `import/export` |
| 标准库雏形 | 已有 `@std/exec`、HTTP client/server、Socket、WebSocket |
| JSON | 已有 `JSON.parse/stringify` 基础实现 |
| 文档 | 已有语言规范、异步模型、builtins、roadmap、development plan |

这些能力足以作为 agent 脚本运行时的起点，但还不足以写出稳定的 Pi-like agent。

### 3.1 本轮已补齐的能力

截至 2026-06-02，本轮已完成第一批语言层和宿主标准库补齐：

| 能力 | 状态 | 说明 |
|---|---|---|
| 结构化错误 | 已完成基础版 | `Error`、`TypeError`、`RangeError`、`ReferenceError`、`SyntaxError` 支持 `name/message/stack`，运行时错误和脚本 `throw/catch` 可区分传播 |
| Promise 链式语义 | 已完成基础版 | 支持 `new Promise()`、`then/catch/finally`、reject 传播、async 函数错误转 rejected Promise、`await` rejected Promise 进入 `catch` |
| `await` 解析 | 已完成基础版 | parser 支持 `await` 表达式，配合 async/Promise 测试通过 |
| C 风格 `for` 循环 | 已修复基础缺陷 | 修复 `for (let i = 0; ...)` 试探解析吞 token 的问题，闭包和工具 registry 循环可稳定运行 |
| Agent 模块别名 | 已完成基础版 | `@agent/*` 映射到项目根目录下 `scripts/agent/*`，并支持默认 `.gs` 扩展 |
| `@std/path` | 已完成增强版 | `join/resolve/relative/normalize/dirname/basename/extname/isAbs/toSlash/fromSlash/matches/parse/format/splitList/sep/delimiter` |
| `@std/fs` | 已完成增强版 | `readFileSync/writeFileSync/readTextSync/writeTextSync/appendFileSync/writeFileAtomicSync/existsSync/readdirSync({withFileTypes})/walkSync/globSync/copyFileSync/rmSync/mkdtempSync/realpathSync/lstatSync/mkdirSync/statSync/renameSync/unlinkSync` |
| `@std/process` | 已完成增强版 | `argv/argv0/env/envObject/pid/cwd/chdir/execPath/getenv/setenv/unsetenv/uptime/hrtime/version/exit` |
| `@std/os` | 已完成增强版 | `platform/arch/eol/type/release/homedir/tmpdir/hostname/cpus/userInfo` |
| `@std/crypto` | 已完成基础版 | `randomUUID/sha256/randomBytes` |
| `@std/schema` | 已完成基础版 | JSON Schema 子集：`type/properties/required/items/enum/additionalProperties/min/max` 等 |
| `@std/db` | 已完成增强版 | 统一 `open/exec/query/queryOne/prepare/begin/ping/close`，支持事务、预编译语句、连接池配置；SQLite 使用 `modernc.org/sqlite` no-cgo 驱动 |
| 脚本工具库 | 已完成最小版 | `@agent/tools/registry`、`@agent/tools/files` 已能完成工具注册、schema 校验、文件读写和目录列表 |
| 脚本 coding tools | 已完成最小版 | `@agent/tools/bash`、`@agent/tools/grep`、`@agent/tools/coding` 已可由脚本执行 shell、纯文本搜索并聚合工具 |
| 脚本 session | 已完成最小版 | `@agent/session/jsonl` 已可写入和读取 JSONL 会话记录 |
| 脚本 agent loop | 已完成最小版 | `@agent/core/agent` + `@agent/llm/fake` 已能完成“provider 产生 tool call -> registry 调用工具 -> provider 返回最终消息”的两轮闭环 |
| 脚本 smoke | 已完成基础版 | `scripts/agent/smoke_import.gs`、`smoke_std.gs`、`smoke_env.gs`、`smoke_schema.gs`、`smoke_tools.gs`、`smoke_coding_tools.gs`、`smoke_agent_loop.gs` 可验证脚本层使用标准库、工具库和最小 agent loop |

这些交付仍然是“基础版”：它们已足够让脚本开始实现 coding tools、session 和 tool schema。L3 已开始补齐 HTTP streaming、SSE 和 stream 抽象；真实 provider 适配仍应由 `.gs` 脚本实现。

---

## 4. 语言层缺口矩阵

### 4.1 P0：必须先补齐

| 能力 | 当前问题 | 为什么 agent 需要 |
|---|---|---|
| 稳定 `Promise`/`async` | Promise 链、reject、微任务顺序、top-level 等待仍需验证 | Agent loop、HTTP 调用、工具执行都依赖异步 |
| Async iterator / `for await` | 当前未形成标准流式消费模型 | LLM streaming、SSE、RPC 都需要增量消费 |
| 完整模块语义 | `import/export`、循环依赖、native import 仍需加固 | agent 框架要拆成多个 `.gs` 模块 |
| 结构化错误 | `Error` 子类、`stack`、file/line 不完整 | provider/tool/session 错误必须可诊断 |
| JSON 和对象稳定性 | `JSON.stringify`、对象枚举、深层结构仍需加强 | provider payload、tool result、session JSONL 都依赖 |
| 文件/路径标准库 | `@std/fs`、`@std/path` 已有增强版，仍需 watch、权限细节、更完整 glob 规则 | read/write/edit/session/skills 都依赖 |
| HTTP streaming/SSE | 已有基础版，仍需 async iterator、abort、背压和更完整超时模型 | LLM provider 流式输出的底层能力 |
| 进程执行增强 | `@std/exec` 已有，但缺少 cwd/env/timeout/stream/cancel 完整模型 | `bash` tool 必须可靠可控 |
| Schema 校验 | 已有 JSON Schema 子集，仍需默认值、格式校验和更完整错误聚合 | 工具调用参数必须校验 |
| 数据库标准库 | 已有 `@std/db` 增强版，仍需命名参数、schema introspection、类型细化 | Agent 可用脚本实现数据库工具和数据查询任务 |

### 4.2 P1：Agent MVP 需要

| 能力 | 说明 |
|---|---|
| `AbortController` / `AbortSignal` | 统一取消 HTTP、进程、timer、stream、agent turn |
| Stream 抽象 | `ReadableStream` 或简化版 async iterable stream |
| Buffer / bytes / base64 | 处理二进制、图片、HTTP body、clipboard/image tool result |
| 正则与文本处理 | grep/search、模板处理、路径匹配、输出裁剪 |
| `Date` / 时间工具 | session timestamp、timeout、耗时统计 |
| `crypto` 基础 | 已有 uuid/hash/randomBytes，后续补 HMAC/base64 组合能力 |
| `process` / `os` | 已有 env、envObject、argv、argv0、execPath、cwd、uptime、hrtime、platform、type、release、userInfo、homedir、tmpdir、eol、cpus 等增强能力 |
| 稳定数组/字符串方法 | map/filter/reduce/slice/split/join/startsWith/includes 等 agent 代码常用方法 |
| 动态加载约定 | `import(path)` 或受控动态 require，用于扩展和技能加载 |

### 4.3 P2：完整 coding agent 需要

| 能力 | 说明 |
|---|---|
| 包/库组织 | 类似 `.gspkg` 或 `project.toml` 的脚本库入口和依赖声明 |
| 文件 watch | 文件触发型扩展、自动 reload |
| diff/patch 标准库 | edit tool、会话导出、变更摘要 |
| glob 标准库 | 文件发现、exclude/include、skills/prompts 搜索 |
| Markdown/模板工具 | skills、prompt templates、session export |
| 资源隔离 | 扩展运行边界、错误隔离、可选权限策略 |
| 类型检查器 | 可选，但有助于大型 agent 脚本维护 |

---

## 5. 推荐脚本化架构

GoScript 应补齐底层能力，然后用脚本实现 agent 库。推荐结构：

```text
cmd/gs
  继续作为通用 GoScript CLI

cmd/gsa
  可选：AI agent 专用入口
  主要职责是加载 scripts/agent/main.gs

internal/runtime
  语言运行时能力增强

internal/stdlib
  @std/fs
  @std/path
  @std/process
  @std/os
  @std/http
  @std/sse
  @std/stream
  @std/schema
  @std/buffer
  @std/crypto
  @std/db

scripts/agent/
  core/agent.gs
  core/events.gs
  core/messages.gs
  llm/openai.gs
  llm/anthropic.gs
  tools/read.gs
  tools/write.gs
  tools/bash.gs
  tools/grep.gs
  tools/find.gs
  tools/ls.gs
  session/jsonl.gs
  resources/skills.gs
  resources/context_files.gs
  cli/print_mode.gs
  cli/json_mode.gs
  cli/rpc_mode.gs
```

其中 Go 层只负责让脚本有足够能力：

- 能发 HTTP/SSE 请求。
- 能流式读取响应。
- 能读写文件和执行进程。
- 能校验 JSON schema。
- 能处理错误、取消、超时。
- 能稳定 import 脚本模块。

Agent 行为则在 `scripts/agent` 中完成。

---

## 6. 语言能力补齐路线

### L0：运行时可靠性基线

**目标：** 让长时间 agent 脚本不会因为基础语义不稳定而失控。

**工作项：**

- 加固 `Promise.then/catch/finally`。
- 明确并测试微任务/宏任务顺序。
- 支持 top-level await，至少在模块入口中支持。
- 完善 async 函数 reject 传播。
- 增加 unhandled rejection 处理。
- 完善 `Error`、`TypeError`、`ReferenceError`、`SyntaxError`。
- 错误对象带 `message/name/stack/file/line/column`。

**验收：**

- 异步、错误、timer 相关测试进入 `go test ./...`。
- 一个脚本可以稳定执行多轮 async HTTP/tool 调用。

### L1：模块和脚本库能力

**目标：** 让 agent 框架可以拆成多个 GoScript 文件维护。

**工作项：**

- 完整支持 named/default/namespace import/export。
- 明确循环依赖行为。
- native module 与脚本 module 使用同一 resolver。
- 支持项目根路径、相对路径、`@std/*`、`@agent/*`。
- 增加模块 cache、重复加载测试。
- 支持动态加载插件的受控接口。

**验收：**

- `scripts/agent/core/agent.gs` 可以 import provider、tools、session 模块。
- 同一模块只执行一次。
- native module 和脚本 module 行为一致且可测试。

### L2：文件、路径、进程标准库

**目标：** 让 coding tools 可以用脚本实现。

**模块：**

| 模块 | 能力 |
|---|---|
| `@std/fs` | 读写文件、目录遍历、带类型目录项、walk、glob、copy、rm、mkdtemp、realpath、lstat、stat、mkdir、rename、unlink、原子写 |
| `@std/path` | join、resolve、relative、normalize、dirname、basename、extname、isAbs、toSlash、fromSlash、matches、parse、format、splitList、sep、delimiter |
| `@std/process` | argv、argv0、env、envObject、cwd、chdir、execPath、uptime、hrtime、version、exit、pid |
| `@std/os` | platform、arch、eol、type、release、homedir、tmpdir、hostname、cpus、userInfo |
| `@std/exec` | cwd、env、timeout、stream stdout/stderr、kill、exitCode |

**当前状态：**

- `@std/fs`、`@std/path`、`@std/process`、`@std/os` 已完成同步增强 API。
- `@std/fs` 已支持文本读写、追加写、原子写、带类型目录项、递归 walk、glob、copy、临时目录、真实路径、lstat 和递归/强制删除。
- 下一步应补 `@std/fs.watch`、权限细节、更完整 glob 规则，以及 `@std/exec` 的 timeout/env/cwd 一体化选项。

**验收：**

- `read/write/bash/ls/find` 工具可以主要用 GoScript 写。
- shell 输出可流式读取并可被取消。

### L3：HTTP、SSE、Stream

**目标：** 只补通用 HTTP streaming、SSE、stream 抽象，让 LLM provider 继续由脚本实现。

**工作项：**

- `@std/http/client` 提供：
  - `request`
  - `fetch`
  - headers
  - method/body
  - timeout
  - abort signal（后续）
  - streaming body
- `@std/sse` 提供：
  - EventSource parser
  - 按事件增量输出
  - `[DONE]` 处理
- `@std/stream` 提供：
  - 同步 reader 基础版
  - text chunk / line reader
  - async iterable reader（后续）
  - bounded accumulator

**验收：**

- GoScript 脚本可以从 HTTP streaming response 中读取 chunk、文本行和 SSE 事件。
- 脚本 provider 可以基于 `@std/sse` 自行解析 OpenAI 兼容、Anthropic 或其他 provider 的 streaming payload。
- Go 层不包含任何真实 provider 逻辑。

**当前状态：**

- `@std/stream` 已提供 `fromString/read/readText/readLine/readAll/close`。
- `@std/net/http/client.stream()` 已返回带 `body` stream 的响应对象。
- `@std/sse` 已提供 `reader(stream).next/readAll` 和 `parse(text)`。
- 已通过本地 HTTP streaming + SSE 测试和 `scripts/agent/smoke_stream_sse.gs`。

### L4：Schema、JSON、Buffer、Crypto

**目标：** 让工具调用和会话数据可靠结构化。

**工作项：**

- `@std/schema`：
  - object/string/number/boolean/array/enum
  - required
  - additionalProperties
  - default
  - 简洁错误路径
- 强化 `JSON.stringify/parse`：
  - pretty print
  - stable stringify 可选
  - unsupported value 策略明确
- `@std/buffer`：
  - bytes
  - base64
  - utf8 encode/decode
- `@std/crypto`：
  - randomUUID
  - sha256
  - randomBytes
- `@std/db`：
  - SQLite/PostgreSQL/MySQL/MSSQL 驱动注册
  - open/exec/query/queryOne/ping/close
  - SQLite 使用 no-cgo 驱动
  - 事务、预编译语句、连接池配置
  - 命名参数、schema introspection 后续补齐

**当前状态：**

- `@std/schema` 已支持 tool 参数校验常用子集，但还未支持 `default` 写回、`format`、`oneOf/anyOf`。
- `@std/crypto` 已支持 `randomUUID/sha256/randomBytes`，后续应补 base64、HMAC、hex 编解码与 Buffer/bytes 互操作。
- `@std/db` 已支持统一数据库 API、事务、预编译语句和连接池配置；SQLite 已通过内存库 smoke 和测试，PostgreSQL/MySQL/MSSQL 已注册驱动，实际连接测试依赖外部服务。

**验收：**

- tool 参数校验错误能指出字段路径。
- session JSONL 可稳定写入/读取。
- provider payload 能稳定序列化。
- SQLite no-cgo 查询可在无外部数据库服务时通过 smoke。

### L5：脚本化 Agent 标准库

**目标：** 在 GoScript 中实现最小 agent framework。

**脚本模块：**

| 模块 | 说明 |
|---|---|
| `@agent/llm/openai` | OpenAI 兼容 provider |
| `@agent/core` | `Agent`、agent loop、事件分发 |
| `@agent/tools` | tool registry、schema validation、tool result |
| `@agent/coding-tools` | read/write/bash/grep/find/ls |
| `@agent/session` | JSONL session |
| `@agent/resources` | AGENTS.md、skills、prompt templates |

**当前状态：**

- `@agent/tools/registry` 已完成最小工具注册表，支持 `register/list/get/call`，调用前使用 `@std/schema` 校验参数。
- `@agent/tools/files` 已完成最小文件工具：`read_file`、`write_file`、`list_dir`，并通过 workspace 路径边界检查避免越界访问。
- `@agent/session/jsonl` 已完成最小 JSONL 会话记录。
- `@agent/tools/bash`、`@agent/tools/grep`、`@agent/tools/coding` 已完成最小 coding tools 聚合。
- `@agent/core/agent` 和 `@agent/llm/fake` 已跑通两轮 fake tool-call loop。
- provider 层应继续保持脚本实现。Go 层 L3 只补 HTTP streaming、SSE、stream 等通用底层能力，不内置任何真实 LLM provider。
- 下一步应继续补 `@agent/core/events`、脚本 provider streaming 解析、tool-call 参数增量组装和取消/超时模型。

**目标 API：**

```javascript
import { Agent } from "@agent/core";
import { openai } from "@agent/llm/openai";
import { createCodingTools } from "@agent/coding-tools";

const agent = new Agent({
  model: openai.model("gpt-4.1"),
  systemPrompt: "You are a careful coding agent.",
  tools: createCodingTools({ cwd: process.cwd() }),
});

agent.on("message_update", event => {
  if (event.kind === "text_delta") print(event.text);
});

await agent.prompt("分析当前项目结构。");
```

**验收：**

- Agent loop、provider、tools 均由 `.gs` 脚本实现。
- Go 层仅提供标准库和解释器能力。

---

## 7. MVP 开发顺序

### Sprint 1：异步和错误稳定

**交付物：**

- Promise 链式行为测试。
- async/await reject 传播测试。
- Error 对象和 stack trace。
- top-level await 设计或初步实现。

**验证：**

```bash
go test ./...
go run ./cmd/gs examples/11-async.gs
```

### Sprint 2：模块系统和脚本库结构

**交付物：**

- 完整 import/export 基础语义。
- `@std/*` 和 `@agent/*` resolver 设计。
- `scripts/agent` 目录和最小模块骨架。

**验证：**

```bash
go test ./...
go run ./cmd/gs scripts/agent/smoke_import.gs
```

### Sprint 3：文件/路径/进程标准库

**交付物：**

- `@std/fs`（增强版已完成）
- `@std/path`（增强版已完成）
- `@std/process`（增强版已完成）
- `@std/os`（增强版已完成）
- 增强 `@std/exec`

**验证：**

```bash
go test ./...
go run ./cmd/gs scripts/agent/smoke_std.gs
go run ./cmd/gs scripts/agent/smoke_env.gs
```

### Sprint 4：HTTP/SSE/Stream

**交付物：**

- streaming HTTP response（基础版已完成）。
- SSE parser（基础版已完成）。
- stream 抽象（同步基础版已完成，async iterable 后续）。
- abort/timeout 支持（timeout 基础版已支持，abort 后续）。

**验证：**

- 本地 fake SSE server 测试。
- `go run ./cmd/gs scripts/agent/smoke_stream_sse.gs`
- 无 API key 的脚本 provider parser 单元测试。

### Sprint 5：Schema 和脚本 Provider 解析

**交付物：**

- `@std/schema`（基础版已完成）。
- `scripts/agent/llm/*` 由脚本实现 provider 适配。
- streaming text 和 tool-call assembly 由脚本实现。

**验证：**

- `go run ./cmd/gs scripts/agent/smoke_schema.gs`
- fake streaming fixture 测试。
- 可选真实 API 手动测试。

### Sprint 6：脚本化 Agent Loop 和 Coding Tools

**交付物：**

- `scripts/agent/core/agent.gs`（最小版已完成）
- `scripts/agent/tools/*.gs`（registry 和文件工具最小版已完成）
- `scripts/agent/session/jsonl.gs`（最小版已完成）
- print mode 原型

**验证：**

- `go run ./cmd/gs scripts/agent/smoke_tools.gs`
- `go run ./cmd/gs scripts/agent/smoke_coding_tools.gs`
- `go run ./cmd/gs scripts/agent/smoke_agent_loop.gs`
- fake provider 触发 `read` tool。
- agent 用脚本完成“读文件 -> 返回总结”的两轮对话。

---

## 8. 完成定义

GoScript 达到以下状态时，即可认为语言层具备实现 Pi-like agent 的能力：

1. Agent loop 完全可以用 `.gs` 脚本表达。
2. Provider streaming 可以用 `.gs` 脚本消费和解析。
3. Tool schema 校验可以由 `.gs` 脚本调用标准库完成。
4. Coding tools 主要由 `.gs` 脚本实现，只依赖标准库提供宿主能力。
5. Session、skills、prompt templates、context files 都可以由 `.gs` 模块实现。
6. Go 层不需要内置具体 agent 逻辑，只负责解释器、标准库和 CLI 启动。
7. 最小 agent 示例可以运行：

```bash
go run ./cmd/gs scripts/agent/main.gs -p "分析这个项目"
```

第一版应优先做到：**异步可靠、模块可靠、HTTP/SSE 可流式、文件/进程可控、schema 可校验、agent loop 可脚本化**。
