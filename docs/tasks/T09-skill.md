# T09 — Agent Skill

**Status:** Backlog
**Depends on:** T08
**PRD sections:** §17 Skill Integration, §28 Agent

## Goal

`skills/cortex/SKILL.md` — teaches any coding agent to use Cortex without MCP (PRD §31.7 "Skill over protocol").

## Scope

Skill instructs the agent to:
1. On new session / unfamiliar task: run `cortex context "<task>"` **before** any repo exploration.
2. Prefer `symbol`/`refs`/`deps` over grep-and-read-everything (structural before broad search, §31.3/§31.4).
3. Read source only for the specific symbols needed.
4. Update durable memory only for lasting knowledge (good vs bad memory examples, §18) with provenance/status (§19).
5. Run `cortex update` after finishing changes; keep memory source-of-truth rules (§31.8).
6. Trust source over stale memory (§27).

## Acceptance criteria

- [ ] SKILL.md has proper frontmatter (name/description) and correct binary usage.
- [ ] Includes copy-pasteable workflow + memory update etiquette.
- [ ] Installed-into-repo path documented (how an agent/user adopts it).
