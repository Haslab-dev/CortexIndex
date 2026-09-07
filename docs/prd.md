# Codebase Memory Engine

## 1. Overview

Codebase Memory Engine is a lightweight local-first developer tool that gives AI coding agents persistent understanding of a software repository.

Instead of forcing an AI agent to repeatedly search and read an entire codebase, the system maintains:

1. **Human-readable Markdown memory** for durable project knowledge.
2. **A structural code index** generated from the source code.
3. **Incremental updates** when files change.
4. **A context retrieval engine** that returns only the information relevant to the current task.
5. **A Skill-based interface** that allows coding agents to use the system without requiring MCP.

The primary goal is not to build a generic code graph. The goal is:

> Make an AI agent behave as if it already understands the repository while minimizing unnecessary context and token usage.

---

# 2. Problem

AI coding agents repeatedly rediscover the same information.

For a new task, an agent may:

```text
search repository
→ inspect files
→ inspect imports
→ read implementations
→ search again
→ build mental model
→ start coding
```

When another task starts, much of this process happens again.

This causes:

* unnecessary input tokens
* repeated tool calls
* increased latency
* inconsistent understanding
* difficulty maintaining architectural knowledge
* duplicated repository exploration across sessions

Existing approaches such as pure vector RAG or graph-based code representations do not completely solve this problem.

The system should instead combine:

```text
Persistent knowledge
+
Deterministic code structure
+
Targeted retrieval
+
Incremental indexing
```

---

# 3. Goals

## Primary Goals

### G1 — Persistent Codebase Understanding

The agent should retain useful knowledge about the project between sessions.

Examples:

* architecture
* module responsibilities
* conventions
* important decisions
* integration patterns
* domain concepts

### G2 — Targeted Code Retrieval

The agent should find relevant code without searching the entire repository manually.

### G3 — Minimal Context

Only relevant symbols, files, relationships, and memory should be returned to the agent.

### G4 — Incremental Updates

Changing one or several files should not require re-indexing the entire repository.

### G5 — Markdown-First Memory

Durable knowledge must remain human-readable, editable, version-controllable, and portable.

### G6 — Agent Skill Integration

The system should be usable through a Skill rather than requiring MCP.

### G7 — Lightweight Runtime

The system should run locally with a small memory footprint and no external server dependency.

---

# 4. Non-Goals

The initial version will NOT attempt to:

* become a full IDE
* replace Git
* replace ripgrep
* replace LSP
* become a general-purpose graph database
* automatically understand every architectural decision
* maintain arbitrary conversational memories
* require a hosted backend
* require a vector database
* expose MCP as the primary interface

---

# 5. Core Architecture

```text
                         Repository
                             │
                             ▼
                     ┌───────────────┐
                     │ Go Codebase   │
                     │ Binary        │
                     └───────┬───────┘
                             │
             ┌───────────────┼────────────────┐
             ▼               ▼                ▼
        Tree-sitter       Git/File        Markdown
          Parser           Changes          Memory
             │               │                │
             ▼               ▼                ▼
         Symbols         Incremental       Project
         Imports          Updates           Knowledge
         References       History           Decisions
             │               │                │
             └───────────────┼────────────────┘
                             ▼
                       Local Index
                         SQLite
                             │
                 ┌───────────┼───────────┐
                 ▼           ▼           ▼
               FTS5      Structural    Semantic
                         Retrieval     Retrieval
                 │           │           │
                 └───────────┼───────────┘
                             ▼
                       Context Engine
                             │
                             ▼
                       Markdown Output
                             │
                             ▼
                           Agent
                             │
                      Codebase Skill
```

---

# 6. Technology Stack

## Core

* **Go**
* **Tree-sitter**
* **SQLite**
* **SQLite FTS5**
* Git CLI / Git metadata
* Filesystem watcher

## Optional Later

* Local embedding model
* Vector index
* Semantic reranker

Semantic search should NOT be a hard dependency for V1.

Structural and lexical retrieval should be sufficient for the initial system.

---

# 7. Project Structure

The tool creates a hidden project directory:

```text
.mycodebase/
├── config.md
├── project.md
├── architecture.md
├── conventions.md
├── decisions/
│   ├── auth.md
│   ├── database.md
│   └── api.md
├── modules/
│   ├── auth.md
│   ├── users.md
│   └── payments.md
└── index/
    └── codebase.db
```

## Principle

Markdown contains **durable knowledge**.

SQLite contains **derived knowledge**.

---

# 8. Markdown Memory

## 8.1 Project Memory

`project.md`

Contains:

* project purpose
* technology stack
* major modules
* entry points
* important external services

Example:

```md
# Project

## Purpose

Internal payment management platform.

## Stack

- TypeScript
- React
- Node.js
- PostgreSQL

## Architecture

Feature-based frontend with service-oriented backend.
```

---

# 9. Architecture Memory

`architecture.md`

Contains high-level architectural relationships.

Example:

```md
# Architecture

Frontend
→ API
→ Application Services
→ Repository
→ PostgreSQL

Authentication:

LoginPage
→ AuthService
→ APIClient
→ TokenStore
```

This should remain relatively small.

It is a **mental map**, not a source-code dump.

---

# 10. Module Memory

Each major module may have its own Markdown file.

Example:

```text
modules/
├── auth.md
├── users.md
├── billing.md
└── notifications.md
```

A module document can contain:

* responsibility
* important symbols
* dependencies
* conventions
* known constraints
* design decisions

---

# 11. Decision Memory

Architectural decisions should be stored separately.

Example:

```text
decisions/
├── auth.md
├── database.md
└── state-management.md
```

Example:

```md
# Authentication Decision

## Decision

Use JWT with refresh tokens.

## Reason

The API is consumed by web and mobile clients.

## Constraint

Token management must remain inside AuthService.

## Source

src/auth/AuthService.ts
```

---

# 12. Structural Code Index

The Go binary parses source code using Tree-sitter.

The index should capture:

### Files

* path
* language
* size
* hash
* modification time

### Symbols

* functions
* methods
* classes
* interfaces
* structs
* types
* constants
* variables

### Relationships

* imports
* exports
* calls
* references
* implementations
* inheritance
* dependencies

### Source locations

Every indexed symbol should retain:

```text
file
start_line
start_column
end_line
end_column
```

This allows the context engine to retrieve precise source ranges instead of entire files.

---

# 13. SQLite Schema

Conceptually:

```text
files
symbols
references
imports
exports
chunks
memory_metadata
```

FTS5 indexes searchable textual information.

The database is derived and can always be rebuilt.

---

# 14. Retrieval Engine

Retrieval should happen in layers.

## Layer 1 — Markdown Memory

Check durable project knowledge first.

```text
project.md
architecture.md
modules/*
decisions/*
```

## Layer 2 — Structural Search

Search:

* symbol names
* references
* callers
* callees
* imports
* dependencies

## Layer 3 — Lexical Search

Use FTS5 for natural-language and code-text searches.

## Layer 4 — Semantic Search

Optional future layer.

Use embeddings when lexical/structural retrieval is insufficient.

## Layer 5 — Source Retrieval

Only retrieve actual implementation source after identifying relevant symbols.

---

# 15. Context Command

The most important interface is:

```bash
codebase context "<task>"
```

Example:

```bash
codebase context "How does authentication work?"
```

The engine should:

```text
task
 ↓
memory retrieval
 ↓
symbol search
 ↓
relationship expansion
 ↓
relevance ranking
 ↓
source extraction
 ↓
context compression
 ↓
Markdown response
```

Output should be optimized for LLM consumption.

Example:

```md
# Codebase Context

## Architecture

Authentication uses JWT.

Login flow:

LoginPage
→ AuthService.login()
→ APIClient
→ TokenStore

## Relevant Symbols

### AuthService.login()

src/auth/AuthService.ts:42

Calls:
- APIClient.post()
- TokenStore.save()

### TokenStore

src/auth/TokenStore.ts

Responsible for token persistence.

## Relevant Files

- src/auth/AuthService.ts
- src/auth/TokenStore.ts
- src/auth/LoginPage.tsx

## Project Conventions

- API calls belong inside services.
- Components should not call APIClient directly.
```

---

# 16. CLI

Initial commands:

```bash
codebase init
```

Initialize project memory and index.

```bash
codebase index
```

Build or rebuild the index.

```bash
codebase update
```

Incrementally update changed files.

```bash
codebase search "<query>"
```

Search codebase knowledge.

```bash
codebase symbol "<symbol>"
```

Retrieve a symbol.

```bash
codebase refs "<symbol>"
```

Find references.

```bash
codebase deps "<symbol>"
```

Find dependencies.

```bash
codebase context "<task>"
```

Build minimal task-specific context.

```bash
codebase memory
```

Inspect project memory.

---

# 17. Skill Integration

The system provides a Skill:

```text
skills/
└── codebase/
    └── SKILL.md
```

The Skill instructs the agent to use the Go binary.

Conceptually:

```text
Agent
 ↓
Codebase Skill
 ↓
codebase context
 ↓
relevant context
 ↓
Agent reasoning
```

The Skill should encourage:

1. Querying existing memory first.
2. Using structural retrieval before broad search.
3. Retrieving symbols before entire files.
4. Avoiding unnecessary repository-wide exploration.
5. Updating durable memory when important knowledge is discovered.

---

# 18. Memory Update

Memory should not be updated after every agent message.

The agent should update memory only when it discovers durable knowledge.

Examples:

### Good memory

```text
Authentication must go through AuthService.
```

### Bad memory

```text
User asked to fix login button today.
```

The first is persistent project knowledge.

The second is temporary task context.

---

# 19. Memory Lifecycle

Memory entries should support:

```text
created
updated
deprecated
deleted
```

Each durable fact should ideally contain provenance:

```text
source:
  src/auth/AuthService.ts

updated_at:
  2026-09-07

status:
  active
```

The system should favor source code over stale Markdown when conflicts occur.

---

# 20. Incremental Indexing

A filesystem watcher may detect changes:

```text
file changed
     ↓
hash comparison
     ↓
parse changed file
     ↓
replace symbols
     ↓
update references
     ↓
update FTS index
```

Git changes can also be used:

```bash
git diff
git status
```

The system must avoid unnecessary full-repository indexing.

---

# 21. Context Ranking

Every retrieved item receives a relevance score based on signals such as:

```text
symbol name match
file path match
memory relevance
reference proximity
dependency proximity
text relevance
recent modification
task keywords
```

Example:

```text
AuthService.login()        0.97
TokenStore.save()          0.91
LoginPage                  0.82
UserRepository             0.54
PaymentService             0.08
```

Only the highest-value context should normally be returned.

---

# 22. Token Optimization

Token reduction is a core metric.

The system should optimize:

```text
less source
less duplication
less repeated exploration
less irrelevant context
```

It should NOT optimize merely for:

```text
number of graph nodes returned
```

Primary measurement:

```text
tokens required to complete task
```

Secondary measurements:

* number of tool calls
* latency
* retrieval precision
* retrieval recall
* task success rate

---

# 23. Source Context Strategy

Never default to:

```text
entire file
```

Prefer:

```text
symbol
+
required surrounding lines
+
related symbols
```

For example:

```text
AuthService.login()
+ constructor
+ imported dependencies
+ directly called methods
```

Expand context only when necessary.

---

# 24. Human Control

All Markdown memory must be:

* readable
* editable
* Git-friendly
* portable
* inspectable

A developer should be able to open:

```text
.mycodebase/architecture.md
```

and immediately understand what the AI knows about the project.

No opaque proprietary memory format should be required.

---

# 25. Performance Requirements

Target for a typical developer repository:

### Runtime

* Single native Go binary.
* No external server.
* No Python runtime.
* No network dependency.

### Indexing

Incremental indexing should process only changed files.

### Memory

Target:

```text
Idle:          <30 MB
Normal query:  <100 MB
Indexing:      <300 MB
```

These are engineering targets, not guaranteed benchmarks.

### Startup

Target startup time:

```text
<100 ms
```

excluding initial index creation.

---

# 26. Supported Languages

V1 should prioritize languages supported reliably by Tree-sitter.

Initial target:

```text
TypeScript
JavaScript
TSX
JSX
Go
Python
Rust
Java
Kotlin
C
C++
```

Additional languages can be added without changing the architecture.

---

# 27. Failure Handling

If the index is unavailable:

```text
codebase context
```

should gracefully fall back to lexical repository search.

If parsing fails:

```text
source file
 ↓
FTS indexing
```

should still work.

If Markdown memory is stale:

```text
source-derived information
```

should take precedence.

The system must never prevent an agent from working because the index is incomplete.

---

# 28. V1 Scope

V1 should implement only:

### Core

* Go binary
* `codebase init`
* `codebase index`
* `codebase update`
* Tree-sitter parsing
* SQLite
* FTS5
* symbol indexing
* import/reference indexing
* incremental updates

### Memory

* `project.md`
* `architecture.md`
* `conventions.md`
* module Markdown
* decision Markdown

### Retrieval

* `search`
* `symbol`
* `refs`
* `deps`
* `context`

### Agent

* Codebase Skill
* Markdown stdout output

No embeddings.

No vector database.

No MCP.

No hosted service.

---

# 29. V2

Potential additions:

* embeddings
* hybrid semantic + lexical retrieval
* smarter reranking
* automatic memory extraction
* automatic memory conflict resolution
* Git history retrieval
* architectural dependency visualization
* multiple workspaces
* background daemon
* LSP integration

---

# 30. Success Criteria

The project is successful if an AI agent can solve repository tasks with significantly less exploration.

Example benchmark:

### Without Codebase Memory

```text
Agent
→ grep
→ read 8 files
→ grep
→ read 5 files
→ understand architecture
→ modify code
```

### With Codebase Memory

```text
Agent
→ codebase context
→ read 2 relevant symbols
→ modify code
```

The benchmark should compare:

```text
                    Baseline    Memory
Input tokens          X           Y
Tool calls            X           Y
Files read            X           Y
Latency               X           Y
Task success          X           Y
```

The primary success metric is:

> **Can the agent understand and modify an unfamiliar codebase while reading substantially less irrelevant context?**

---

# 31. Design Principles

### 1. Markdown is memory.

Human-readable, editable, version-controlled.

### 2. SQLite is derived state.

It can be deleted and rebuilt.

### 3. Structure before semantics.

Use symbols and relationships before embeddings.

### 4. Retrieve before reading.

Find the relevant code before loading source.

### 5. Symbols before files.

Return the smallest useful unit.

### 6. Incremental by default.

Never unnecessarily re-index the repository.

### 7. Skill over protocol.

The Go binary is a local capability, not a network service.

### 8. Source is truth.

Generated knowledge must never permanently override actual source code.

### 9. Memory should evolve.

Facts can be updated, deprecated, or removed.

### 10. Optimize for task completion.

The objective is not "70× fewer tokens."

The objective is:

> **Minimum context required for the agent to correctly understand and complete the task.**

