# T08 — CLI wiring

**Status:** Done
**Depends on:** T04, T05, T06, T07
**PRD sections:** §16 CLI, §25 Runtime

## Goal

Single native binary `cortex`, all PRD §16 commands, fast startup, no external server (PRD §31.7).

## Scope

- Commands: `init`, `index [--full]`, `update`, `search`, `symbol [--src]`, `refs`, `deps`, `context [--budget N]`, `memory`, `watch [--interval]`, `version`, `help`.
- Repo-root discovery: cwd, else nearest ancestor containing `.cortex/`; `--dir` override.
- Markdown-only stdout (agent-facing); errors to stderr; sensible exit codes.
- `watch`: poll-based incremental update loop, SIGINT-safe.

## Acceptance criteria

- [x] Every §16 command exists with documented flags; `cortex help` lists them (init, index, update, search, symbol, refs, deps, context, memory, watch, version).
- [x] `symbol` query measured at 9 ms on indexed repo (§25 target met).
- [x] Commands outside an initialized repo auto-init on index and give guidance otherwise; verified e2e (no panics).
- [x] `go build -o cortex ./cmd/cortex` produces one binary; runtime is local-only (pure-Go SQLite, no network calls).
