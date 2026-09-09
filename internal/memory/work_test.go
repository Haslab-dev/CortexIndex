package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorkRecord(t *testing.T) {
	content := "---\ncortex: work\nid: auth-task\nscope: task\nstatus: active\nupdated_at: 2026-09-09\n---\n\n## Goal\n\nPreserve the login API.\n\n## Plan\n\n1. Inspect callers.\n"
	w, ok, err := ParseWorkRecord("work/auth.md", content)
	if err != nil || !ok || w.ID != "auth-task" || w.Sections["Goal"] != "Preserve the login API." {
		t.Fatalf("ParseWorkRecord() = %#v, %v, %v", w, ok, err)
	}
}

func TestLoadWorkRecordsSkipsMalformed(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".cortex", "work")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	valid := "---\ncortex: work\nid: one\nstatus: blocked\nupdated_at: 2026-09-09\n---\n## Goal\n\nInvestigate.\n"
	if err := os.WriteFile(filepath.Join(path, "one.md"), []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "bad.md"), []byte("---\ncortex: work\nid: bad\n---\n## Plan\n\nNo goal.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := LoadWorkRecords(root)
	if err != nil || len(rows) != 1 || rows[0].ID != "one" {
		t.Fatalf("rows=%#v err=%v", rows, err)
	}
}
