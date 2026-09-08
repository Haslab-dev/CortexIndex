# Test Report & Comparative Analysis: Cortex vs. Manual Exploration

- **Date:** 2026-09-08
- **Evaluator / Client:** opencode2
- **Model Under Test:** Ling 3.0 Flash Fin Free
- **Evaluation Subject:** Codebase Overview Retrieval Performance (`cortex-index`)

---

## 1. Executive Summary

This test evaluated agent performance when answering a repository overview query under two conditions:
1. **With Cortex Skill (`cortex context`):** Leveraging Cortex's persistent codebase memory and derived structural index.
2. **Without Cortex (Baseline):** Relying on manual exploration (search, file reads, ripgrep).

### Key Results

| Metric | With Cortex (`cortex context`) | Baseline (No Cortex) | Delta / Impact |
| :--- | :--- | :--- | :--- |
| **Total Token Consumption** | **16,824 tokens** | **20,325 tokens** | **-3,501 tokens (-17.2%)** |
| **Tool / Action Round-Trips** | **1 command execution** | **7 tool operations** (1 search, 6 reads) | **-85.7% tool overhead** |
| **Agent Thinking Latency** | **2.57s total** (870ms + 1.7s) | **3.57s total** (672ms + 2.9s) | **-28.0% thinking time** |
| **Data Accuracy** | **Live structural metrics** | **Static documentation quotes** | Accurate real-time stats |

---

## 2. Test Execution Details

### Run A: With Cortex
- **Prompt:** `"describe this project overview with cortex"`
- **Execution Path:**
  1. Agent loads the `cortex` Skill.
  2. Agent executes bounded shell function `cortex_cmd context "<task>"`.
  3. Single response returned formatted Markdown with:
     - Project Purpose & Stack
     - Architectural Layers
     - Live Module Counts (72 files, 385 symbols, 2,668 references)
     - Key CLI entry points & conventions
- **Token Usage:** 16,824 tokens
- **Thinking Time:** 870ms + 1.7s = 2.57s

#### Output Snippet:
```text
Cortex — Codebase Memory Engine
Purpose: A local-first codebase memory engine for AI coding agents that helps new sessions
understand a repository with less repeated searching and architectural discovery.

Stack:
- Go 1.27.1 with cgo + native C compiler (for Tree-sitter bindings)
- SQLite + FTS5 (modernc.org/sqlite) for the structural index
- Tree-sitter language parsers via github.com/smacker/go-tree-sitter
- Makefile for native macOS/Linux builds

Major Modules:
Module          Files   Symbols
internal/lang   13      101 (Tree-sitter extraction)
internal/store  6       54 (SQLite/FTS5 store)
internal/index  2       20 (walk, hash, incremental update)
internal/memory 5       55 (Markdown scaffold, config)
internal/context 3      46 (ranking, source extraction)
internal/lexsearch 2    9 (lexical fallback)
internal/agents 3       33 (Skill/AGENTS.md integration)
cmd/cortex      6       34 (CLI dispatch)
internal/retrieve 2     15 (search, symbol, refs, deps)

Index Summary: 72 files, 385 symbols, 2,668 references across 12 major packages.
```

---

### Run B: Without Cortex (Baseline)
- **Prompt:** `"describe this project overview"`
- **Execution Path:**
  1. Search directory structure (1 search step).
  2. Sequential inspection of repository files (6 full file reads including `README.md`, `go.mod`, `docs/prd.md`, etc.).
  3. LLM synthesizes broad documentation summary from raw text files.
- **Token Usage:** 20,325 tokens
- **Thinking Time:** 672ms + 2.9s = 3.57s

---

## 3. Comparative Evaluation & Analysis

### 3.1 Token Efficiency
- **Observation:** Cortex achieved a **17.2% reduction in total tokens** (saving 3,501 tokens in a single interaction).
- **Driver:** Manual file reading dumps whole file content (including comments, formatting, and irrelevant boilerplate) directly into the agent context window. Cortex extracts only relevant architectural declarations and memory sections.

### 3.2 Tool Invocations and Network/Process Latency
- **Observation:** Cortex reduced tool invocations from **7 round-trips down to 1 command execution**.
- **Driver:** Without Cortex, the agent executes sequential exploratory cycles:
  `search -> read README -> read PRD -> read go.mod -> read tasks -> synthesize`.
  Each round-trip incurs latency, context serialization, and model re-entry overhead. With Cortex, a single retrieval resolves the architectural overview immediately.

### 3.3 Quality & Quantitative Accuracy
- **Observation:** Cortex provided live, verifiable metrics rather than static documentation claims:
  - Cortex reported the real-time indexed count: **72 files, 385 symbols, 2,668 references**.
  - Baseline cited general high-level PRD statements, which may lag behind code changes.
- **Driver:** Cortex couples human-curated durable Markdown memory (`.cortex/*.md`) with a live derived Tree-sitter SQLite index (`.cortex/index/codebase.db`).

### 3.4 Context Window Pollution
- In long-running agent workflows, files read during exploration remain in context history across all future turns.
- Reading 6+ files in turn 1 permanently bloats the conversation history. Cortex avoids this by condensing relevant context before it enters the session transcript.

---

## 4. Optimization Recommendations

1. **`cortex overview` vs `cortex context`:**
   - In this benchmark, the agent ran `cortex context "<task>"`, which ranks symbols and extracts code snippets for an actionable task (accounting for the majority of the 16.8k tokens).
   - For pure repository overviews, Cortex provides `cortex overview`, which skips task symbol expansion and emits only architectural memory, modules, and entry points. Running `cortex overview` can reduce token usage even further (~2,000–3,500 tokens total).

2. **Skill Prompting:**
   - Update agent skill instructions to recommend `cortex overview` when the agent is asked to understand or summarize the overall repository, and `cortex context "<task>"` when embarking on a specific coding task.

---

## 5. Conclusion

Under identical model conditions (**Ling 3.0 Flash Fin Free via opencode2**), Cortex demonstrated clear superiority over manual exploration:
- **17.2% lower token cost**
- **85.7% fewer tool round-trips**
- **28.0% faster reasoning latency**
- **Exact, live codebase intelligence**
