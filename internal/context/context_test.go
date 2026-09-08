package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cortex/internal/index"
	"cortex/internal/store"
)

func setupRepo(t *testing.T, withIndex bool) string {
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
	mk(".cortex/project.md", "# Project\n\nAn authentication platform for payments.\n")
	mk(".cortex/conventions.md", "# Conventions\n\n- API calls belong inside services.\n- Components should not call APIClient directly.\n")
	mk("auth/server.go", `package auth

// Server serves auth requests.
type Server struct {
	store *TokenStore
}

// Login authenticates a user with the token store.
func (s *Server) Login(user string) error {
	return s.store.Issue(user)
}

// TokenStore persists tokens.
type TokenStore struct{}

// Issue creates and saves a token.
func (t *TokenStore) Issue(user string) error {
	return save(user)
}

func save(user string) error { return nil }
`)
	mk("main.go", `package main

import "example/auth"

func main() {
	s := &auth.Server{}
	_ = s.Login("alice")
}
`)
	if withIndex {
		dbPath := filepath.Join(root, ".cortex", "index", "codebase.db")
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			t.Fatal(err)
		}
		st, err := store.Open(dbPath)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		if _, err := index.Full(index.Options{Root: root}, st); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func openStore(t *testing.T, root string) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(root, ".cortex", "index", "codebase.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestBuildContextWithIndex(t *testing.T) {
	root := setupRepo(t, true)
	st := openStore(t, root)
	out := Build("How does authentication work?", Options{
		Root: root, HasIndex: true, Store: st,
	})
	for _, want := range []string{
		"# Codebase Context",
		"Task: How does authentication work?",
		"Project Memory",         // Layer 1: memory mentions authentication
		"Relevant Symbols",       // Layer 2
		"Server.Login",           // ranked symbol
		"Source",                 // Layer 5
		"func (s *Server) Login", // actual source body
		"Relevant Files",         // file list
		"tokens estimated",       // budget footer
	} {
		if !strings.Contains(out, want) {
			t.Errorf("context output missing %q:\n%s", want, out)
		}
	}
}

func TestContextRanksLoginAboveSave(t *testing.T) {
	root := setupRepo(t, true)
	st := openStore(t, root)
	out := Build("How does authentication work?", Options{
		Root: root, HasIndex: true, Store: st,
	})
	loginIdx := strings.Index(out, "Server.Login")
	if loginIdx < 0 {
		t.Fatalf("Login missing:\n%s", out)
	}
	// The generic `save` helper must not outrank auth symbols; both appear
	// (or save doesn't appear at all), but Login must come first.
	if saveIdx := strings.Index(out, "\nsave ("); saveIdx >= 0 && saveIdx < loginIdx {
		t.Errorf("save ranked above Login:\n%s", out)
	}
}

func TestContextWithoutIndexFallsBack(t *testing.T) {
	root := setupRepo(t, false)
	out := Build("token store", Options{Root: root, HasIndex: false})
	if !strings.Contains(out, "Lexical Scan") {
		t.Errorf("fallback should label itself:\n%s", out)
	}
	if !strings.Contains(out, "server.go") {
		t.Errorf("fallback should find auth/server.go:\n%s", out)
	}
	if !strings.Contains(out, "cortex index") {
		t.Errorf("fallback should advise indexing:\n%s", out)
	}
}

func TestContextRespectsBudget(t *testing.T) {
	root := setupRepo(t, true)
	st := openStore(t, root)
	budget := 300
	out := Build("authentication", Options{
		Root: root, HasIndex: true, Store: st, Budget: budget,
	})
	if estTokenLen(out) > budget+120 { // small tolerance for the footer
		t.Errorf("output exceeds budget: est=%d budget=%d\n%s", estTokenLen(out), budget, out)
	}
}

func TestKeywordsExtraction(t *testing.T) {
	kw := Keywords("How does the AuthService login flow work?")
	joined := strings.Join(kw, ",")
	if !strings.Contains(joined, "authservice") || !strings.Contains(joined, "auth") ||
		!strings.Contains(joined, "service") || !strings.Contains(joined, "login") || !strings.Contains(joined, "flow") {
		t.Errorf("keywords = %v", kw)
	}
	for _, stop := range []string{"how", "does", "the", "work"} {
		if strings.Contains(joined, stop) {
			t.Errorf("stopword %q leaked into %v", stop, kw)
		}
	}
}
