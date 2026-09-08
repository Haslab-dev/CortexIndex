package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDestinationMatrix(t *testing.T) {
	root, home := "/project", "/home/tester"
	cases := []struct {
		agent  Agent
		global bool
		want   string
	}{
		{AgentOpenCode, false, "/project/.opencode/skills/cortex/SKILL.md"},
		{AgentOpenCode, true, "/home/tester/.config/opencode/skills/cortex/SKILL.md"},
		{AgentCodex, false, "/project/.codex/skills/cortex/SKILL.md"},
		{AgentCodex, true, "/home/tester/.codex/skills/cortex/SKILL.md"},
		{AgentOMP, false, "/project/.omp/skills/cortex/SKILL.md"},
		{AgentOMP, true, "/home/tester/.omp/agent/skills/cortex/SKILL.md"},
		{AgentPi, false, "/project/.pi/skills/cortex/SKILL.md"},
		{AgentPi, true, "/home/tester/.pi/agent/skills/cortex/SKILL.md"},
		{AgentClaude, false, "/project/.claude/skills/cortex/SKILL.md"},
		{AgentClaude, true, "/home/tester/.claude/skills/cortex/SKILL.md"},
		{AgentShared, false, "/project/.agents/skills/cortex/SKILL.md"},
		{AgentShared, true, "/home/tester/.agents/skills/cortex/SKILL.md"},
	}
	for _, tc := range cases {
		got, err := Destination(root, home, tc.global, tc.agent)
		if err != nil || got != filepath.FromSlash(tc.want) {
			t.Errorf("Destination(%s,%v) = %q, err=%v; want %q", tc.agent, tc.global, got, err, tc.want)
		}
	}
}

func TestInstallIdempotentAndForce(t *testing.T) {
	root := t.TempDir()
	results := Install(InstallOptions{Root: root, Agents: []Agent{AgentShared}})
	if len(results) != 1 || results[0].Action != "installed" || results[0].Err != nil {
		t.Fatalf("first install = %+v", results)
	}
	data, err := os.ReadFile(results[0].Path)
	if err != nil || !strings.HasPrefix(string(data), "---\nname: cortex") {
		t.Fatalf("embedded skill missing: err=%v content=%q", err, data[:minInt(len(data), 50)])
	}
	info, _ := os.Stat(results[0].Path)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("skill mode = %o", info.Mode().Perm())
	}

	results = Install(InstallOptions{Root: root, Agents: []Agent{AgentShared}})
	if results[0].Action != "skipped" {
		t.Errorf("identical second install = %+v", results)
	}
	results = Install(InstallOptions{Root: root, Agents: []Agent{AgentShared}, Force: true})
	if results[0].Action != "installed" || results[0].Err != nil {
		t.Errorf("forced install on identical = %+v", results)
	}
	if err := os.WriteFile(results[0].Path, []byte("custom"), 0o600); err != nil {
		t.Fatal(err)
	}
	results = Install(InstallOptions{Root: root, Agents: []Agent{AgentShared}})
	if results[0].Action != "conflict" || results[0].Err == nil {
		t.Errorf("conflicting install = %+v", results)
	}
	results = Install(InstallOptions{Root: root, Agents: []Agent{AgentShared}, Force: true})
	if results[0].Action != "installed" || results[0].Err != nil {
		t.Errorf("forced install = %+v", results)
	}
}

func TestParseAgents(t *testing.T) {
	got, err := ParseAgents("claude,shared,claude")
	if err != nil || len(got) != 2 || got[0] != AgentClaude || got[1] != AgentShared {
		t.Errorf("ParseAgents = %v, err=%v", got, err)
	}
	got, err = ParseAgents("all")
	if err != nil || len(got) != 5 {
		t.Errorf("all = %v, err=%v", got, err)
	}
	if _, err := ParseAgents("nope"); err == nil {
		t.Error("unknown agent should fail")
	}
}

func TestUpdateAgentsCreateAppendReplaceIdempotent(t *testing.T) {
	root := t.TempDir()
	action, err := UpdateAgents(root)
	if err != nil || action != "created" {
		t.Fatalf("create action=%q err=%v", action, err)
	}
	path := filepath.Join(root, "AGENTS.md")
	first, _ := os.ReadFile(path)
	if !strings.Contains(string(first), agentsBegin) || !strings.Contains(string(first), agentsEnd) {
		t.Fatal("managed block missing")
	}
	action, err = UpdateAgents(root)
	if err != nil || action != "skipped" {
		t.Errorf("idempotent action=%q err=%v", action, err)
	}
	if second, _ := os.ReadFile(path); string(second) != string(first) {
		t.Error("idempotent generation changed bytes")
	}
	if err := os.WriteFile(path, []byte("# Human instructions\n\n"+agentsBegin+"\nold\n"+agentsEnd+"\n\n# Keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	action, err = UpdateAgents(root)
	if err != nil || action != "updated" {
		t.Fatalf("replace action=%q err=%v", action, err)
	}
	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "# Human instructions") || !strings.Contains(string(updated), "# Keep me") || !strings.Contains(string(updated), "Before broad repository exploration") {
		t.Errorf("unmanaged content or generated block missing: %q", updated)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("existing mode not preserved: %o", info.Mode().Perm())
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
