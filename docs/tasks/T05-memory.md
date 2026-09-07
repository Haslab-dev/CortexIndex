# T05 — Markdown memory layer

**Status:** Backlog
**Depends on:** T01
**PRD sections:** §7 Project Structure, §8–§11, §18 Memory Update, §19 Memory Lifecycle, §24 Human Control

## Goal

Human-readable durable knowledge in `.cortex/` — editable, Git-friendly, never opaque (PRD §31.1).

## Scope

- `init` scaffold (idempotent, never overwrites user edits):
  `config.md`, `project.md`, `architecture.md`, `conventions.md`, `decisions/`, `modules/`, `index/.gitignore` (derived DB ignored).
- Templates carry guidance: durable knowledge vs temporary task context (§18), provenance + status lifecycle (§19).
- `config.md` minimal frontmatter: extra `ignore:` globs, `max-file-size`, `languages`.
- `memory` command: renders the memory tree + file contents for inspection.
- Memory reader for the context engine (scored by keyword relevance).

## Acceptance criteria

- [ ] `cortex init` creates the full scaffold; second run changes nothing user-facing.
- [ ] Edits to `project.md` survive `cortex init` / `cortex update`.
- [ ] `cortex memory` prints current memory contents to stdout.
- [ ] `config.md` `ignore:` entries actually exclude paths from indexing (T04 honors them).
