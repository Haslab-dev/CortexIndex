# T11 — Tests

**Status:** Done
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

- [x] `go test ./...` green (7 packages, ~1.5 s total, no network).
- [x] Every supported grammar family has extraction assertions (go, ts, python, rust, java, kotlin, c, cpp in extract_test.go).
- [x] Incremental behavior tests prove "only changed files re-indexed" (index package: skip/edit/delete/new-file).
