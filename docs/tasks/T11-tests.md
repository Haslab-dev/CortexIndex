# T11 — Tests

**Status:** Backlog
**Depends on:** T03, T04, T06 (context engine covered via e2e)
**PRD sections:** §28 V1 Scope (quality gate)

## Goal

Fast, deterministic `go test ./...` covering the risky parts: extraction correctness, incremental semantics, retrieval behavior.

## Scope

- `lang`: table-driven fixtures per language (symbols/kinds/ranges/imports/calls/inheritance).
- `store`: schema idempotence, replace/remove cleanliness, FTS queries incl. camelCase parts.
- `index`: unchanged-skip, single-file edit, deletion sweep, full rebuild equivalence (temp-dir fixture repos).
- `retrieve`: search/symbol/refs/deps on fixture repo; context output sections + budget; FTS sanitizer.
- `lexsearch`: fallback path returns useful windows without a DB.

## Acceptance criteria

- [ ] `go test ./...` green, no network, <60 s.
- [ ] Every supported language has at least one extraction assertion.
- [ ] Incremental behavior tests prove "only changed files re-indexed".
