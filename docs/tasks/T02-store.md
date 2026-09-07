# T02 — SQLite store layer

**Status:** Done
**Depends on:** T01
**PRD sections:** §12 Structural Code Index, §13 SQLite Schema, §20 Incremental Indexing

## Goal

All persistence: derived index in `.cortex/index/codebase.db` (SQLite + FTS5). The database is rebuildable state; nothing lives only here (PRD §31.2).

## Scope

- Tables: `files`, `symbols`, `refs`, `imports`, `chunks`, `memory_metadata`, `meta`.
- FTS5: `symbols_fts(name, parts, kind, file_path, doc)` (rowid = symbol id; `parts` = camelCase/snake-split name for recall), `files_fts(path, content)` (rowid = files.id).
- Locations stored per PRD §12: file, start_line, start_column, end_line, end_column (lines 1-based, columns 0-based).
- `ReplaceFile(file, content, extracted)` — transactional: delete prior rows for path (symbols, refs, imports, chunks, both FTS tables), insert new. Refs get `from_symbol_id` resolved by innermost containing symbol.
- `RemoveFile(path)` — clean removal for deleted files.
- Query API: symbol by name (exact/prefix/FTS), file FTS with `snippet()`, refs by name / from-symbol, imports by file, chunk by symbol, counts, per-file hashes for incremental compare.

## Acceptance criteria

- [x] Schema auto-created on open; safe to call repeatedly.
- [x] `ReplaceFile` twice for same path leaves zero duplicate rows (all tables + FTS).
- [x] `RemoveFile` leaves zero orphan rows.
- [x] Symbol lookup by exact name and FTS (incl. camelCase part, e.g. query `serv` finds `Server`) works.
- [x] File FTS returns snippet highlights.
- [x] WAL mode on; bulk insert inside one transaction for a file.
