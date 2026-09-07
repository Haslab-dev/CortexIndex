package lexsearch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanFindsKeywordLines(t *testing.T) {
	root := t.TempDir()
	mk := func(rel, content string) {
		p := filepath.Join(root, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o644)
	}
	mk("auth/server.go", "package auth\n\nfunc Login(user string) error {\n\treturn saveToken(user)\n}\n")
	mk("notes.md", "The auth flow uses saveToken.\n")
	mk("node_modules/x.js", "saveToken everywhere")
	mk("skipme.png", "\x00\x01binary")

	hits, err := Scan(root, []string{"savetoken"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Path != "auth/server.go" || hits[0].Line != 4 {
		t.Errorf("top hit = %+v", hits[0])
	}
	if !strings.Contains(hits[1].Path, "notes.md") {
		t.Errorf("second hit = %+v", hits[1])
	}
}

func TestScanEmptyRepo(t *testing.T) {
	hits, err := Scan(t.TempDir(), []string{"anything"}, 5)
	if err != nil || len(hits) != 0 {
		t.Errorf("empty repo should yield no hits: %+v %v", hits, err)
	}
}
