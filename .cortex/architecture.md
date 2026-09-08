# Architecture

Keep this as a compact mental map, not a source-code dump.

## Layers

```text
CLI (cmd/cortex)
→ Markdown memory (.cortex/*.md)
→ Indexer (walk, hash, parse changed files)
→ Tree-sitter extractors (internal/lang)
→ SQLite + FTS5 store (internal/store)
→ Retrieval (internal/retrieve)
→ Context engine (internal/context)
→ Markdown output for the coding agent
```

`internal/lexsearch` is the failure-tolerant lexical path used when the index is missing or incomplete. SQLite is derived state and can always be rebuilt.

## Indexing Flow

```text
cortex index
→ walk repository with ignore rules
→ hash each eligible file
→ parse supported source files
→ extract symbols, imports, calls, and inheritance
→ replace that file's rows transactionally
→ update FTS5 content
```

```text
cortex update
→ compare current hashes with files table
→ skip unchanged files
→ re-parse and replace changed/new files
→ remove rows for deleted files
```

Parsing failures never block the agent: the file remains available through full-text search.

## Context Flow

```text
cortex context "<task>"
→ extract task keywords and identifier parts
→ retrieve relevant Markdown memory
→ search exact, prefix, and FTS symbols
→ rank by name/text matches and call proximity
→ expand one hop into calls/callers/imports
→ retrieve focused source for top symbols
→ trim lowest-value sections to the token budget
→ print Markdown context
```

If no index is available, the context command falls back to a lexical repository scan.

## CLI and Agent Integration

The CLI is a single native binary. `skills/cortex/SKILL.md` instructs agents to query Cortex before broad exploration. `cortex skill install` copies the embedded Skill into native directories for OpenCode, Codex, OMP, pi, Claude, or the shared `.agents/skills` location. `cortex agents init` writes the managed Cortex block in the repository-root `AGENTS.md`.

## Key Flows

### New agent session

```text
AGENTS.md / Cortex Skill
→ cortex context "<task>"
→ cortex symbol / refs / deps
→ focused source read
→ code change
→ tests
→ cortex update
```

### Source of truth

```text
Source code > stale Markdown memory > derived SQLite state
```

When memory conflicts with source, trust and update the source-derived understanding.

## Derived State

`.cortex/index/codebase.db` contains files, symbols, references, imports, source chunks, metadata, and FTS5 tables. It is ignored by Git and safe to delete and rebuild with `cortex index --full`.
