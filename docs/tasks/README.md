# Task tracker — how it works

- **[kanban.md](../kanban.md)** is the board: Backlog → In Progress → Review/QA → Done. Move cards there.
- Each task file in this folder holds the detailed contract: goal, scope, and acceptance criteria that gate "Done".
- Rules:
  1. Check an acceptance box only with real evidence (command output, test run, generated file).
  2. Keep card ↔ task-file `Status` in sync.
  3. New discoveries become new task files (`TNN-slug.md`) and board cards, not silent scope creep.
  4. **One git commit per implemented feature/task** (see commit policy below).
- Task ID numbering: `T01`–`T12` cover the original V1 scope; follow-up delivery work uses `T13+`.

| ID | Task | PRD anchor |
|----|------|-----------|
| [T01](./T01-scaffold.md) | Scaffold & dependency verification | §6 |
| [T02](./T02-store.md) | SQLite store layer | §12, §13 |
| [T03](./T03-languages.md) | Tree-sitter extractors (11 languages) | §12, §26 |
| [T04](./T04-indexer.md) | Indexer (full + incremental) | §20 |
| [T05](./T05-memory.md) | Markdown memory layer | §7–§11, §18, §19, §24 |
| [T06](./T06-retrieval.md) | Retrieval commands | §14, §16, §21 |
| [T07](./T07-context.md) | Context engine | §14, §15, §21–§23, §27 |
| [T08](./T08-cli.md) | CLI wiring | §16, §25 |
| [T09](./T09-skill.md) | Agent Skill | §17, §28 |
| [T10](./T10-readme.md) | README & docs | — |
| [T11](./T11-tests.md) | Tests | §28 |
| [T12](./T12-dogfood.md) | Dogfooding & polish | §30, §31 |
| [T13](./T13-build-install.md) | Makefile, native build/install/PATH | Delivery follow-up |
| [T14](./T14-skill-installer.md) | Agent Skill installer | Delivery follow-up |
| [T15](./T15-agents.md) | `AGENTS.md` generator/updater | Delivery follow-up |
| [T16](./T16-overview.md) | Intent-aware overview and ranking | Audit follow-up |
| [T17](./T17-bootstrap.md) | Deterministic bootstrap and validation | Audit follow-up |
| [T18](./T18-resolver.md) | Binary discovery and agent startup | Audit follow-up |
| [T19](./T19-benchmark.md) | Local deterministic benchmark | Audit follow-up |

## Commit policy

Every implemented feature/task must land as its own git commit — history should map 1:1 onto the kanban.

- Commit a task when it is functionally complete (acceptance criteria met or a coherent, buildable sub-step).
- Message format: `<type>(TNN): <summary>` — e.g.
  - `feat(T02): sqlite store layer with fts5 and per-file replace`
  - `test(T03): language extraction fixtures`
  - `docs(T09): agent skill for coding agents`
- `<type>` is one of `feat`, `fix`, `test`, `docs`, `chore`, `refactor`.
- Never bundle unrelated tasks into one commit; follow-up fixes reference the same task ID (`fix(T04): ...`).
- Update the task file's acceptance boxes and kanban card in the same commit when a task completes.
