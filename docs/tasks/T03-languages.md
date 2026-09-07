# T03 — Tree-sitter language extractors

**Status:** Done
**Depends on:** T01
**PRD sections:** §12 Structural Code Index, §26 Supported Languages

## Goal

For every supported language, extract symbols, call references, inheritance, and imports from the tree-sitter AST into the common `lang.Result` shape.

## Scope

Languages (PRD §26): TypeScript, TSX, JavaScript, JSX, Go, Python, Rust, Java, Kotlin, C, C++.

Per language:
- **Symbols:** functions, methods, classes, interfaces, structs, enums, traits, impls, type aliases, constants, variables, properties — with name, kind, signature (first line), doc comment, parent (class/impl/receiver), precise range.
- **Refs:** call targets (`name` + `qualifier` e.g. receiver for `a.b()`), `extends`/`implements` edges.
- **Imports:** module path + imported names (incl. `require()` for JS).
- Content-only detection for non-source files (md/json/yaml/toml/sql/text) → FTS-only, no AST.

Grammar node kinds/fields must be verified empirically (tree dump per language) rather than assumed.

## Acceptance criteria

- [x] Fixture file per language parses; extractor yields expected symbols/imports/calls (table-driven tests: go, ts, python, rust, java, kotlin, c, cpp).
- [x] TSX/JSX handled via tsx/javascript grammars; `.h`→C, `.hpp/.cc/.cxx`→C++.
- [x] Unparseable/unknown files never crash extraction (walker tolerates ERROR nodes; indexer treats unknown exts as FTS-only).
- [x] Method/impl/namespace parentage recorded (Go receiver, Rust `impl Type`, Java/Kotlin method-in-class, C++ out-of-line `Server::login`).
