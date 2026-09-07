package index

import (
	"os"
	"path/filepath"
	"testing"

	"cortex/internal/store"
)

// fixtureRepo writes a small multi-file repo into a temp dir.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mk := func(rel, content string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("auth/server.go", "package auth\n\ntype Server struct{}\n\nfunc (s *Server) Login(u string) error { return store.Save(u) }\n")
	mk("auth/store.go", "package auth\n\nfunc Save(u string) error { return nil }\n")
	mk("main.go", "package main\n\nfunc main() { println(\"hi\") }\n")
	mk("README.md", "# Fixture\n\nThis is a fixture repo about authentication.\n")
	mk("node_modules/junk.js", "function junk() {}")
	mk(".git/HEAD", "ref: refs/heads/main")
	return root
}

func openIndexStore(t *testing.T, root string) (*store.Store, string) {
	t.Helper()
	dir := filepath.Join(root, ".cortex", "index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "codebase.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st, dir
}

func TestFullThenIncrementalSkip(t *testing.T) {
	root := fixtureRepo(t)
	st, _ := openIndexStore(t, root)
	o := Options{Root: root}

	s1, err := Full(o, st)
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	if s1.Indexed != 4 { // 3 go files + README; node_modules and .git ignored
		t.Errorf("full indexed = %d, want 4", s1.Indexed)
	}
	if s1.Symbols == 0 {
		t.Errorf("expected symbols from go files")
	}

	s2, err := Incremental(o, st)
	if err != nil {
		t.Fatalf("incremental: %v", err)
	}
	if s2.Indexed != 0 || s2.Skipped != 4 || s2.Deleted != 0 {
		t.Errorf("incremental with no changes: %+v", s2)
	}
}

func TestIncrementalEditAndDelete(t *testing.T) {
	root := fixtureRepo(t)
	st, _ := openIndexStore(t, root)
	o := Options{Root: root}
	if _, err := Full(o, st); err != nil {
		t.Fatal(err)
	}

	// Edit one file: only it re-indexes.
	if err := os.WriteFile(filepath.Join(root, "auth/store.go"),
		[]byte("package auth\n\n// Save persists a user.\nfunc Save(u string) error { return audit(u) }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Incremental(o, st)
	if err != nil {
		t.Fatal(err)
	}
	if s.Indexed != 1 || s.Skipped != 3 || s.Deleted != 0 {
		t.Errorf("after edit: %+v", s)
	}
	rows, _ := st.SearchSymbolsFTS(`"persists"`, store.QueryParams{})
	if len(rows) != 1 || rows[0].Name != "Save" {
		t.Errorf("edited symbol not found: %+v", rows)
	}

	// Delete one file: sweep removes its rows.
	if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
		t.Fatal(err)
	}
	s, err = Incremental(o, st)
	if err != nil {
		t.Fatal(err)
	}
	if s.Deleted != 1 {
		t.Errorf("deleted = %d, want 1: %+v", s.Deleted, s)
	}
	files, _, _, _, _ := st.Stats()
	if files != 3 {
		t.Errorf("files after delete = %d, want 3", files)
	}
	if hits, _ := st.SearchFilesFTS(`"hi"`, store.QueryParams{}); len(hits) != 0 {
		t.Errorf("deleted file still searchable: %+v", hits)
	}
}

func TestNewFileAppears(t *testing.T) {
	root := fixtureRepo(t)
	st, _ := openIndexStore(t, root)
	o := Options{Root: root}
	if _, err := Full(o, st); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth/token.go"),
		[]byte("package auth\n\ntype TokenStore struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Incremental(o, st)
	if err != nil {
		t.Fatal(err)
	}
	if s.Indexed != 1 || s.Skipped != 4 {
		t.Errorf("after new file: %+v", s)
	}
	rows, _ := st.SearchSymbolsExact("TokenStore", store.QueryParams{})
	if len(rows) != 1 {
		t.Errorf("new symbol not indexed: %+v", rows)
	}
}

func TestIgnoreRules(t *testing.T) {
	root := fixtureRepo(t)
	st, _ := openIndexStore(t, root)
	o := Options{Root: root, ExtraIgnores: []string{"README.md"}}
	if _, err := Full(o, st); err != nil {
		t.Fatal(err)
	}
	hits, _ := st.SearchFilesFTS(`"fixture"`, store.QueryParams{})
	if len(hits) != 0 {
		t.Errorf("extra ignore not honored: %+v", hits)
	}
}

func TestParseFailureFallsBackToFTS(t *testing.T) {
	root := t.TempDir()
	mk := filepath.Join(root, "broken.go")
	if err := os.WriteFile(mk, []byte("this is ))) not go code <<< at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, _ := openIndexStore(t, root)
	o := Options{Root: root}
	s, err := Full(o, st)
	if err != nil {
		t.Fatalf("full with broken file: %v", err)
	}
	if s.Indexed != 1 {
		t.Errorf("broken file should still be indexed (FTS-only), got %+v", s)
	}
	// Content is searchable even though parsing produced no symbols.
	hits, err := st.SearchFilesFTS(`"code"`, store.QueryParams{})
	if err != nil || len(hits) != 1 {
		t.Errorf("broken file content not searchable: %+v err=%v", hits, err)
	}
}
