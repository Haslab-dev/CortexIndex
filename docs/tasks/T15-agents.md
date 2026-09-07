# T15 — `AGENTS.md` generator/updater

**Status:** Done
**Depends on:** T08, T09

## Goal

Generate repository-root instructions so every new coding session knows to use Cortex's persistent memory before broad exploration.

## Acceptance criteria

- [x] `cortex agents init` creates `AGENTS.md` when absent.
- [x] Existing unmanaged content is preserved; Cortex marker block is replaced in place on repeat runs.
- [x] Repeated generation is byte-idempotent.
- [x] `--dir` selects the target project root.
- [x] Generated instructions cover context-first retrieval, durable memory, provenance, `cortex update`, source precedence, and fallback.
