package main

import (
	"fmt"
	"os"
	"strings"

	"cortex/internal/agents"
)

func printSkillUsage() {
	fmt.Print(`Usage: cortex skill [install|update] [flags]

Flags:
  -g, --global     Install skills globally in user home directory
  --codex          Install for Codex
  --omp            Install for OMP
  --opencode       Install for OpenCode
  --claude         Install for Claude
  --pi             Install for Pi
  --shared         Install for shared (.agents/skills)
  --all              Install for all native agents (default if none specified)
  --agent <list>     Comma-separated list of agents (e.g. --agent codex,omp)
  --skip-existing    Skip installation if skill file already exists (default: override)
  --override, -f     Override / update existing skills (enabled by default)
  --dir <path>       Target project directory (default: current repository)
`)
}

func cmdSkill(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printSkillUsage()
		return nil
	}
	if args[0] == "update" {
		return cmdSkillInstall(append([]string{"--override"}, args[1:]...))
	}
	if args[0] != "install" {
		return fmt.Errorf("unknown skill command %q; use `cortex skill install` or `cortex skill update`", args[0])
	}
	if len(args) > 1 && (args[1] == "help" || args[1] == "--help" || args[1] == "-h") {
		printSkillUsage()
		return nil
	}
	return cmdSkillInstall(args[1:])
}

func containsAgentSlice(list []agents.Agent, a agents.Agent) bool {
	for _, item := range list {
		if item == a {
			return true
		}
	}
	return false
}

func parseAgentFlags(args []string) ([]agents.Agent, error) {
	var selected []agents.Agent
	if hasFlag(args, "--opencode") {
		selected = append(selected, agents.AgentOpenCode)
	}
	if hasFlag(args, "--codex") {
		selected = append(selected, agents.AgentCodex)
	}
	if hasFlag(args, "--omp") {
		selected = append(selected, agents.AgentOMP)
	}
	if hasFlag(args, "--pi") {
		selected = append(selected, agents.AgentPi)
	}
	if hasFlag(args, "--claude") {
		selected = append(selected, agents.AgentClaude)
	}
	if hasFlag(args, "--shared") {
		selected = append(selected, agents.AgentShared)
	}

	if hasFlag(args, "--all") {
		all, _ := agents.ParseAgents("all")
		for _, a := range all {
			if !containsAgentSlice(selected, a) {
				selected = append(selected, a)
			}
		}
	}

	if agentRaw := flagOr(args, "--agent"); agentRaw != "" {
		parsed, err := agents.ParseAgents(agentRaw)
		if err != nil {
			return nil, err
		}
		for _, a := range parsed {
			if !containsAgentSlice(selected, a) {
				selected = append(selected, a)
			}
		}
	}

	if len(selected) > 0 {
		return selected, nil
	}

	return agents.ParseAgents("all")
}

func cmdSkillInstall(args []string) error {
	selected, err := parseAgentFlags(args)
	if err != nil {
		return err
	}
	root, err := findRoot(flagOr(args, "--dir"))
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	isGlobal := hasFlag(args, "--global") || hasFlag(args, "-g")
	skipExisting := hasFlag(args, "--skip-existing")
	results := agents.Install(agents.InstallOptions{
		Root:   root,
		Home:   home,
		Global: isGlobal,
		Agents: selected,
		Force:  !skipExisting,
	})
	agents.SortResults(results)
	fmt.Println("# Cortex Skill Installation")
	fmt.Println()
	fmt.Printf("Scope: %s\n", map[bool]string{true: "global", false: "project"}[isGlobal])
	fmt.Printf("Agents: %s\n\n", strings.Join(agentNames(selected), ", "))
	for _, result := range results {
		if result.Err != nil {
			fmt.Printf("- **%s** `%s` — %s: %v\n", result.Agent, result.Path, result.Action, result.Err)
		} else {
			fmt.Printf("- **%s** `%s` — %s\n", result.Agent, result.Path, result.Action)
		}
	}
	for _, result := range results {
		if result.Err != nil {
			return fmt.Errorf("skill installation had conflicts; rerun with --override or --force to replace: %s", result.Path)
		}
	}
	return nil
}

func agentNames(as []agents.Agent) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = string(a)
	}
	return out
}

func cmdAgents(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print("Usage: cortex agents init [--dir PATH]\n")
		return nil
	}
	if args[0] != "init" {
		return fmt.Errorf("unknown agents command %q; use `cortex agents init`", args[0])
	}
	root, err := findRoot(flagOr(args[1:], "--dir"))
	if err != nil {
		return err
	}
	action, err := agents.UpdateAgents(root)
	if err != nil {
		return err
	}
	fmt.Printf("# Cortex Agent Instructions\n\n- AGENTS.md: **%s**\n- Path: `%s/AGENTS.md`\n", action, root)
	return nil
}
