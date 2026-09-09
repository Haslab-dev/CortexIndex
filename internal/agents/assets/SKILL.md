---
name: cortex
description: Harness-independent codebase brain for AI coding agents. Use Cortex for unfamiliar, architectural, cross-module, impact, convention, or planning-heavy work in repositories with a .cortex/ directory; use native grep/glob/source reads for known local edits and native Git for direct history questions. Cortex provides current code structure, durable project memory, and bounded evidence-backed context.
---

# Cortex — Codebase Memory

Use this bounded resolver at the start of a fresh shell so the workflow also works when `cortex` is not on PATH:

```bash
cortex_cmd() {
  _cortex_bin="${CORTEX_BIN:-}"
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then _cortex_bin="$(command -v cortex 2>/dev/null || true)"; fi
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then
    for _cortex_root in "$PWD" "$(dirname "$PWD")" "$(dirname "$(dirname "$PWD")")"; do
      if [ -x "$_cortex_root/dist/cortex" ]; then _cortex_bin="$_cortex_root/dist/cortex"; break; fi
      if [ -x "$_cortex_root/cortex" ]; then _cortex_bin="$_cortex_root/cortex"; break; fi
    done
  fi
  if [ -z "$_cortex_bin" ] && [ -x "${HOME:-}/.local/bin/cortex" ]; then _cortex_bin="${HOME}/.local/bin/cortex"; fi
  if [ -z "$_cortex_bin" ] || [ ! -x "$_cortex_bin" ]; then printf '%s\n' 'Cortex unavailable; continue normally and run make install later.' >&2; return 127; fi
  "$_cortex_bin" "$@"
}
```

Use `cortex_cmd` below instead of assuming the binary is on PATH.

You have persistent memory for this repository via the Cortex binary.
It gives you architecture knowledge, symbols, call graphs, and conventions
**without** reading files one by one.

## Workflow

### 1. Route the task before retrieving context

Cortex complements native tools; it does not replace them.

- Use `cortex overview` for repository orientation or a project tour.
- Use `cortex context "<task>"` for unfamiliar, cross-module, architectural,
  impact-analysis, convention, or planning-heavy work.
- Use `cortex symbol`, `refs`, `deps`, and `search` for structural questions.
- Use native `grep`, `glob`, direct source reads, or an editor for a known local
  edit or exact file inspection.
- Use `cortex history` for bounded commit/message/file-history questions.
- Use native `git show`, `diff`, and `blame` for full or advanced history; Cortex history is live Git evidence, not a SQLite cache.
- Use `cortex taste lint/import/list/show` to review local Taste packages.
- Imported Taste preferences are proposed by default; use explicit `cortex taste enable` only after review.
- Confidence is belief strength, not authority. Taste never overrides source code or explicit task requirements.
- Harnesses may emit explicit feedback with `cortex taste feedback`; Cortex does not observe UI actions automatically.

Example: `cortex context "add rate limiting to the login endpoint"`

Context returns bounded Markdown with relevant memory, ranked symbols,
relationships, focused source, relevant files, and conventions. Read exact
source before editing. For simple known-symbol work, skip the Cortex preamble.

### 2. Ask targeted questions with structure, not search-and-read

| Need | Command |
|------|---------|
| Where is X and what does it do | `cortex symbol "<Name>"` |
| Full source of a symbol | `cortex symbol "<Name>" --src` |
| Everything that calls X | `cortex refs "<Name>"` |
| What X depends on / depends on X | `cortex deps "<Name>"` |
| Fuzzy search code and text | `cortex search "<terms>"` |

Prefer these over ripgrep/grep. They use a prebuilt index: results include
locations, enclosing symbols, and call relationships that raw text search
cannot give you.

### 3. Read source only for the specific symbols you need

The context output already includes source for top symbols. Use
`cortex symbol "<Name>" --src` for the rest. Avoid reading whole files
unless editing them directly.

### 4. Record work at task boundaries

For a meaningful task, keep goals, plans, decisions, outcomes, and open questions in `.cortex/work/` using `cortex: work` records. Do not turn temporary chat chatter into repository facts. Record an outcome after tests and code changes are complete.

### 5. Update memory only for durable knowledge

While working, if you discover lasting facts (architecture, conventions,
decisions, gotchas), record them in `.cortex/`:

- `project.md` — purpose, stack, entry points
- `architecture.md` — layer/flow map
- `conventions.md` — rules the code must follow
- `modules/<name>.md` — one file per major module
- `decisions/<topic>.md` — decisions with reason + constraint + source
- `work/<task>.md` — goals, plans, decisions, outcomes, and open questions
- `preferences/<topic>.md` — scoped soft preferences with confidence and provider

Good memory:

```
Authentication must go through AuthService; handlers must not call
APIClient directly.
```

Bad memory (task chatter — do NOT store):

```
User asked to fix the login button today.
```

Include provenance when possible:

```md
source: src/auth/AuthService.ts
updated_at: 2026-09-07
status: active
```

### 6. Keep the index fresh

After you change code, run:

```bash
cortex update
```

It is incremental (only changed files are re-parsed) and fast.

## Rules

1. Source code is truth. If memory contradicts code, trust the code and fix the memory.
2. Route by task uncertainty: Cortex for broad/cross-module understanding,
   native tools for known local work, and Git for direct history questions.
3. Never re-explore what reliable memory already answers, but verify exact source before editing.
4. Typed claims expose status, confidence, and evidence; proposed or inferred claims are not project constraints.
5. The index is derived state: `cortex index` rebuilds it if it seems wrong.
6. If `cortex` is missing or has no index, fall back to normal exploration and
   suggest running `cortex init && cortex index`.
