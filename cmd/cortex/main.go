package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortex-index/cortex/pkg/brain"
	"github.com/cortex-index/cortex/pkg/knowledge"
	"github.com/cortex-index/cortex/pkg/mcp"
	"github.com/cortex-index/cortex/pkg/project"
	"github.com/cortex-index/cortex/pkg/search"
	"github.com/cortex-index/cortex/pkg/skills"
	"github.com/cortex-index/cortex/pkg/tasks"
)

var version = "2.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	if command == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	workDir, _ := os.Getwd()

	for i, arg := range os.Args {
		if arg == "--workspace" && i+1 < len(os.Args) {
			workDir = os.Args[i+1]
		} else if strings.HasPrefix(arg, "--workspace=") {
			workDir = strings.TrimPrefix(arg, "--workspace=")
		}
	}

	proj := project.Discover(workDir)

	switch command {
	case "init":
		if proj != nil {
			log.Fatalf("Cortex project already exists at %s", proj.CortexDir)
		}
		p, err := project.Init(workDir)
		if err != nil {
			log.Fatalf("failed initializing .cortex: %v", err)
		}
		projName := filepath.Base(p.Root)
		if err := brain.Init(p.CortexDir, projName); err != nil {
			log.Printf("warning: failed to init brain.md: %v", err)
		}
		if err := knowledge.InitCodebase(p.CortexDir); err != nil {
			log.Printf("warning: failed to init codebase.md: %v", err)
		}
		// Generate initial codebase.md content
		if err := knowledge.UpdateCodebase(p.CortexDir, p.Root); err != nil {
			log.Printf("warning: failed to generate initial codebase.md: %v", err)
		}
		fmt.Printf("✓ Initialized Cortex 2.0 project '%s'\n", projName)

	case "update":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if err := proj.Update(); err != nil {
			log.Fatalf("failed updating project metadata: %v", err)
		}
		if err := knowledge.UpdateCodebase(proj.CortexDir, proj.Root); err != nil {
			log.Fatalf("failed updating codebase.md: %v", err)
		}
		fmt.Println("✓ Updated project metadata and codebase.md")

	case "doctor":
		fmt.Println("=== Cortex 2.0 Doctor ===")
		fmt.Println("✓ Go runtime: OK")
		if proj != nil {
			fmt.Printf("✓ .cortex: Found at %s\n", proj.CortexDir)
			fmt.Printf("✓ project.json: Present (Language: %s, Framework: %s)\n", proj.Meta.Language, proj.Meta.Framework)
			if _, err := os.Stat(brain.Path(proj.CortexDir)); err == nil {
				fmt.Println("✓ brain.md: Present")
			} else {
				fmt.Println("✗ brain.md: Missing")
			}
			if _, err := os.Stat(knowledge.CodebasePath(proj.CortexDir)); err == nil {
				fmt.Println("✓ codebase.md: Present")
			} else {
				fmt.Println("✗ codebase.md: Missing")
			}
		} else {
			fmt.Println("✗ .cortex: Not found (run 'cortex init')")
		}

	case "brain":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			content, err := brain.Get(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed reading brain: %v", err)
			}
			fmt.Println(content)
			return
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "add", "append":
			content := ""
			if len(os.Args) > 3 {
				content = strings.Join(os.Args[3:], " ")
			} else {
				log.Fatal("usage: cortex brain add <content>")
			}
			_, err := brain.Append(proj.CortexDir, content)
			if err != nil {
				log.Fatalf("failed adding to brain: %v", err)
			}
			fmt.Println("Learned.")
		case "show":
			content, err := brain.Get(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed reading brain: %v", err)
			}
			fmt.Println(content)
		case "search":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex brain search <query>")
			}
			queryStr := strings.Join(os.Args[3:], " ")
			results, err := brain.Search(proj.CortexDir, queryStr)
			if err != nil {
				log.Fatalf("failed searching brain: %v", err)
			}
			for _, r := range results {
				fmt.Println(r)
			}
		case "rule":
			if len(os.Args) < 5 || os.Args[3] != "add" {
				log.Fatal("usage: cortex brain rule add <rule>")
			}
			ruleStr := strings.Join(os.Args[4:], " ")
			_, err := brain.Append(proj.CortexDir, "RULE: "+ruleStr)
			if err != nil {
				log.Fatalf("failed adding rule: %v", err)
			}
			fmt.Println("Rule added.")
		case "permission":
			if len(os.Args) < 5 || os.Args[3] != "add" {
				log.Fatal("usage: cortex brain permission add <permission>")
			}
			permStr := strings.Join(os.Args[4:], " ")
			_, err := brain.Append(proj.CortexDir, "PERMISSION: "+permStr)
			if err != nil {
				log.Fatalf("failed adding permission: %v", err)
			}
			fmt.Println("Permission added.")
		case "instruction":
			if len(os.Args) < 5 || os.Args[3] != "set" {
				log.Fatal("usage: cortex brain instruction set <prompt>")
			}
			instStr := strings.Join(os.Args[4:], " ")
			_, err := brain.Append(proj.CortexDir, "INSTRUCTION: "+instStr)
			if err != nil {
				log.Fatalf("failed setting instruction: %v", err)
			}
			fmt.Println("Instruction set.")
		case "prune":
			err := brain.Prune(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed pruning brain: %v", err)
			}
			fmt.Println("Brain pruned.")
		default:
			fmt.Println("Usage: cortex brain [add|show|search|rule add|permission add|instruction set|prune]")
		}

	case "codebase":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			content, err := knowledge.GetCodebase(proj.CortexDir)
			if err != nil {
				// GetCodebase helper function might not exist in knowledge, use os.ReadFile instead
				p := knowledge.CodebasePath(proj.CortexDir)
				data, err := os.ReadFile(p)
				if err != nil {
					log.Fatalf("failed reading codebase.md: %v", err)
				}
				content = string(data)
			}
			fmt.Println(content)
			return
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "init":
			err := knowledge.InitCodebase(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed initializing codebase.md: %v", err)
			}
			fmt.Println("✓ Initialized codebase.md")
		case "update":
			err := knowledge.UpdateCodebase(proj.CortexDir, proj.Root)
			if err != nil {
				log.Fatalf("failed updating codebase.md: %v", err)
			}
			fmt.Println("✓ Updated codebase.md")
		case "show", "get":
			p := knowledge.CodebasePath(proj.CortexDir)
			data, err := os.ReadFile(p)
			if err != nil {
				log.Fatalf("failed reading codebase.md: %v", err)
			}
			fmt.Println(string(data))
		case "add", "append":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex codebase append <line>")
			}
			line := strings.Join(os.Args[3:], " ")
			p := knowledge.CodebasePath(proj.CortexDir)
			f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				log.Fatalf("failed opening codebase.md: %v", err)
			}
			defer f.Close()
			if _, err := f.WriteString("\n" + line + "\n"); err != nil {
				log.Fatalf("failed writing to codebase.md: %v", err)
			}
			fmt.Println("Appended to codebase.md")
		default:
			fmt.Println("Usage: cortex codebase [init|update|show|append]")
		}

	case "task":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			fmt.Println("Usage: cortex task [create|update|continue|complete|list|show|delete]")
			return
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "create":
			if len(os.Args) < 5 {
				log.Fatal("usage: cortex task create <name> <goal> [requirements]")
			}
			name := os.Args[3]
			goal := os.Args[4]
			reqs := ""
			if len(os.Args) > 5 {
				reqs = strings.Join(os.Args[5:], " ")
			}
			t, err := tasks.Create(proj.CortexDir, name, goal, reqs)
			if err != nil {
				log.Fatalf("failed creating task: %v", err)
			}
			fmt.Printf("✓ Created task '%s' at %s\n", t.Name, t.FilePath)
		case "update":
			if len(os.Args) < 5 {
				log.Fatal("usage: cortex task update <name> <status>")
			}
			name := os.Args[3]
			status := os.Args[4]
			err := tasks.UpdateStatus(proj.CortexDir, name, status)
			if err != nil {
				log.Fatalf("failed updating task: %v", err)
			}
			fmt.Printf("✓ Updated task '%s' status to %s\n", name, status)
		case "continue":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex task continue <name>")
			}
			name := os.Args[3]
			err := tasks.Continue(proj.CortexDir, name)
			if err != nil {
				log.Fatalf("failed continuing task: %v", err)
			}
			fmt.Printf("✓ Activated task '%s' as IN_PROGRESS\n", name)
		case "complete":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex task complete <name>")
			}
			name := os.Args[3]
			err := tasks.Complete(proj.CortexDir, name)
			if err != nil {
				log.Fatalf("failed completing task: %v", err)
			}
			fmt.Printf("✓ Completed task '%s'\n", name)
		case "list":
			all, err := tasks.List(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed listing tasks: %v", err)
			}
			fmt.Println("=== Cortex Tasks ===")
			for _, t := range all {
				fmt.Printf("- %s [%s]: %s\n", t.Name, t.Status, t.Goal)
			}
		case "show":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex task show <name>")
			}
			name := os.Args[3]
			data, err := tasks.Show(proj.CortexDir, name)
			if err != nil {
				log.Fatalf("failed showing task: %v", err)
			}
			fmt.Println(data)
		case "delete":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex task delete <name>")
			}
			name := os.Args[3]
			err := tasks.Delete(proj.CortexDir, name)
			if err != nil {
				log.Fatalf("failed deleting task: %v", err)
			}
			fmt.Printf("✓ Deleted task '%s'\n", name)
		default:
			fmt.Println("Usage: cortex task [create|update|continue|complete|list|show|delete]")
		}

	case "skill":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			fmt.Println("Usage: cortex skill [install|uninstall|list|search|update]")
			return
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "install":
			global := false
			name := ""
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--global" || os.Args[i] == "-g" {
					global = true
				} else {
					name = os.Args[i]
				}
			}
			if name == "" {
				log.Fatal("usage: cortex skill install [--global] <name>")
			}
			s, err := skills.Install(proj.CortexDir, name, global, "")
			if err != nil {
				log.Fatalf("failed installing skill: %v", err)
			}
			fmt.Printf("✓ Installed skill '%s' to %s\n", s.Name, s.FilePath)
		case "uninstall":
			global := false
			name := ""
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--global" || os.Args[i] == "-g" {
					global = true
				} else {
					name = os.Args[i]
				}
			}
			if name == "" {
				log.Fatal("usage: cortex skill uninstall [--global] <name>")
			}
			err := skills.Uninstall(proj.CortexDir, name, global)
			if err != nil {
				log.Fatalf("failed uninstalling skill: %v", err)
			}
			fmt.Printf("✓ Uninstalled skill '%s'\n", name)
		case "list":
			all, err := skills.List(proj.CortexDir)
			if err != nil {
				log.Fatalf("failed listing skills: %v", err)
			}
			fmt.Println("=== Cortex Skills ===")
			for _, s := range all {
				scope := "workspace"
				if s.IsGlobal {
					scope = "global"
				}
				fmt.Printf("- %s (%s): %s\n", s.Name, scope, s.Description)
			}
		case "search":
			if len(os.Args) < 4 {
				log.Fatal("usage: cortex skill search <query>")
			}
			queryStr := strings.Join(os.Args[3:], " ")
			results, err := skills.Search(proj.CortexDir, queryStr)
			if err != nil {
				log.Fatalf("failed searching skills: %v", err)
			}
			for _, s := range results {
				scope := "workspace"
				if s.IsGlobal {
					scope = "global"
				}
				fmt.Printf("- %s (%s): %s\n", s.Name, scope, s.Description)
			}
		case "update":
			global := false
			name := ""
			content := ""
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--global" || os.Args[i] == "-g" {
					global = true
				} else if name == "" {
					name = os.Args[i]
				} else {
					content = strings.Join(os.Args[i:], " ")
					break
				}
			}
			if name == "" || content == "" {
				log.Fatal("usage: cortex skill update [--global] <name> <content>")
			}
			s, err := skills.Install(proj.CortexDir, name, global, content)
			if err != nil {
				log.Fatalf("failed updating skill: %v", err)
			}
			fmt.Printf("✓ Updated skill '%s'\n", s.Name)
		default:
			fmt.Println("Usage: cortex skill [install|uninstall|list|search|update]")
		}

	case "search":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			log.Fatal("usage: cortex search <query>")
		}
		queryStr := strings.Join(os.Args[2:], " ")
		results, err := search.Search(proj.Root, proj.CortexDir, queryStr)
		if err != nil {
			log.Fatalf("search failed: %v", err)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "grep":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			log.Fatal("usage: cortex grep <query>")
		}
		queryStr := strings.Join(os.Args[2:], " ")
		results, err := search.Grep(proj.Root, queryStr)
		if err != nil {
			log.Fatalf("grep failed: %v", err)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "glob":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		if len(os.Args) < 3 {
			log.Fatal("usage: cortex glob <pattern>")
		}
		pattern := os.Args[2]
		results, err := search.Glob(proj.Root, pattern)
		if err != nil {
			log.Fatalf("glob failed: %v", err)
		}
		for _, r := range results {
			fmt.Println(r)
		}

	case "mcp":
		if proj == nil {
			log.Fatal("No Cortex project found (run 'cortex init')")
		}
		sub := ""
		if len(os.Args) > 2 {
			sub = os.Args[2]
		}
		if sub == "" {
			sub = "stdio"
		}

		mcpServer := mcp.NewServer(proj.CortexDir, proj.Root)

		if sub == "stdio" {
			ctx := context.Background()
			if err := mcpServer.ServeStdio(ctx); err != nil {
				log.Fatalf("mcp stdio error: %v", err)
			}
		} else if sub == "call" {
			if len(os.Args) < 4 {
				fmt.Println("Usage: cortex mcp call <toolName> [jsonArgs]")
				break
			}
			toolName := os.Args[3]
			argsMap := make(map[string]interface{})
			if len(os.Args) > 4 {
				raw := os.Args[4]
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					argsMap = obj
				} else {
					argsMap["name"] = raw
					argsMap["query"] = raw
					argsMap["pattern"] = raw
					argsMap["path"] = raw
					argsMap["content"] = raw
					argsMap["line"] = raw
				}
			}
			req := mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
			}
			params := map[string]interface{}{
				"name":      toolName,
				"arguments": argsMap,
			}
			paramBytes, _ := json.Marshal(params)
			req.Params = paramBytes

			resp := mcpServer.HandleRequest(context.Background(), req)
			respJSON, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(respJSON))
		} else {
			fmt.Println("Usage: cortex mcp [stdio|call <toolName> <arg>]")
		}

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Print(`Cortex 2.0 — Local-First AI Workspace Manager

Usage:
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
  cortex mcp call                  Execute MCP tool directly
`)
}
