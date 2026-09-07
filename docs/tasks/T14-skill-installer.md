# T14 — Agent Skill installer

**Status:** Done
**Depends on:** T08, T09

## Goal

Install Cortex's embedded `SKILL.md` into the native project or global skill directories for OpenCode, Codex, oh-my-pi (OMP), pi, Claude, or the shared Agent Skills path.

## Acceptance criteria

- [x] Supports `opencode`, `codex`, `omp`, `pi`, `claude`, and `shared` destination mappings in project and global scopes (destination matrix tests).
- [x] `cortex skill install` is idempotent; identical files are skipped.
- [x] Conflicting files are not overwritten without `--force`; writes create parents and use safe file permissions.
- [x] Skill source is embedded so a binary installed on PATH does not depend on the source checkout.
- [x] CLI output lists installed/skipped/conflict paths as Markdown (CLI smoke test).
