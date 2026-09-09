package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildExcludesProposedClaims(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cortex", "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := func(id, status, text string) string {
		return "---\nid: " + id + "\ntype: decision\nstatus: " + status + "\nupdated_at: 2026-09-09\n---\n" + text + "\n"
	}
	if err := os.WriteFile(filepath.Join(root, ".cortex", "decisions", "active.md"), []byte(content("active-rule", "active", "Active authentication boundary.")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cortex", "decisions", "proposed.md"), []byte(content("proposed-rule", "proposed", "Secret proposed rule that must not enter context.")), 0o644); err != nil {
		t.Fatal(err)
	}
	out := Build("authentication boundary", Options{Root: root, HasIndex: false})
	if !strings.Contains(out, "active-rule") || !strings.Contains(out, "Active authentication boundary") {
		t.Fatalf("active claim missing:\n%s", out)
	}
	if strings.Contains(out, "proposed-rule") || strings.Contains(out, "Secret proposed rule") {
		t.Fatalf("proposed claim leaked into context:\n%s", out)
	}
}
