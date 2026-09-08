# Cortex Benchmark Suite & Prompt Guide

A comprehensive guide and ready-to-use evaluation suite for benchmarking AI coding agents with and without Cortex persistent codebase memory.

---

### 1. The Anatomy of a Good Test Prompt

A standard vague prompt like *"tell me about this repo"* tests generic LLM summarization rather than codebase intelligence. A comprehensive test prompt contains four elements:

```text
[Task / Goal] + [Technical Scope] + [Verifiable Deliverables] + [Constraint]
```

1. **Specific Technical Goal:** Targets real code paths, dependencies, or architectural flows.
2. **Explicit Verification Criteria:** Demands exact symbol names, signatures, and `file:line` references (so you can score accuracy).
3. **Execution Constraints:** Explicitly tests the agent's workflow discipline (e.g., *"Retrieve the necessary context using the fewest tool calls possible without reading entire files"*).

---

### 2. The 4 Benchmark Archetypes

To get a complete, balanced evaluation, test across four distinct developer scenarios:

```
┌────────────────────────────────────────────────────────┐
│               Test Evaluation Matrix                   │
├───────────────────────────┬────────────────────────────┤
│ 1. Architecture Flow      │ 2. Targeted Symbol Tracing │
│ (Breadth & Layers)        │ (Call Graph & References)  │
├───────────────────────────┼────────────────────────────┤
│ 3. Impact / Refactor      │ 4. Convention Adherence    │
│ (Multi-module Ripple)     │ (Policy & Memory Rules)    │
└───────────────────────────┴────────────────────────────┘
```

---

### 3. Ready-to-Use Test Prompts

#### Test 1: High-Level Architecture & Data Flow (Breadth)

> **Prompt:**  
> *"Explain the end-to-end lifecycle when a user runs cortex update. Trace how files are scanned, hashed, parsed by Tree-sitter, and transactionally replaced in SQLite. Specify the exact functions involved, their file paths, and why unchanged files are skipped."*  

- **What this tests:** Can the agent understand cross-package flows (`cmd` → `internal/index` → `internal/lang` → `internal/store`) in a single retrieval step without grepping 10 different files?
- **Success Criteria:** Mentions `Incremental`, `Store.ReplaceFile`, file SHA-256 comparison, and transactional rollbacks.

---

#### Test 2: Deep Call Graph & Reference Tracing (Depth)

> **Prompt:**  
> *"Locate the ReplaceFile method. List its exact signature, enclosing struct, all incoming callers, and all database tables modified inside it. Do this without loading entire files."*  

- **What this tests:** Tests Cortex's structural index (`cortex symbol "ReplaceFile"` / `cortex refs "ReplaceFile"`) vs manual `grep`/`ripgrep`.
- **Success Criteria:**
  - **Signature:** `func (s *Store) ReplaceFile(f File, content string, res lang.Result) error`
  - **Location:** `internal/store/store.go:205`
  - **Callers:** `Full`, `Incremental` in `internal/index`.

---

#### Test 3: Impact Analysis / Refactor Planning (Dependency Ripple)

> **Prompt:**  
> *"We plan to refactor cmdSkillInstall to add a --json output format. Identify all callers, dependencies, flags parsed, and any downstream agent instruction generators that might be impacted."*  

- **What this tests:** `cortex deps` and intent-aware ranking.
- **Success Criteria:** Identifies [cmd/cortex/agents_cmd.go](file:///Users/hy4-mac-002/hasdev/research/cortex-index/cmd/cortex/agents_cmd.go), `agents.Install`, `agents.UpdateAgents`, and CLI dispatch in [cmd/cortex/main.go](file:///Users/hy4-mac-002/hasdev/research/cortex-index/cmd/cortex/main.go).

---

#### Test 4: Architecture Conventions & Constraints (Memory Retrieval)

> **Prompt:**  
> *"What rules and constraints must be followed when adding a new CLI command to Cortex? Specifically address standard library usage, external network dependencies, test fixtures, and Git commit policies."*  

- **What this tests:** Whether the agent pulls durable conventions from `.cortex/conventions.md` and `.cortex/project.md` instead of guessing.
- **Success Criteria:**
  - Local-first (no network or daemon dependencies).
  - Pure standard library / cgo Tree-sitter / pure-Go SQLite driver.
  - Test fixtures using `t.TempDir()`.
  - One Git commit per feature in format `<type>(TNN): <summary>`.

---

### 4. Metrics Scorecard to Track

When running the prompt in both arms (**With Cortex** vs **Without Cortex**), record these 5 metrics:

| Metric | How to Measure | Cortex Advantage |
| :--- | :--- | :--- |
| **Tool Call Count** | Total bash / read / search invocations | Expected: **1 vs 5–10** |
| **Context Tokens** | Total prompt + context tokens consumed | Expected: **15–40% lower** |
| **Response Latency** | Time until first token & total completion | Expected: **Lower reasoning latency** |
| **Hallucination Rate** | Check if file paths and line numbers actually exist | Expected: **0% hallucination** with Cortex index |
| **Context Pollution** | Lines of unnecessary code added to history | Cortex only returns targeted symbols |
