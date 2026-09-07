# T15 — `AGENTS.md` generator/updater

**Status:** Backlog
**Depends on:** T08, T09

## Goal

Generate repository-root instructions so every new coding session knows to use Cortex's persistent memory before broad exploration.

## Acceptance criteria

- [ ] `cortex agents init` creates `AGENTS.md` when absent.
- [ ] Existing unmanaged content is preserved; Cortex marker block is replaced in place on repeat runs.
- [ ] Repeated generation is byte-idempotent.
- [ ] `--dir` selects the target project root.
- [ ] Generated instructions cover context-first retrieval, durable memory, provenance, `cortex update`, source precedence, and fallback.
