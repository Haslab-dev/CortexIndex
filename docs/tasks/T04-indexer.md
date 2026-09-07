# T04 — Indexer

**Status:** Backlog
**Depends on:** T02, T03
**PRD sections:** §20 Incremental Indexing, §25 Performance, §27 Failure Handling

## Goal

`cortex index` (full rebuild) and `cortex update` (incremental) that never re-index unchanged files (PRD §31.6).

## Scope

- Walk repo from root honoring default ignores (`.git`, `.cortex`, `node_modules`, `vendor`, build dirs, lockfiles, minified/binary/media, `.env`/key files) plus `config.md` ignore globs; size cap per file.
- Incremental pass: sha256 per file vs `files` table → skip unchanged, `ReplaceFile` changed, `RemoveFile` deleted (deletion sweep).
- Full pass: wipe derived tables first, then same pipeline.
- One reused parser per language; per-batch transactions; stats (indexed/skipped/deleted/symbols, duration).
- `watch` = periodic incremental scan (stdlib ticker, no extra deps).

## Acceptance criteria

- [ ] Second `update` with no changes reports 0 indexed, 0 deleted (hash skip proven).
- [ ] Edit one file → `update` touches only that file's rows; others untouched.
- [ ] Delete a file → `update` removes its rows.
- [ ] `index --full` rebuilds identical stats from scratch.
- [ ] Indexing a mid-size tree completes with no parse-error aborts (failures → FTS-only).
