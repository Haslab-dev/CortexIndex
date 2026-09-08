---
name: cortex
description: Persistent codebase memory for AI coding agents. Use at the start of any task in a repository that has a .cortex/ directory (or after `cortex init` + `cortex index`): run `cortex context "<task>"` to instantly load architecture, relevant symbols, calls, and conventions instead of grepping and reading files one by one. Also use `cortex search/symbol/refs/deps` for targeted code questions, and update memory when durable knowledge is discovered.
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

### 1. Start every task with context

```bash
cortex context "<task description>"
```

Example: `cortex context "add rate limiting to the login endpoint"`

This returns (Markdown): relevant memory, ranked symbols with file:line,
their calls, source for the top symbols, relevant files, and conventions.
**Read this before any grep/glob/read.** It usually replaces most of the
exploration you would otherwise do.

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

### 4. Update memory only for durable knowledge

While working, if you discover lasting facts (architecture, conventions,
decisions, gotchas), record them in `.cortex/`:

- `project.md` — purpose, stack, entry points
- `architecture.md` — layer/flow map
- `conventions.md` — rules the code must follow
- `modules/<name>.md` — one file per major module
- `decisions/<topic>.md` — decisions with reason + constraint + source

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

### 5. Keep the index fresh

After you change code, run:

```bash
cortex update
```

It is incremental (only changed files are re-parsed) and fast.

## Rules

1. Source code is truth. If memory contradicts code, trust the code and fix the memory.
2. `context` first, then `symbol`/`refs`/`deps`, then read files — in that order.
3. Never re-explore what memory already answers.
4. The index is derived state: `cortex index` rebuilds it if it seems wrong.
5. If `cortex` is missing or has no index, fall back to normal exploration and
   suggest running `cortex init && cortex index`.
