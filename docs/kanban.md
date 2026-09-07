# Cortex — Implementation Kanban

Progress tracker for the Cortex (Codebase Memory Engine) V1 build, derived from [prd.md](./prd.md).

- One card per task. Detailed scope + acceptance criteria live in [tasks/](./tasks/).
- Update a card's column **and** its task file's `Status` field whenever work happens.
- A card may only move to **Done** when every acceptance checkbox in its task file is checked with real evidence (build/test output, command run, file created).

## Columns

### 🟦 Backlog

| ID | Task | Depends on |
|----|------|------------|
| [T02](./tasks/T02-store.md) | SQLite store layer (schema, FTS5, incremental replace/delete) | T01 |
| [T03](./tasks/T03-languages.md) | Tree-sitter symbol/relationship extractors (11 languages) | T01 |
| [T04](./tasks/T04-indexer.md) | Indexer: walk, hash, full + incremental update, deletion sweep | T02, T03 |
| [T05](./tasks/T05-memory.md) | Markdown memory layer: `init` scaffold, config, memory inspection | T01 |
| [T06](./tasks/T06-retrieval.md) | Retrieval commands: `search`, `symbol`, `refs`, `deps` | T02, T03, T04 |
| [T07](./tasks/T07-context.md) | Context engine: layered retrieval, ranking, token budget, fallback | T05, T06 |
| [T08](./tasks/T08-cli.md) | CLI wiring: all commands, flags, `watch`, graceful fallbacks | T04–T07 |
| [T09](./tasks/T09-skill.md) | Agent Skill (`skills/cortex/SKILL.md`) | T08 |
| [T10](./tasks/T10-readme.md) | README + usage docs | T08 |
| [T11](./tasks/T11-tests.md) | Unit + end-to-end tests (per-language fixtures) | T03, T04, T06 |
| [T12](./tasks/T12-dogfood.md) | Dogfood on Cortex repo itself + polish | T08, T11 |

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

## V1 scope recap (from PRD §28)

- **Core:** Go binary, `init`, `index`, `update`, Tree-sitter parsing, SQLite, FTS5, symbol indexing, import/reference indexing, incremental updates.
- **Memory:** `project.md`, `architecture.md`, `conventions.md`, module + decision Markdown.
- **Retrieval:** `search`, `symbol`, `refs`, `deps`, `context`.
- **Agent:** Codebase Skill, Markdown stdout.
- **Explicitly out:** embeddings, vector DB, MCP, hosted service.

## Naming note

PRD refers to the binary as `codebase` and the memory dir as `.mycodebase/`. Per project decision the application is named **Cortex**: binary `cortex`, memory dir `.cortex/`. All acceptance criteria below use the Cortex names.
