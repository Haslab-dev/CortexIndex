# Project

## Purpose

Cortex is a local-first codebase memory engine for AI coding agents. It helps a new session understand a repository with less repeated searching, file reading, and duplicated architectural discovery.

## Stack

- Go 1.27.1
- Tree-sitter language parsers through `github.com/smacker/go-tree-sitter`
- SQLite with FTS5 through `modernc.org/sqlite`
- cgo and a native C compiler for Tree-sitter bindings
- Human-readable Markdown memory under `.cortex/`
- Makefile for native macOS/Linux build and installation

## Major Modules

- `cmd/cortex` — CLI entry point and command dispatch
- `internal/lang` — Tree-sitter extraction for Go, TypeScript, TSX, JavaScript, JSX, Python, Rust, Java, Kotlin, C, and C++
- `internal/store` — SQLite schema, FTS5 indexes, symbols, references, imports, chunks, and transactional file replacement
- `internal/index` — repository walk, ignore rules, hashing, full rebuilds, incremental updates, and deletion sweeps
- `internal/memory` — `.cortex` Markdown scaffold, configuration parsing, and memory rendering
- `internal/retrieve` — `search`, `symbol`, `refs`, and `deps` retrieval
- `internal/context` — task-specific ranking, source extraction, token budgeting, and fallback context
- `internal/lexsearch` — lexical repository fallback when the structural index is unavailable
- `internal/agents` — embedded Skill installation and managed `AGENTS.md` generation

## Entry Points

- `cmd/cortex/main.go` — CLI dispatch
- `cortex init` — initialize `.cortex/` memory
- `cortex index` — build the index
- `cortex update` — incrementally update changed files
- `cortex context "<task>"` — produce agent-oriented context
- `cortex skill install` — install the Cortex Skill for supported agents
- `cortex agents init` — create or update `AGENTS.md`
- `Makefile` — build, test, vet, install, and PATH helpers
- `skills/cortex/SKILL.md` — agent workflow instructions

## Architecture

Cortex combines durable Markdown knowledge with a derived SQLite/FTS5 structural index. Tree-sitter extracts symbols and relationships; the context engine retrieves memory, structural results, lexical matches, and focused source before emitting compact Markdown for an agent.

## External Services

None. Cortex is local-only and has no hosted backend, daemon, network dependency, MCP server, vector database, or embedding service in V1.

## Memory Provenance

- `source`: relevant repository file or package
- `updated_at`: 2026-09-08
- `status`: active
