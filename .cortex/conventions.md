# Conventions

## Agent Workflow

- Run `cortex context "<task>"` before broad repository exploration.
- Prefer `cortex symbol`, `cortex refs`, `cortex deps`, and `cortex search` before reading whole files.
- Read the smallest useful source unit: a symbol plus required surrounding context.
- Run `cortex update` after code changes so the next session sees current structure.
- If Cortex is unavailable, continue normally and restore the index later.

## Memory

- Store durable project knowledge only; do not store temporary task chatter.
- Keep Markdown memory readable, editable, Git-friendly, and portable.
- Include provenance when recording facts: `source`, `updated_at`, and `status`.
- Source code is authoritative over stale Markdown memory.
- SQLite under `.cortex/index/` is derived state and may be rebuilt.

## Go and Repository Structure

- Keep CLI dispatch in `cmd/cortex` and reusable behavior in `internal/` packages.
- Match the existing package boundaries: language extraction, store, indexing, memory, retrieval, context, lexical fallback, and agent integration.
- Use standard library APIs where practical; Cortex has no runtime server or network dependency.
- Tree-sitter requires cgo and a native C compiler; SQLite/FTS5 uses the pure-Go driver.

## Testing and Quality

- Add tests next to the package they cover.
- Use `t.TempDir()` for filesystem and SQLite fixtures.
- Run `go test ./...`, `go vet ./...`, and `make check` before completing work.
- Validate CLI behavior with real temporary repositories when adding commands.
- Preserve graceful fallbacks: parse failures and missing indexes must not prevent an agent from working.

## Delivery

- Create one Git commit for every implemented feature or task.
- Use commit messages in the form `<type>(TNN): <summary>`.
- Keep `docs/kanban.md` and the relevant task file synchronized with implementation evidence.
- Do not commit `.cortex/index/codebase.db` or build output.
- Native Makefile builds target macOS and Linux; Windows support is future work.

## Retrieval Output

- Normal agent-facing output is Markdown on stdout.
- Errors go to stderr.
- Prefer locations, signatures, relationships, and focused source over entire files.
- Keep context within the requested token budget and trim low-value material first.
