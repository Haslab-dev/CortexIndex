# T07 — Context engine

**Status:** Done
**Depends on:** T05, T06
**PRD sections:** §14, §15 Context Command, §21 Ranking, §22 Token Optimization, §23 Source Context Strategy, §27 Failure Handling

## Goal

`cortex context "<task>"` — the flagship command. Layered retrieval → expansion → ranking → source extraction → budgeted Markdown for LLM consumption.

## Scope

- Keyword extraction from task (stopwords, camelCase splitting).
- Layer 1 memory: relevant `.cortex` md files/sections (§14 Layer 1).
- Layer 2 structural: symbol candidates via exact/prefix/parts/FTS/path/memory-mention; scoring per §21.
- Relationship expansion: callees/callers/inheritance one hop for top symbols.
- Layer 5 source: signature always; bodies only for top-N (§23: symbol + surrounding lines, never default whole file).
- Token budget (`--budget`, default ~6000): trim lowest-ranked source first, then memory.
- Fallback (§27): missing/empty index → lexical repo scan on disk; parse-failed files still searchable via files_fts; source always beats stale memory.

## Acceptance criteria

- [x] On fixture repo, `context "How does authentication work?"` includes memory excerpt, ranked symbols with locations/calls, source for top symbols, relevant files (TestBuildContextWithIndex).
- [x] Output fits budget; estimate printed (TestContextRespectsBudget, "~N tokens estimated" footer).
- [x] Missing index → lexical fallback results, labeled, no crash (TestContextWithoutIndexFallsBack).
- [x] Source-derived symbols always render independently of memory (separate sections; symbols layer never reads memory).
