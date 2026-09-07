# T08 — CLI wiring

**Status:** Backlog
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

- [ ] Every §16 command exists with documented flags; `cortex help` lists them.
- [ ] Startup + `symbol` on indexed repo feels instant (target <100 ms excluding initial index, §25).
- [ ] Commands run outside a Cortex-initialized repo fail with guidance, not panics.
- [ ] `go build` produces one static-ish binary; no Python/network at runtime.
