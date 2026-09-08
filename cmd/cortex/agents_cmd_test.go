package main

import (
	"reflect"
	"testing"

	"cortex/internal/agents"
)

func TestParseAgentFlags(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    []agents.Agent
		wantErr bool
	}{
		{
			name: "default when no flags",
			args: []string{},
			want: []agents.Agent{agents.AgentOpenCode, agents.AgentCodex, agents.AgentOMP, agents.AgentPi, agents.AgentClaude},
		},
		{
			name: "single agent flag codex",
			args: []string{"--codex"},
			want: []agents.Agent{agents.AgentCodex},
		},
		{
			name: "single agent flag omp",
			args: []string{"--omp"},
			want: []agents.Agent{agents.AgentOMP},
		},
		{
			name: "single agent flag opencode",
			args: []string{"--opencode"},
			want: []agents.Agent{agents.AgentOpenCode},
		},
		{
			name: "single agent flag claude",
			args: []string{"--claude"},
			want: []agents.Agent{agents.AgentClaude},
		},
		{
			name: "single agent flag pi",
			args: []string{"--pi"},
			want: []agents.Agent{agents.AgentPi},
		},
		{
			name: "single agent flag shared",
			args: []string{"--shared"},
			want: []agents.Agent{agents.AgentShared},
		},
		{
			name: "multiple flags",
			args: []string{"--codex", "--omp"},
			want: []agents.Agent{agents.AgentCodex, agents.AgentOMP},
		},
		{
			name: "all flag",
			args: []string{"--all"},
			want: []agents.Agent{agents.AgentOpenCode, agents.AgentCodex, agents.AgentOMP, agents.AgentPi, agents.AgentClaude},
		},
		{
			name: "legacy agent flag",
			args: []string{"--agent", "codex,claude"},
			want: []agents.Agent{agents.AgentCodex, agents.AgentClaude},
		},
		{
			name: "global with agent flag",
			args: []string{"--global", "--codex"},
			want: []agents.Agent{agents.AgentCodex},
		},
		{
			name: "short global with agent flag",
			args: []string{"-g", "--omp"},
			want: []agents.Agent{agents.AgentOMP},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAgentFlags(tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseAgentFlags(%v) err = %v, wantErr %v", tc.args, err, tc.wantErr)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseAgentFlags(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
