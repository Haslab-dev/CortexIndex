# T14 — Agent Skill installer

**Status:** Backlog
**Depends on:** T08, T09

## Goal

Install Cortex's embedded `SKILL.md` into the native project or global skill directories for OpenCode, Codex, oh-my-pi (OMP), pi, Claude, or the shared Agent Skills path.

## Acceptance criteria

- [ ] Supports `opencode`, `codex`, `omp`, `pi`, `claude`, and `shared` destination mappings in project and global scopes.
- [ ] `cortex skill install` is idempotent; identical files are skipped.
- [ ] Conflicting files are not overwritten without `--force`; writes create parents and use safe file permissions.
- [ ] Skill source is embedded so a binary installed on PATH does not depend on the source checkout.
- [ ] CLI output lists installed/skipped/conflict paths as Markdown.
