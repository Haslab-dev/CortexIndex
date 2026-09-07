# T12 — Dogfooding & polish

**Status:** Backlog
**Depends on:** T08, T11
**PRD sections:** §30 Success Criteria, §31 Design Principles

## Goal

Cortex indexes itself; real queries behave per PRD; rough edges filed/fixed.

## Scope

- `cortex init && cortex index` on this repo; `context`, `symbol`, `refs`, `deps`, `search` against real Go code.
- Check output quality, token estimates, edge cases (empty memory, fresh clone, missing binary assumptions).
- Benchmark notes: index time, update time after single edit, query latency (§25 targets: startup <100 ms, incremental = changed files only).
- Record a before/after exploration anecdote vs manual grep (§30 framing) in task notes.

## Acceptance criteria

- [ ] Cortex indexes its own repo without errors; `cortex context "how does the context command work"` returns genuinely useful output.
- [ ] Timing notes recorded (index/update/query).
- [ ] Issues discovered are fixed or filed into this kanban.
