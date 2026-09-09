# Cortex

**The harness-independent brain skill for a codebase.**

Cortex gives Claude Code, Codex, OpenCode, Command Code, and other agent
harnesses an evidence-backed understanding of a repository — architecture,
symbols, call graphs, conventions, and typed durable claims — so a session can
start with a reliable map instead of guessing through files one by one.

Cortex complements the harness: use it for unfamiliar, architectural, and
cross-module work; keep native grep, file reads, and Git as the fast path for
known local edits and direct history questions.

> Make an AI agent behave as if it already understands the repository
> while minimizing unnecessary context and token usage.

- **Markdown is memory.** Durable knowledge lives in human-readable, editable,
  Git-friendly files under `.cortex/`.
- **SQLite is derived state.** A structural index (Tree-sitter + FTS5) that can
  be deleted and rebuilt at any time.
- **Skill over protocol.** A single local Go binary — no MCP server, no hosted
  service, no vector database, no network dependency.

## How it works

```
Repository
   │
   ├── Tree-sitter parsing ──► symbols, imports, calls, inheritance
   ├── File hashing ─────────► incremental updates (only changed files)
   └── Markdown memory ──────► project knowledge, decisions, conventions
                │
                ▼
        SQLite + FTS5 index (.cortex/index/codebase.db)
                │
                ▼
   cortex context "<task>"  ──►  budgeted, ranked Markdown for the agent
```

Retrieval is layered (PRD §14): **① Markdown memory → ② structural search
(symbols, callers, callees) → ③ lexical FTS5 → ④ semantic search (future) →
⑤ source extraction.** Results are ranked by name match, path match, text
relevance, and call proximity, then trimmed to a token budget.

## Install

Cortex currently supports **native macOS and Linux builds**. Windows support is planned. Requires the Go version declared in `go.mod` and a working C compiler because Tree-sitter uses cgo; SQLite/FTS5 itself is pure Go.

```bash
git clone <this-repo> && cd cortex
make build
make install
eval "$(make env)"     # updates PATH in the current shell
cortex version
```

`make install` defaults to `$HOME/.local/bin` and does not modify shell startup files. Persist the PATH export in `~/.zprofile`/`~/.zshrc` (macOS) or `~/.profile`/`~/.bashrc` (Linux), or choose another destination:

```bash
make install PREFIX=/usr/local
make install BINDIR="$(go env GOPATH)/bin"
make install DESTDIR="$PWD/stage"   # packaging/staging
```

The direct fallback is `go build -o dist/cortex ./cmd/cortex`. Builds are host-native; a simple `GOOS=linux` cross-build from macOS is not supported because it needs a Linux C cross-compiler and sysroot.

## Quickstart

```bash
cd your-project

cortex init       # creates .cortex/ memory scaffold
cortex index      # parses the repo into the structural index

cortex context "How does authentication work?"   # ask anything
```

Fill in `.cortex/project.md` (and friends) with durable knowledge — the more
the memory knows, the less the agent explores. Commit `.cortex/` to Git
(`.cortex/index/` is auto-ignored; it is rebuildable).

## Commands

| Command | Purpose |
|---------|---------|
| `cortex init` | Create the `.cortex/` memory scaffold (idempotent, never overwrites your edits) |
| `cortex index [--full]` | Build the index (default is incremental) |
| `cortex update` | Incrementally index changed/deleted files |
| `cortex search "<q>"` | Search symbols and file contents |
| `cortex symbol "<Name>" [--src]` | Show a symbol: location, signature, docs, calls, callers |
| `cortex refs "<Name>"` | All references to a symbol, grouped by file |
| `cortex deps "<Name>"` | Callees, callers, and file imports of a symbol |
| `cortex history [query]` | Bounded live Git commit and file history |
| `cortex context "<task>"` | **The flagship:** task-specific, budgeted context for an agent |
| `cortex memory` | Print all project memory |
| `cortex memory work` | Show active task work memory |
| `cortex memory preferences` | Show active scoped preferences |
| `cortex memory proposals` | Show reviewable proposed memory |
| `cortex taste <command>` | Import and manage local Taste packages |
| `cortex watch [--interval 2s]` | Keep the index fresh in the background |
| `cortex version` | Print version |

Examples:

```bash
cortex symbol "AuthService.login" --src
cortex refs "TokenStore"
cortex context "add rate limiting to the login endpoint" --budget 4000
```

## Agent skill and session instructions

Install the bundled skill for OpenCode, Codex, oh-my-pi (OMP), pi, and Claude:

```bash
cortex skill install                         # project-local native skills
cortex skill install --agent shared          # interoperable .agents/skills copy
cortex skill install --agent all --global    # user-wide native skills
```

Project destinations are `.opencode/skills`, `.codex/skills`, `.omp/skills`,
`.pi/skills`, and `.claude/skills`; global destinations follow each tool's
native config directory. Existing identical files are skipped; conflicting
files require `--force`.

Generate repository instructions for every new session:

```bash
cortex agents init
```

This creates or updates only the Cortex-managed block in the root `AGENTS.md`,
preserving your existing instructions. The generated block tells agents to
run `cortex context` before broad exploration and keep `.cortex/` memory fresh.

The source skill is [skills/cortex/SKILL.md](./skills/cortex/SKILL.md).

## Memory layout

```
.cortex/
├── config.md          # frontmatter: ignore globs, max-file-size, languages
├── project.md         # purpose, stack, entry points
├── architecture.md    # compact mental map of layers and flows
├── conventions.md     # rules the code must follow
├── decisions/         # one file per architectural decision
│   └── auth.md
├── modules/           # one file per major module
│   └── auth.md
├── work/              # task goals, plans, outcomes, and open questions
├── preferences/       # scoped soft preferences and Taste-compatible records
├── taste/              # local Taste package manifests (derived metadata)
├── feedback/           # explicit local feedback audit events
└── index/
    └── codebase.db    # derived SQLite index (rebuildable)
```

### Memory etiquette

Durable knowledge only — no task chatter:

- ✅ “Authentication must go through AuthService.”
- ❌ “User asked to fix login button today.”

For machine-readable durable claims, use one claim per Markdown file:

```md
---
id: auth-service-boundary
type: constraint
scope: repository
confidence: 0.95
status: active
updated_at: 2026-09-09
source: [src/auth/AuthService.ts]
evidence: [src/auth/AuthService.ts#L10-L22]
---

Authentication must go through AuthService.
```

Supported claim types include `fact`, `decision`, `constraint`, `architecture`,
`module`, `convention`, `behavior`, and `preference`. Confidence expresses
belief strength for soft or inferred knowledge; it does not override explicit
requirements or source authority. Claims can use `supersedes` and `contradicts`
relationships. Proposed, deprecated, and superseded claims remain inspectable
but are excluded from ordinary context retrieval.

Legacy free-form Markdown remains supported. Source code always wins over stale
memory. `cortex memory check` reports invalid claims, missing relationships, and
stale generated source hashes.

## Configuration

Edit `.cortex/config.md` frontmatter:

```md
---
ignore:               # extra glob patterns to skip
  - "**/*_gen.go"
max-file-size: 1048576
languages: []         # empty = all supported
---
```

Supported languages (Tree-sitter): **TypeScript, TSX, JavaScript, JSX, Go,
Python, Rust, Java, Kotlin, C, C++**. Text files (md, json, yaml, …) are
content-indexed; unknown or unparseable files degrade to full-text search —
indexing never fails hard.

## Design principles

1. Markdown is memory — readable, editable, version-controlled.
2. SQLite is derived state — delete it and rebuild.
3. Structure before semantics — symbols and relationships before embeddings.
4. Retrieve before reading — find code before loading it.
5. Symbols before files — return the smallest useful unit.
6. Incremental by default — never re-index unchanged files.
7. Skill over protocol — a local binary, not a network service.
8. Source is truth — generated knowledge never overrides code.
9. Memory evolves — facts can be updated, deprecated, removed.
10. Optimize for task completion — minimum context to correctly finish the task.

## Performance

Targets (PRD §25): startup <100 ms (excluding first index), incremental
indexing touches only changed files, single native binary, no server. All
output is Markdown, sized to a token budget (`--budget`, default ≈6000).

## V2 outlook

Embeddings + hybrid semantic retrieval, smarter reranking, Git history
retrieval, dependency visualization, LSP integration. Deliberately out of V1.

## Development

```bash
go build ./...     # build
go test ./...      # unit + integration tests
go vet ./...
```

Project layout:

```
cmd/cortex/          CLI entry point
internal/lang/       Tree-sitter extractors (per-language)
internal/store/      SQLite + FTS5 persistence layer
internal/index/      Repo walker, full/incremental indexing
internal/memory/     Markdown memory scaffold, config, reader
internal/retrieve/   search / symbol / refs / deps
internal/context/    Context engine (ranking, budget, fallbacks)
internal/lexsearch/  No-index lexical fallback
skills/cortex/       Agent skill
docs/                PRD, kanban, task tracker
```
