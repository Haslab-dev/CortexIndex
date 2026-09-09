# T20 — Codebase Brain Foundation

## Status

In progress — first slice.

## Goal

Make Cortex a harness-independent brain skill for a codebase. Cortex should
provide evidence-backed understanding of the current code and durable project
knowledge while complementing, rather than replacing, the harness, native file
search, and Git.

## Responsibility boundaries

- **Agent harness** (Claude Code, Codex, OpenCode, Command Code, or another
  runtime): session state, context assembly, tools, permissions, retries,
  compaction, and feedback collection.
- **Cortex**: current code structure, repository memory, evidence, and bounded
  task context.
- **Git/native tools**: direct change history, diffs, blame, editing, and
  verification until Cortex history support is implemented.
- **Preference providers** (including Taste-compatible packages): soft
  user/team judgment with confidence values; preferences never override source
  truth or explicit task constraints.

## Typed claim contract

A claim is one Markdown file with a small frontmatter block:

```yaml
id: auth-service-boundary
type: constraint
scope: repository
confidence: 0.95
status: active
updated_at: 2026-09-09
source: [internal/auth/service.go]
evidence: [internal/auth/service.go#L10-L22]
```

The body is the human-readable statement. Supported types are `fact`,
`decision`, `constraint`, `architecture`, `module`, `convention`, `behavior`,
and `preference`. Supported scopes are `repository`, `team`, `user`, and
`task`. Status values are `active`, `deprecated`, `superseded`, and `proposed`.
Claims may link to other claims with `supersedes` and `contradicts`.

Confidence is belief strength for soft or inferred knowledge. It is not an
authority score: an explicit user constraint and current source code outrank a
high-confidence generic preference. Proposed, deprecated, and superseded claims
remain inspectable but are excluded from ordinary context retrieval.

Legacy free-form Markdown remains valid untracked memory. Generated bootstrap
memory continues to use its existing source-hash validation contract.

## Routing policy

Use Cortex for unfamiliar repositories, architecture and flow questions,
cross-module implementation, impact analysis, conventions, and planning. Use
native grep/glob/source reads for known local edits and exact file inspection.
Use native `git log`, `show`, `diff`, and `blame` for direct history questions;
Git history is not yet indexed by Cortex.

## Implemented in this slice

- Proposed claims are excluded from ordinary `context` and `overview` output.
- `cortex history` provides bounded live Git log/show retrieval with optional
  file filters and capped diffs.
- `.cortex/work/` records goals, plans, decisions, outcomes, and open questions.
- `.cortex/preferences/` records scoped confidence-scored preferences with a
  provider field; preferences are soft guidance and proposed records remain
  review-only.
- Local Taste packages support manifest validation, deterministic digests,
  proposal-first import, explicit enable/disable, confidence updates, local
  push/export, and explicit feedback audit events.
- `cortex memory work`, `preferences`, and `proposals` expose these families.

## Next slices

1. Add a normalized Taste package importer and explicit preference conflict
   resolver.
2. Integrate history selectively into context for explicit why/change prompts.
3. Expand the benchmark into controlled orientation, history, freshness,
   convention, and preference cases with evidence-grounded scoring.
