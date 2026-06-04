# GoScript REPL Development Plan

> Date: 2026-06-04

## Goal

Make `gs` without a script start an interactive GoScript shell while preserving the existing embedded-executable behavior.

## Phase 1: MVP

Scope:

- Keep appended `.gspkg` executable startup ahead of REPL startup.
- Start REPL when no script, inline code, or subcommand is provided.
- Reuse the existing lexer, parser, evaluator, module resolver, and async pool.
- Keep one VM and one top-level environment for the whole session.
- Support `.exit`, `.quit`, `.help`, and `.load <file>`.
- Print non-`undefined` expression results.
- Report parse/runtime errors without terminating the session.
- Detect basic multiline input using delimiter, string, and block-comment balance.

Acceptance:

- `go run ./cmd/gs` opens the REPL.
- Declarations persist across entries.
- Function/block input can span multiple lines.
- `.load` evaluates a file into the current session.
- Focused REPL tests pass.

## Phase 2: Usability

Scope:

- Add command history and line editing through a small cross-platform dependency or terminal abstraction.
- Improve prompt behavior for incomplete syntax that is not delimiter-based.
- Add `.clear`, `.vars`, and `.reset` if they prove useful.
- Make display formatting friendlier for strings and structured objects.

## Phase 3: Hardening

Scope:

- Add REPL-specific parser recovery tests.
- Verify Windows terminal behavior manually.
- Document REPL commands in `README.md`.
- Consider context cancellation for long-running interactive entries.
