# Cortex — Implementation Kanban

Progress tracker for the Cortex (Codebase Memory Engine) V1 build, derived from [prd.md](./prd.md).

- One card per task. Detailed scope + acceptance criteria live in [tasks/](./tasks/).
- Update a card's column **and** its task file's `Status` field whenever work happens.
- A card may only move to **Done** when every acceptance checkbox in its task file is checked with real evidence (build/test output, command run, file created).

## Columns

### 🟦 Backlog

| ID | Task | Depends on |
|----|------|------------|
| — | — | — |

### 🟨 In Progress

| ID | Task | Notes |
|----|------|-------|
| — | — | — |

### 👀 Review / QA

| ID | Task | Notes |
|----|------|-------|
| — | — | — |

### ✅ Done

| ID | Task | Evidence |
|----|------|----------|
| [T01](./tasks/T01-scaffold.md) | Project scaffold + dependency verification | `go mod init cortex`; smoke test: tree-sitter parsed Go source + FTS5 MATCH returned row (`modernc.org/sqlite` v1.58.0, `smacker/go-tree-sitter` dd81d9e) |
| [T02](./tasks/T02-store.md) | SQLite store layer | `feat(T02)` commit 7630335; tests prove idempotent replace, clean remove, FTS camel-part lookup, snippets, ref attribution |
| [T03](./tasks/T03-languages.md) | Tree-sitter extractors (11 languages) | `feat(T03)` commit 7102aa0; per-language assertion tests green; AST quirks pinned by dump tests |
| [T04](./tasks/T04-indexer.md) | Indexer (full + incremental) | `feat(T04)` commit d2e126c; tests prove 0-reindex on unchanged, single-file edit, deletion sweep, parse-failure FTS fallback |
| [T05](./tasks/T05-memory.md) | Markdown memory layer | `feat(T05)` commit 44c10cb; init idempotence + config parsing tested; edits survive re-init |
| [T06](./tasks/T06-retrieval.md) | Retrieval commands | `feat(T06)` commit 1fcbf13; search/symbol/refs/deps tested on fixture incl. calls, callers, imports, --src |
| [T07](./tasks/T07-context.md) | Context engine | `feat(T07)` commit 7f6912f; layered output, ranking, budget trimming, no-index fallback tested |
| [T08](./tasks/T08-cli.md) | CLI wiring | `feat(T08)` commit aca3823; all 11 commands verified e2e on fixture repo incl. deletion sweep and fallback |
| [T09](./tasks/T09-skill.md) | Agent Skill | `docs(T09,T10)` commit aadbe6c; skills/cortex/SKILL.md with frontmatter, workflow, etiquette |
| [T10](./tasks/T10-readme.md) | README & docs | `docs(T09,T10)` commit aadbe6c; README.md with quickstart, commands, config, principles |
| [T11](./tasks/T11-tests.md) | Tests | `go test ./...` green across lang/store/index/memory/retrieve/context/lexsearch; `test(T11)` commit below |
| [T12](./tasks/T12-dogfood.md) | Dogfooding & polish | indexed own repo: 51 files/265 symbols in 88 ms; update 14 ms (0 reindexed); query 9 ms; `fix(T12)` ranking polish commit |
| [T13](./tasks/T13-build-install.md) | Makefile, native build/install/PATH | `make check` green; staged install produced mode 0755 binary and `cortex 0.1.0`; `make env` emitted sourceable PATH export |
| [T14](./tasks/T14-skill-installer.md) | Skill installer | `feat(T14)` commit 919e8dc; destination/idempotence/conflict tests green; CLI installed all five native skills plus shared path |
| [T15](./tasks/T15-agents.md) | `AGENTS.md` generator/updater | `feat(T15)` commit 324b7bb; create/append/replace/idempotence tests and CLI smoke test green |
| [T16](./tasks/T16-overview.md) | Intent-aware overview and ranking | commits 8487e90/ea70fbd/3be39f7; overview sections, module summaries, entry-point ranking, and tests green |
| [T17](./tasks/T17-bootstrap.md) | Deterministic bootstrap and validation | commits a01e8ca/1c092a7; proposal/write flow, valid provenance frontmatter, memory check, and tests green |
| [T18](./tasks/T18-resolver.md) | Binary discovery and agent startup | commits 486d08a/3c77dda; resolver tests and synchronized Skill/AGENTS instructions |
| [T19](./tasks/T19-benchmark.md) | Local deterministic benchmark | commit 51e802d; `make bench` emits baseline/Cortex metrics with local token estimates |
| [T20](./tasks/T20-codebase-brain.md) | Codebase brain foundation | typed claims, proposed filtering, live `history`, work records, scoped preferences, and real-repo smoke validation |

## V1 scope recap (from PRD §28)

- **Core:** Go binary, `init`, `index`, `update`, Tree-sitter parsing, SQLite, FTS5, symbol indexing, import/reference indexing, incremental updates.
- **Memory:** `project.md`, `architecture.md`, `conventions.md`, module + decision Markdown.
- **Retrieval:** `search`, `symbol`, `refs`, `deps`, `context`.
- **Agent:** Codebase Skill, Markdown stdout.
- **Explicitly out:** embeddings, vector DB, MCP, hosted service.

## Naming note

PRD refers to the binary as `codebase` and the memory dir as `.mycodebase/`. Per project decision the application is named **Cortex**: binary `cortex`, memory dir `.cortex/`. All acceptance criteria below use the Cortex names.
