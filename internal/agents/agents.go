// Package agents installs Cortex's Agent Skill and generates repository
// instructions for coding agents.
package agents

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

//go:embed assets/SKILL.md
var embeddedSkill []byte

const SkillName = "cortex"

// Agent is a supported native skill destination.
type Agent string

const (
	AgentOpenCode Agent = "opencode"
	AgentCodex    Agent = "codex"
	AgentOMP      Agent = "omp"
	AgentPi       Agent = "pi"
	AgentClaude   Agent = "claude"
	AgentShared   Agent = "shared"
)

var agentOrder = []Agent{AgentOpenCode, AgentCodex, AgentOMP, AgentPi, AgentClaude, AgentShared}

// InstallOptions controls skill installation.
type InstallOptions struct {
	Root   string
	Home   string
	Global bool
	Agents []Agent
	Force  bool
}

// InstallResult reports one destination action.
type InstallResult struct {
	Agent  Agent
	Path   string
	Action string // installed, skipped, conflict
	Err    error
}

// SkillContent returns a copy of the embedded canonical skill.
func SkillContent() []byte { return append([]byte(nil), embeddedSkill...) }

// SupportedAgents returns the stable supported-agent ordering.
func SupportedAgents() []Agent { return append([]Agent(nil), agentOrder...) }

// ParseAgents parses a comma-separated list. "all" selects native agents;
// shared is explicit to avoid duplicate skill copies by default.
func ParseAgents(raw string) ([]Agent, error) {
	if raw == "" || raw == "all" {
		return []Agent{AgentOpenCode, AgentCodex, AgentOMP, AgentPi, AgentClaude}, nil
	}
	seen := map[Agent]bool{}
	var out []Agent
	for _, item := range strings.Split(raw, ",") {
		a := Agent(strings.ToLower(strings.TrimSpace(item)))
		if a == "" {
			continue
		}
		if !containsAgent(a) {
			return nil, fmt.Errorf("unknown agent %q (choose opencode,codex,omp,pi,claude,shared, or all)", a)
		}
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no agents selected")
	}
	return out, nil
}

func containsAgent(a Agent) bool {
	for _, known := range agentOrder {
		if a == known {
			return true
		}
	}
	return false
}

// Destination returns the platform-specific skill path for one agent.
func Destination(root, home string, global bool, agent Agent) (string, error) {
	base := root
	if global {
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return "", err
			}
		}
		base = home
	}
	var dir string
	switch agent {
	case AgentOpenCode:
		if global {
			dir = filepath.Join(base, ".config", "opencode", "skills")
		} else {
			dir = filepath.Join(base, ".opencode", "skills")
		}
	case AgentCodex:
		dir = filepath.Join(base, ".codex", "skills")
	case AgentOMP:
		if global {
			dir = filepath.Join(base, ".omp", "agent", "skills")
		} else {
			dir = filepath.Join(base, ".omp", "skills")
		}
	case AgentPi:
		if global {
			dir = filepath.Join(base, ".pi", "agent", "skills")
		} else {
			dir = filepath.Join(base, ".pi", "skills")
		}
	case AgentClaude:
		dir = filepath.Join(base, ".claude", "skills")
	case AgentShared:
		dir = filepath.Join(base, ".agents", "skills")
	default:
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
	return filepath.Join(dir, SkillName, "SKILL.md"), nil
}

// Install writes the embedded skill safely to every selected destination.
func Install(o InstallOptions) []InstallResult {
	if len(o.Agents) == 0 {
		o.Agents, _ = ParseAgents("all")
	}
	results := make([]InstallResult, 0, len(o.Agents))
	for _, agent := range o.Agents {
		path, err := Destination(o.Root, o.Home, o.Global, agent)
		result := InstallResult{Agent: agent, Path: path}
		if err != nil {
			result.Action, result.Err = "conflict", err
			results = append(results, result)
			continue
		}
		old, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Equal(old, embeddedSkill) {
			result.Action = "skipped"
			results = append(results, result)
			continue
		}
		if readErr == nil && !o.Force {
			result.Action = "conflict"
			result.Err = fmt.Errorf("file exists with different content; use --force")
			results = append(results, result)
			continue
		}
		if err := atomicWrite(path, embeddedSkill, 0o644); err != nil {
			result.Action, result.Err = "conflict", err
		} else {
			result.Action = "installed"
		}
		results = append(results, result)
	}
	return results
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	oldMode := mode
	if info, err := os.Stat(path); err == nil {
		oldMode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cortex-write-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(oldMode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

const agentsBegin = "<!-- cortex:begin -->"
const agentsEnd = "<!-- cortex:end -->"

const agentsBlock = `<!-- cortex:begin -->
## Cortex codebase memory

Before broad repository exploration, run:

    cortex context "<task>"

Prefer targeted structural retrieval before reading whole files:

- cortex symbol "<Name>" [--src] for definitions and focused source
- cortex refs "<Name>" for callers and references
- cortex deps "<Name>" for callees, callers, and imports
- cortex search "<terms>" for lexical symbol/file search

Treat .cortex/*.md as durable, human-editable project memory. Record only lasting
architecture, conventions, decisions, and constraints, with source, updated_at,
and status provenance when possible. Do not store temporary task chatter.

Run cortex update after code changes. Source code is authoritative when it
contradicts stale Markdown. If Cortex is unavailable or the index is incomplete,
continue with normal exploration and run cortex index when possible.
<!-- cortex:end -->`

// AgentsBlock returns the generated managed content.
func AgentsBlock() string { return agentsBlock }

// UpdateAgents writes a root AGENTS.md, preserving all unmanaged content.
func UpdateAgents(root string) (action string, err error) {
	path := filepath.Join(root, "AGENTS.md")
	old, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return "error", readErr
	}
	if os.IsNotExist(readErr) {
		if err := atomicWrite(path, []byte(agentsBlock+"\n"), 0o644); err != nil {
			return "error", err
		}
		return "created", nil
	}
	updated := replaceOrAppend(string(old), agentsBlock)
	if updated == string(old) {
		return "skipped", nil
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	if err := atomicWrite(path, []byte(updated), mode); err != nil {
		return "error", err
	}
	return "updated", nil
}

func replaceOrAppend(old, block string) string {
	start := strings.Index(old, agentsBegin)
	if start >= 0 {
		endRel := strings.Index(old[start+len(agentsBegin):], agentsEnd)
		if endRel >= 0 {
			end := start + len(agentsBegin) + endRel + len(agentsEnd)
			return old[:start] + block + old[end:]
		}
	}
	if strings.TrimSpace(old) == "" {
		return block + "\n"
	}
	return strings.TrimRight(old, "\n") + "\n\n" + block + "\n"
}

// Runtime reports build/runtime context useful to diagnostics.
func Runtime() string { return runtime.GOOS + "/" + runtime.GOARCH }

// ResultDigest returns a short stable digest of installed content.
func ResultDigest() string {
	sum := sha256.Sum256(embeddedSkill)
	return fmt.Sprintf("%x", sum[:6])
}

// SortResults gives deterministic output ordering by agent then path.
func SortResults(results []InstallResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Agent != results[j].Agent {
			return results[i].Agent < results[j].Agent
		}
		return results[i].Path < results[j].Path
	})
}
