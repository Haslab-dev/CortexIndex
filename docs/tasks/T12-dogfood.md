# T12 — Dogfooding & polish

**Status:** Done
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

- [x] Cortex indexes its own repo (51 files, 265 symbols); `cortex context` on its own code returns useful ranked output after test-symbol down-ranking polish.
- [x] Timings: full index 88 ms (51 files), incremental update 14 ms (0 files reindexed), symbol query 9 ms — all within PRD §25 targets.
- [x] Issue found (test fixtures outranking real symbols) fixed in `fix(T12)` commit 7221aa2.
