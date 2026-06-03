# GoScript Development Plan

> Updated from the current repository state on 2026-06-03.
> This document turns the original roadmap into an executable plan based on what is already implemented.

---

## 1. Current State Summary

The project is no longer at the original M0 design stage. The core interpreter has working lexer, parser, AST, evaluator, runtime objects, builtins, async primitives, standard-library module registration, examples, documentation, a VS Code syntax extension, package distribution support, and a CLI under `cmd/gs`.

The main remaining gap is language/runtime completion: full module semantics, static scope resolution, type checking, richer errors, async/event-loop hardening, and alignment between the documented standard library and the implemented one.

### Implemented or Mostly Implemented

| Area | Status | Evidence |
|------|--------|----------|
| Lexer | Mostly done | `internal/lexer`, lexer tests pass |
| Parser | Mostly done | `internal/parser`, parser tests pass |
| AST | Mostly done | `internal/ast` |
| Evaluator core | Mostly done | `internal/evaluator`, evaluator tests pass |
| Functions / closures | Mostly done | evaluator tests cover functions and closures |
| Arrays / objects | Mostly done | array, object, JSON, string method files |
| Classes / inheritance | Mostly done | class evaluator tests |
| Exceptions | Partially done | `try/catch/finally` exists, rich Error objects/stack traces still incomplete |
| Async / Promise | Partially done | `Promise`, `async/await`, timers exist, but event-loop semantics need hardening |
| Native stdlib modules | Partially done | multiple `@std/*` modules register through `internal/module`, including fs/path/os/process/exec/db/http/socket/ws/crypto/buffer/timers |
| CLI | Mostly done | `cmd/gs` supports file execution, project `run`, `init`, `pack`, `dist`, `bundle`, timeout protection, and workers |
| `require` integration | Mostly done | CLI wires `require(path)` with source/package/native resolution and module cache |
| Package distribution | Mostly done | `.gspkg` packing, nested package reads, and executable embedding exist |
| Example verification | Basic done | `cmd/gs` tests run a stable example list |
| VS Code extension | Basic done | `vscode-extension` syntax package exists |

### Missing or Incomplete

| Area | Gap |
|------|-----|
| REPL | No interactive shell |
| Module runtime | Basic `import/export` works; full semantics are incomplete |
| Type checker | Type annotations parse, but `internal/typechecker` is missing |
| Static resolver | `internal/module.Resolver` exists for module paths; a separate static scope resolver for declarations, `this`, and `super` is still missing |
| Error model | Dedicated Error subclasses and stack traces are not complete |
| C-style `for` edge cases | Some `for (let i = ...; ...)` forms still need parser/evaluator hardening |
| Docs alignment | Several deeper spec docs still describe target behavior rather than current behavior |
| Example verification | Stable coverage exists, but examples still need clearer stable/pending/manual grouping as capabilities grow |
| Embedding API | No stable public facade for Go hosts |

---

## 2. Planning Principles

1. Make the interpreter runnable before adding more language surface.
2. Turn examples into executable regression tests as soon as the CLI exists.
3. Treat `import/export`, `require`, and native modules as one integration milestone.
4. Keep type checking optional and layered after runtime execution is stable.
5. Keep documentation tied to acceptance tests so user-facing promises stay honest.

---

## 3. Revised Milestones

### P0: Project Alignment and Build Baseline

**Goal:** Establish a reliable baseline and remove confusion from outdated roadmap statements.

**Scope:**

- Keep `go test ./...` green.
- Document the current implemented/missing matrix.
- Decide whether `docs/roadmap.md` remains historical or becomes the live roadmap.
- Update README once CLI behavior is real.

**Acceptance:**

- `go test ./...` passes.
- This plan exists and is linked from the README or roadmap.
- Open gaps are tracked as concrete milestones.

**Current status:** Basic alignment started. README now points to this plan and describes current CLI behavior.

---

### P1: CLI Runner

**Goal:** Provide the `gs` command so scripts can be executed from the repository.

**Current status:** Mostly implemented.

**Scope:**

- Add `cmd/gs/main.go`.
- Support direct file execution:

```bash
go run ./cmd/gs examples/01-basics.gs
```

- Support project execution from `project.toml`:

```bash
go run ./cmd/gs run
```

- Register global builtins before evaluation.
- Print parser/runtime errors with source location when available.
- Wait for pending async work that is tracked by the runtime.
- Protect script execution with `--timeout` to avoid runaway examples.
- Add flags:
  - `--version`
  - `--check-types` as accepted but initially guarded if the type checker is not implemented
  - `--workers N` for async pool size if useful
  - `--timeout D` for maximum script runtime

**Acceptance:**

- `go build ./cmd/gs` succeeds.
- `go run ./cmd/gs main.gs` runs the root script.
- Basic examples can be run manually.
- CLI has focused tests or at least a smoke script.

**Dependencies:** Current lexer/parser/evaluator.

**Risk:** Existing examples use features that may be only partially implemented. Start with a small verified subset.

---

### P2: Example Regression Suite

**Goal:** Convert examples and docs examples into a repeatable quality gate.

**Current status:** Stable example suite exists in `cmd/gs` tests and `examples/README.md` lists verified examples.

**Scope:**

- Add a test or script that runs selected `.gs` examples through the CLI.
- Classify examples:
  - `stable`: should pass in CI
  - `pending`: documents intended behavior but not yet enforced
  - `manual`: requires network, timers, or external services
- Start with:
  - `examples/01-basics.gs`
  - `docs/examples/hello.gs`
  - `docs/examples/fib.gs`
  - `docs/examples/counter.gs`

**Acceptance:**

- Stable example suite runs in one command.
- CI candidates are deterministic and do not require network.
- Any failing example has a linked missing-feature note.

**Dependencies:** P1 CLI.

---

### P3: Module Runtime

**Goal:** Make multi-file GoScript projects work.

**Current status:** Basic runtime support is implemented for named exports, export specifiers, default expression exports, aliases, namespace imports, `require`, native modules, and file cache.

**Scope:**

- Implement runtime loader around `internal/module.Cache`.
- Wire `require(path)` into CLI-hosted evaluation.
- Support native module resolution:
  - `@std/exec`
  - `@std/net/http/client`
  - `@std/net/http/server`
  - `@std/net/socket/client`
  - `@std/net/socket/server`
  - `@std/net/ws/client`
  - `@std/net/ws/server`
- Expand `import/export` evaluation on top of the shared module runtime.
- Define export behavior:
  - named exports
  - default export
  - `module.exports` / `exports` compatibility if retained
- Add circular import behavior explicitly.

**Acceptance:**

- A two-file script can import a function and run it.
- `require("./file.gs")` caches the module and does not re-execute it twice.
- Native module imports resolve consistently.
- Parser-level `import/export` no longer evaluates as a silent no-op for basic named/default cases.

**Dependencies:** P1 CLI.

**Risk:** Current `bundle` code looks for `require(...)`, while parser supports ES-style import/export. Keep bundling separate from runtime loading until behavior is stable.

---

### P4: Runtime Semantics Hardening

**Goal:** Close the largest differences between intended language behavior and current evaluator behavior.

**Current status:** `const` reassignment, undeclared assignment, and top-level `break`/`continue` now produce errors. Loop-local `break`/`continue` is handled for the covered loop cases.

**Scope:**

- Continue hardening loop/control-flow edge cases, especially C-style `for`.
- Error object fields:
  - `name`
  - `message`
  - `stack`
- Error subclasses:
  - `Error`
  - `TypeError`
  - `RangeError`
  - `ReferenceError`
  - `SyntaxError`
- `console.error`, `console.warn`, `console.info`, and basic `console.assert`.
- Align `parseInt` with the documented strict behavior or update docs.

**Acceptance:**

- Dedicated evaluator tests for each semantic rule.
- Existing tests remain green.
- Docs no longer overclaim unimplemented behavior.

**Dependencies:** P1 recommended, but many items can be implemented independently.

---

### P5: Static Scope Resolver

**Goal:** Add a static analysis pass for scope and `this` correctness.

This is separate from `internal/module.Resolver`, which already resolves module specifiers and package paths.

**Scope:**

- Create `internal/resolver` or an equivalent static-analysis package.
- Track lexical scopes and declarations.
- Reject duplicate declarations where the language requires it.
- Reject undefined variable reads/writes before evaluation when possible.
- Validate `this` and `super` usage.
- Prepare metadata for better runtime errors.

**Acceptance:**

- Resolver unit tests pass.
- CLI can run with resolver enabled by default or behind a temporary flag.
- Runtime behavior does not regress for closures and classes.

**Dependencies:** P4 decisions on strict runtime semantics.

---

### P6: Optional Type Checker

**Goal:** Make parsed type annotations useful through `--check-types`.

**Scope:**

- Create `internal/typechecker`.
- Support primitive annotations:
  - `number`
  - `string`
  - `boolean`
  - `null`
  - `undefined`
  - `any`
  - `void`
- Support arrays and simple object shapes.
- Support function parameter and return annotations.
- Support union types if AST already represents them reliably.
- Decide which checks happen statically and which stay runtime assertions.

**Acceptance:**

- `go run ./cmd/gs --check-types docs/examples/types.gs` reports meaningful errors or succeeds intentionally.
- Type checker errors include file/line/column.
- Running without `--check-types` preserves dynamic behavior.

**Dependencies:** P1 CLI, P5 resolver recommended.

---

### P7: Async and Event Loop Completion

**Goal:** Make async behavior predictable enough for examples and real scripts.

**Scope:**

- Clarify Promise chaining behavior:
  - `then`
  - `catch`
  - `finally` if supported
- Define microtask ordering.
- Make timers return cancellable IDs if docs keep `clearTimeout` / `clearInterval`.
- Avoid runaway `setInterval` in CLI runs.
- Add async example tests that do not depend on network.
- Decide whether `await` blocks the evaluator or models continuation more faithfully.

**Acceptance:**

- `examples/11-async.gs` or a trimmed stable async example passes.
- Promise rejection propagates through `await`.
- CLI exits cleanly after tracked async work completes.

**Dependencies:** P1 CLI.

---

### P8: Standard Library Completion

**Goal:** Align builtins and `docs/builtins.md`.

**Scope:**

- Audit implemented methods against docs.
- Split builtins into:
  - guaranteed v0.1
  - experimental
  - planned
- Fill the highest-value gaps:
  - console methods
  - Math constants and common functions
  - Object helpers
  - Array static helpers
  - URI helpers
- Add unit tests for each supported builtin.

**Acceptance:**

- `docs/builtins.md` matches implementation.
- Builtin test coverage covers success and common type errors.

**Dependencies:** P4 for semantic choices.

---

### P9: REPL

**Goal:** Make `gs` without a file open an interactive shell.

**Scope:**

- Multi-line input for blocks/functions/classes.
- Persistent environment across entries.
- `.exit`, `.help`, `.load`.
- Friendly display of returned values and errors.

**Acceptance:**

- `go run ./cmd/gs` starts the REPL.
- Basic declarations persist between commands.
- Syntax errors do not terminate the session.

**Dependencies:** P1 CLI.

---

### P10: Embedding API

**Goal:** Make the interpreter pleasant to use from Go applications.

**Scope:**

- Add a public facade package if desired.
- Provide an API similar to:

```go
engine := goscript.New()
result, err := engine.EvalString(ctx, source)
```

- Allow hosts to:
  - inject globals
  - register native modules
  - capture stdout/stderr
  - set working directory
  - limit async workers

**Acceptance:**

- Go embedding example builds and runs.
- API docs explain the lifecycle and error model.

**Dependencies:** P1, P3, P4.

---

## 4. Recommended Execution Order

1. P1 CLI Runner
2. P2 Example Regression Suite
3. P3 Module Runtime
4. P4 Runtime Semantics Hardening
5. P8 Standard Library Completion
6. P7 Async and Event Loop Completion
7. P5 Static Scope Resolver
8. P6 Optional Type Checker
9. P9 REPL
10. P10 Embedding API

The ordering favors fast feedback. Once the CLI and example suite exist, every later milestone has a concrete way to prove progress.

---

## 5. Immediate Sprint Plan

### Sprint 1: Make It Run

**Deliverables:**

- `cmd/gs/main.go`
- File execution
- Project `run` command using `project.toml`
- Builtin registration
- Basic error printing

**Current status:** Done at baseline level.

**Validation:**

```bash
go test ./...
go build ./cmd/gs
go run ./cmd/gs main.gs
go run ./cmd/gs examples/01-basics.gs
```

### Sprint 2: Make Examples Trustworthy

**Deliverables:**

- Stable example list
- Example runner test/script
- README updates for actual supported commands

**Current status:** Done at baseline level; continue expanding coverage when features stabilize.

**Validation:**

```bash
go test ./...
go test ./cmd/gs
```

### Sprint 3: Make Modules Real

**Deliverables:**

- `require` loader callback in CLI
- Runtime file cache
- Native module resolution
- Initial `import/export` evaluation

**Current status:** Mostly done for basic source/package/native modules; full semantics and edge cases remain.

**Validation:**

```bash
go test ./...
go run ./cmd/gs examples/module-app.gs
```

---

## 6. Documentation Updates Needed

| File | Update |
|------|--------|
| `README.md` | Keep quick-start and feature status aligned with current CLI |
| `docs/roadmap.md` | Historical document; keep clearly marked as outdated |
| `docs/builtins.md` | Separate implemented from planned builtins |
| `docs/language-spec.md` | Make compatibility matrix reflect actual implementation |
| `examples/README.md` | Add stable/pending/manual example status |

---

## 7. Definition of Done

For each milestone:

- Code is implemented.
- New or affected tests pass.
- At least one script/example demonstrates the behavior when applicable.
- Documentation is updated in the same change.
- Known limitations are explicit rather than silent.
