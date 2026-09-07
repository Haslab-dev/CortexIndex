# T01 — Project scaffold & dependency verification

**Status:** Done
**Depends on:** —
**PRD sections:** §6 Technology Stack, §25 Performance Requirements

## Goal

Standalone Go module `cortex` whose two risky dependencies are proven to download, compile, and work together on this machine before any feature code is written.

## Scope

- `go.mod` (module `cortex`), `.gitignore` for build output.
- Dependencies: `github.com/smacker/go-tree-sitter` (bundled grammars), `modernc.org/sqlite` (pure-Go SQLite with FTS5 enabled by default; no cgo build-tag footguns).
- Smoke program: parse a Go snippet via tree-sitter; create FTS5 virtual table; MATCH query returns a row.

## Acceptance criteria

- [x] `go mod init cortex` succeeds.
- [x] `smacker/go-tree-sitter` resolves; includes grammars: golang, javascript, typescript, python, rust, java, kotlin, c, cpp.
- [x] `modernc.org/sqlite` resolves; FTS5 `CREATE VIRTUAL TABLE ... USING fts5` + `MATCH` works.
- [x] CGO builds tree-sitter binding on darwin/arm64 (smoke run prints parsed root node).
