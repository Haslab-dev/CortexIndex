# Cortex Index — Local-First AI Workspace Manager

> Persistent knowledge graph for your codebase. Cortex Index converts source code into a queryable graph with symbol resolution, call tracking, and token-optimized AI context building — all local, no cloud, no vector DB.

## ⚡ Key Features

- **20 MCP Tools** — Full Model Context Protocol (MCP v1.0) over stdio JSON-RPC 2.0
- **🧠 Brain System** — Persistent `.cortex/brain.md` stores project purpose, rules, and knowledge. Created on `cortex init`, queryable via `cortex_brain_get/update/append` MCP tools. AI agents learn once, remember forever.
- **Search Layer** — Three search tools: recursive knowledge base search, line-level grep across workspace files, and glob pattern matching for file discovery.
- **Skills Management** — Install, uninstall, list, search, and update reusable AI skill conventions templates. Both workspace-scoped and global.
- **Task Tracking** — Create, update, complete, list, show, and delete tracked tasks in `.cortex/tasks/`.
- **Codebase Documentation** — Auto-generated architecture overview (`codebase.md`) with metadata that stays in sync via `cortex update`.
- **Project-Local** — No global database. Each project owns `.cortex/` next to it. `cortex init` creates it. `cortex clean` resets it.
- **Embedded Storage** — Pebble LSM tree embedded KV store at `.cortex/data/`.

---

## 🏗️ Architecture

```
                     ┌──────────────┐
                     │  AI Agent     │
                     │ (Claude/etc)  │
                     └──────┬───────┘
                             │ MCP (Stdio)
                ┌─────────────────────────┐
                │    Cortex MCP Server     │
                │  20 tools · JSON-RPC 2.0 │
                └─────────────────────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                  │
      ┌────▼────┐     ┌────▼─────┐     ┌──────▼──────┐
      │  Brain   │     │  Search   │     │  Skills &    │
      │ (3 tools)│     │ (3 tools) │     │  Tasks       │
      └─────────┘     └──────────┘     │  (7 tools)   │
                                        └──────────────┘
                                                │
                          ┌─────────────────────┼─────────────────────┐
                          │                                         │
                    ┌─────▼─────┐                           ┌──────▼──────┐
                    │  Codebase  │                           │   Project    │
                    │  (3 tools) │                           │   Metadata   │
                    └───────────┘                           └──────────────┘
```

---

## 🚀 Getting Started

```bash
# Install
git clone https://github.com/Haslab-dev/CortexIndex.git
cd CortexIndex && make && make install

# Create a project
cd ~/my-project
cortex init            # creates .cortex/ + project.json + brain.md + codebase.md + full index
cortex update          # refresh project metadata and codebase.md

# Quick MCP test
cortex mcp call cortex_brain_get '{}'
```

---

## 🧠 Brain System — Persistent Agent Knowledge

The brain file (`.cortex/brain.md`) records your project's purpose, rules, and explanations. Created automatically on `cortex init`. AI agents read it to understand context without codebase exploration.

```bash
# Read brain
cortex brain

# Add knowledge (append)
cortex brain add "Uses Pebble KV storage"

# Append as "Learned"
cortex brain append "Providers self-register via init() calling Register()"

# Add a rule
cortex brain rule add "All APIs must have timeout handling"

# Add a permission
cortex brain permission add "can access database credentials"

# Set an instruction
cortex brain instruction set "Always check .cortex/brain.md before modifying code"

# Search brain content
cortex brain search "Pebble"

# Via MCP (agent-friendly)
cortex mcp call cortex_brain_append '{"line":"Uses Pebble KV storage"}'
```

The brain persists between agent sessions — knowledge learned once is retained.

---

## 📊 Performance Benchmarks

Tested on `cortex-index` (self-project):

| Operation | Time | Notes |
|:---|---:|:---|
| Brain get | **<1ms** | Read brain.md content |
| Brain append | **<1ms** | append-only write |
| Grep workspace | **<1ms** | Recursive text search |
| Glob pattern | **<1ms** | File path matching |
| Skill CRUD | **<1ms** | install/uninstall/list/search/update |
| Task CRUD | **<1ms** | create/update/complete/list/show/delete |
| Codebase init | **<1ms** | Generate architecture overview |
| Codebase update | **<1ms** | Refresh metadata |

---

## 🔌 20 MCP Tools Reference

### 🧠 Brain (3 tools)
| Tool | Description | Example |
|:---|:---|---:|
| `cortex_brain_get` | Read brain.md (purpose/rules) | `{}` |
| `cortex_brain_update` | Replace brain.md content | `{"content":"## Purpose\n..."}` |
| `cortex_brain_append` | Append line as "Learned" | `{"line":"Uses Pebble storage"}` |

### 🔍 Search (3 tools)
| Tool | Description | Example |
|:---|:---|---:|
| `cortex_search` | Search knowledge base and codebase files recursively | `{"query":"MCP"}` |
| `cortex_grep` | Search for target text within workspace files | `{"query":"cortex"}` |
| `cortex_glob` | Match file paths in workspace using glob pattern | `{"pattern":"**/*.go"}` |

### 🎯 Skills (5 tools)
| Tool | Description | Example |
|:---|:---|---:|
| `cortex_skill_install` | Install a coding skill conventions template | `{"name":"refactor"}` |
| `cortex_skill_uninstall` | Uninstall a coding skill conventions template | `{"name":"refactor"}` |
| `cortex_skill_list` | List all global and workspace skills | `{}` |
| `cortex_skill_search` | Search for skills by query text | `{"query":"refactor"}` |
| `cortex_skill_update` | Update or install coding skill content | `{"name":"refactor"}` |

### ✅ Tasks (6 tools)
| Tool | Description | Example |
|:---|:---|---:|
| `cortex_task_create` | Create a task tracking document | `{"name":"test","goal":"Test"}` |
| `cortex_task_update` | Update active task status | `{"name":"test","status":"in_progress"}` |
| `cortex_task_complete` | Mark a task tracking document completed | `{"name":"test"}` |
| `cortex_task_list` | List all registered workspace tasks | `{}` |
| `cortex_task_show` | Show detailed task instructions | `{"name":"test"}` |
| `cortex_task_delete` | Delete/remove task document from workspace | `{"name":"test"}` |

### 📁 Codebase (3 tools)
| Tool | Description | Example |
|:---|:---|---:|
| `cortex_codebase_init` | Initialize codebase.md architecture overview | `{}` |
| `cortex_codebase_update` | Update codebase.md metadata recursively | `{}` |
| `cortex_codebase_get` | Get the content of the codebase.md architecture overview file | `{}` |

---

## 🔌 MCP Client Configuration

### Claude Desktop / Antigravity / Cursor / Kilo AI

```json
{
  "mcpServers": {
    "cortex-index": {
      "command": "cortex",
      "args": ["mcp", "stdio"]
    }
  }
}
```

MCP is stdio-only. No HTTP transport. The server runs per-session and exits when the parent closes stdin.

---

## 🧪 Tutorial: AI Agent Working with Cortex

### Step 1 — Initialize the project

```bash
cortex init
# → ✓ Initialized Cortex 2.0 project 'my-project'
```

### Step 2 — Explore the brain

```bash
cortex brain
# → Shows brain.md content

cortex mcp call cortex_brain_get '{}'
# → Returns brain.md content as JSON-RPC response
```

### Step 3 — Search the codebase

```bash
cortex search "MCP"
# → Returns matching lines from knowledge base and codebase files

cortex grep "cortex"
# → Returns matching lines in workspace files

cortex glob "**/*.go"
# → Returns matching file paths
```

### Step 4 — Manage tasks

```bash
cortex task create refactor-router "Refactor API router" "Use handler pattern"
cortex task list
cortex task update refactor-router in_progress
cortex task complete refactor-router
cortex task delete refactor-router
```

### Step 5 — Manage skills

```bash
cortex skill install refactor
cortex skill list
cortex skill search refactor
cortex skill update refactor "Updated content here"
cortex skill uninstall refactor
```

### Step 6 — Persistent learning via brain

```bash
cortex brain append "Uses Pebble KV storage at .cortex/data/"
cortex brain rule add "All APIs must have timeout handling"
cortex brain permission add "can access database credentials"

# Via MCP (agent-friendly)
cortex mcp call cortex_brain_append '{"line":"Uses Pebble KV storage"}'
```

---

## 🧪 Testing & Verification

### MCP Tool Verification (2026-07-31)

All 20 MCP tools were tested against this project using the cortex MCP interface:

| # | Tool | Test Input | Result |
|:---:|:---|:---|:---:|
| 1 | `cortex_search` | `{"query":"MCP"}` | ✅ Returns matching results |
| 2 | `cortex_grep` | `{"query":"cortex"}` | ✅ Returns matching lines in workspace files |
| 3 | `cortex_glob` | `{"pattern":"**/*.go"}` | ✅ Returns matching file paths |
| 4 | `cortex_brain_get` | `{}` | ✅ Returns brain.md content |
| 5 | `cortex_brain_append` | `{"line":"Test line"}` | ✅ Appended "Learned" entry |
| 6 | `cortex_brain_update` | `{"content":"Updated"}` | ✅ Replaced brain content |
| 7 | `cortex_skill_install` | `{"name":"refactor"}` | ✅ Installed skill |
| 8 | `cortex_skill_uninstall` | `{"name":"refactor"}` | ✅ Uninstalled skill |
| 9 | `cortex_skill_list` | `{}` | ✅ Lists installed skills |
| 10 | `cortex_skill_search` | `{"query":"refactor"}` | ✅ Returns matching skills |
| 11 | `cortex_skill_update` | `{"name":"refactor"}` | ✅ Updated skill |
| 12 | `cortex_task_create` | `{"name":"test","goal":"Test"}` | ✅ Task created |
| 13 | `cortex_task_list` | `{}` | ✅ Lists tasks |
| 14 | `cortex_task_show` | `{"name":"test"}` | ✅ Shows task details |
| 15 | `cortex_task_update` | `{"name":"test","status":"in_progress"}` | ✅ Updated |
| 16 | `cortex_task_complete` | `{"name":"test"}` | ✅ Completed |
| 17 | `cortex_task_delete` | `{"name":"test"}` | ✅ Deleted |
| 18 | `cortex_codebase_init` | `{}` | ✅ Initialized codebase.md |
| 19 | `cortex_codebase_update` | `{}` | ✅ Updated codebase.md metadata |
| 20 | `cortex_codebase_get` | `{}` | ✅ Returns codebase.md content |

### Category Summary

| Category | Tools | Count | Status |
|:---|---:|:---:|:---:|
| Brain | brain_get, brain_update, brain_append | 3 | ✅ All passing |
| Search | search, grep, glob | 3 | ✅ All passing |
| Skills | skill_install, skill_uninstall, skill_list, skill_search, skill_update | 5 | ✅ All passing |
| Tasks | task_create, task_update, task_complete, task_list, task_show, task_delete | 6 | ✅ All passing |
| Codebase | codebase_init, codebase_update, codebase_get | 3 | ✅ All passing |
| **Total** | | **20** | **✅ All passing** |

---

## 🛠️ CLI Reference

```bash
cortex init                      Initialize .cortex directory, project.json, brain.md, codebase.md
cortex update                    Update project metadata and codebase.md
cortex doctor                    Check system environment and project health

cortex codebase [init|update|show|append]
                                 Manage codebase.md architecture files
cortex brain [add|show|search|rule add|permission add|instruction set|prune]
                                 Manage persistent append-only workspace brain logs
cortex task [create|update|continue|complete|list|show|delete]
                                 Manage workspace tasks in .cortex/tasks/
cortex skill [install|uninstall|list|search|update]
                                 Manage workspace and global AI skills
cortex search <query>            Orchestrate searches across knowledge and files
cortex grep <query>              Grep text in workspace
cortex glob <pattern>            Match file paths in workspace

cortex mcp stdio                 Start MCP stdio server
cortex mcp call <toolName> [jsonArgs]
                                 Execute MCP tool directly
```

---

## 📄 License

MIT License © 2026 Cortex Index Team.