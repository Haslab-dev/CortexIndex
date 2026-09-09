package main

import (
	"strings"
	"testing"

	"cortex/internal/history"
)

func TestHistoryOptionsValidation(t *testing.T) {
	if _, err := historyQuery([]string{"one", "two"}); err == nil {
		t.Fatal("expected multiple query error")
	}
	if _, err := historyLimit([]string{"--limit", "0"}); err == nil {
		t.Fatal("expected non-positive limit error")
	}
}

func TestFormatHistoryAndCommit(t *testing.T) {
	commits := []history.Commit{{Hash: "abcdef1234", ShortHash: "abcdef1", Author: "Test", Subject: "Add login"}}
	out := formatHistory("/tmp/repo", commits)
	if !strings.Contains(out, "# Git History") || !strings.Contains(out, "Add login") {
		t.Fatalf("history format = %s", out)
	}
	detail := history.CommitDetail{Commit: commits[0], Files: []history.ChangedFile{{Status: "M", Path: "auth.go"}}, Diff: "@@ -1 +1 @@", DiffTruncated: true}
	out = formatCommit(detail)
	for _, want := range []string{"# Git Commit", "auth.go", "Diff truncated"} {
		if !strings.Contains(out, want) {
			t.Fatalf("commit format missing %q: %s", want, out)
		}
	}
}
