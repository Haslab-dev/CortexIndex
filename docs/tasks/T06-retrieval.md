# T06 — Retrieval commands

**Status:** Backlog
**Depends on:** T02, T03, T04
**PRD sections:** §14 Retrieval Engine (Layers 2–3, 5), §16 CLI, §21 Context Ranking

## Goal

Precise structural + lexical retrieval over the index, Markdown stdout, smallest useful unit first (PRD §31.5).

## Scope

- `search "<q>"`: symbols (FTS + exact/prefix on name) and files (FTS + snippets), ranked, limited.
- `symbol "<name>"`: exact match (else FTS) → signature, kind, location, doc, parent, calls, called-by; `--src` prints source (chunks/disk).
- `refs "<name>"`: call refs by name across repo with `file:line`, enclosing symbol.
- `deps "<name>"`: callees (resolved), callers, file imports.
- Query sanitizer for FTS5 (quoting, OR-join, prefix fallback); camelCase part matching.
- Ranking signals per §21: name match, path match, FTS bm25, ref degree, recency.

## Acceptance criteria

- [ ] `symbol AuthService.login` returns location + calls + callers on a fixture repo.
- [ ] `refs`, `deps` return consistent call-graph views both directions.
- [ ] `search` returns relevant symbols AND files/snippets; garbage query returns empty gracefully.
- [ ] Missing index → clear "run `cortex index`" message, non-zero-but-helpful exit (no stack traces).
