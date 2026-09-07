package main

import (
	"fmt"
	"os"
	"strings"

	"cortex/internal/agents"
)

func cmdSkill(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Print("Usage: cortex skill install [--agent all|opencode,codex,omp,pi,claude,shared] [--global] [--dir PATH] [--force]\n")
		return nil
	}
	if args[0] != "install" {
		return fmt.Errorf("unknown skill command %q; use `cortex skill install`", args[0])
	}
	return cmdSkillInstall(args[1:])
}

func cmdSkillInstall(args []string) error {
	agentRaw := flagOr(args, "--agent")
	selected, err := agents.ParseAgents(agentRaw)
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
	results := agents.Install(agents.InstallOptions{
		Root:   root,
		Home:   home,
		Global: hasFlag(args, "--global"),
		Agents: selected,
		Force:  hasFlag(args, "--force"),
	})
	agents.SortResults(results)
	fmt.Println("# Cortex Skill Installation")
	fmt.Println()
	fmt.Printf("Scope: %s\n", map[bool]string{true: "global", false: "project"}[hasFlag(args, "--global")])
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
			return fmt.Errorf("skill installation had conflicts; rerun with --force to replace: %s", result.Path)
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
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
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
