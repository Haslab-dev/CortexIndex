package memory

import "fmt"

func fmtSscan(s string, args ...any) (int, error) {
	return fmt.Sscanf(s, "%d", args...)
}

// configTemplate guides durable configuration; parsed by LoadConfig.
const configTemplate = `# Cortex Configuration

Optional frontmatter (parsed by cortex). Everything below the frontmatter
is free-form notes for humans and agents.

---
ignore: []
max-file-size: 1048576
languages: []
---

## Notes

Add repository-wide configuration notes here.
`

// projectTemplate is the top-level project memory (PRD §8).
const projectTemplate = `# Project

## Purpose

<!-- What is this project? One or two sentences. -->

## Stack

<!-- Primary languages, frameworks, databases, infra. -->

## Architecture

<!-- Feature-based? Layered? Services? A short paragraph. -->

## Entry Points

<!-- main.go, cmd/, index.ts, etc. -->

## External Services

<!-- Databases, APIs, message queues this project depends on. -->
`

// architectureTemplate is the compact mental map (PRD §9).
const architectureTemplate = `# Architecture

Keep this small — a mental map, not a source dump.

## Layers

<!-- e.g.
Frontend → API → Application Services → Repository → PostgreSQL
-->

## Key Flows

<!-- e.g.
LoginPage → AuthService.login() → APIClient → TokenStore
-->
`

// conventionsTemplate holds project rules agents must respect (PRD §8/§15).
const conventionsTemplate = `# Conventions

<!-- Rules the codebase follows. Examples:
- API calls belong inside services; components never call APIClient directly.
- Errors are wrapped with %w and contextual messages.
- Tests live next to the code they cover.
-->
`

// memoryUpdateGuide is embedded in decisions/modules templates as guidance.
const memoryUpdateGuide = `
<!--
Memory etiquette (PRD §18/§19):
- Durable knowledge only, not task chatter.
  Good: "Authentication must go through AuthService."
  Bad:  "User asked to fix login button today."
- Keep provenance and status:
  source: path/to/file
  updated_at: 2026-09-07
  status: active        # active | deprecated
- Source code wins over stale memory when they conflict.
-->
`

func init() {
	decisionTemplate = `# <Decision Name>

## Decision

<!-- What was decided, e.g. "Use JWT with refresh tokens." -->

## Reason

<!-- Why, e.g. "The API is consumed by web and mobile clients." -->

## Constraint

<!-- Boundaries, e.g. "Token management must remain inside AuthService." -->

## Source

<!-- path/to/relevant/code.ts -->
` + memoryUpdateGuide

	moduleTemplate = `# <Module Name>

## Responsibility

<!-- What this module owns. -->

## Important Symbols

<!-- e.g. AuthService.login — entry point for all auth flows. -->

## Dependencies

<!-- What this module depends on. -->

## Known Constraints

<!-- Things future changes must respect. -->
` + memoryUpdateGuide
}

var (
	decisionTemplate string
	moduleTemplate   string
)
