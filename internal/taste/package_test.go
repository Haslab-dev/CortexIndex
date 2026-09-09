package taste

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPackageAndDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "preferences"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ncortex: taste-package\nformat: 1\nid: team-style\nname: Team Style\nversion: 1.0.0\nprovider: taste\nupdated_at: 2026-09-09\n---\n\nTeam preferences.\n"
	pref := "---\ncortex: preference\nid: table-tests\nkey: testing.style\nscope: team\nstatus: active\nconfidence: 0.8\nupdated_at: 2026-09-09\n---\n\nPrefer table-driven tests.\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "preferences", "table-tests.md"), []byte(pref), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := LoadPackage(root)
	if err != nil || p.Manifest.ID != "team-style" || len(p.Records) != 1 || p.Records[0].Status != "proposed" || p.Digest == "" {
		t.Fatalf("p=%#v err=%v", p, err)
	}
}

func TestLoadPackageRejectsUnsafeID(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "preferences"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "---\ncortex: taste-package\nformat: 1\nid: ../bad\nupdated_at: 2026-09-09\n---\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.md"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPackage(root); err == nil {
		t.Fatal("expected unsafe ID error")
	}
}
