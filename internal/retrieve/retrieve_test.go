package retrieve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cortex/internal/index"
	"cortex/internal/store"
)

func setupRepo(t *testing.T) (*Engine, string) {
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
	mk("auth/server.go", `package auth

import "strings"

// Server serves auth requests.
type Server struct {
	store *Store
}

// Login authenticates a user.
func (s *Server) Login(user string) error {
	tok, err := s.store.Issue(user)
	if err != nil {
		return err
	}
	strings.ToUpper(tok)
	return nil
}

type Store struct{}

func (st *Store) Issue(user string) (string, error) {
	return "tok-" + user, nil
}
`)
	mk("main.go", `package main

import "example/auth"

func main() {
	s := &auth.Server{}
	_ = s.Login("alice")
}
`)
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
	return New(st, root), root
}

func TestSearchFindsSymbolsAndFiles(t *testing.T) {
	e, _ := setupRepo(t)
	out, err := e.Search("login", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Server.Login") {
		t.Errorf("search should find Server.Login:\n%s", out)
	}
	if !strings.Contains(out, "auth/server.go") {
		t.Errorf("search should mention file paths:\n%s", out)
	}

	out, err = e.Search("zzz_no_such_thing_xyz", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No matches") {
		t.Errorf("garbage query should say no matches:\n%s", out)
	}
}

func TestSymbolShowsCallsAndCallers(t *testing.T) {
	e, _ := setupRepo(t)
	out, err := e.Symbol("Server.Login", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "auth/server.go:11") {
		t.Errorf("should show location:\n%s", out)
	}
	if !strings.Contains(out, "Issue") {
		t.Errorf("should list callee Issue:\n%s", out)
	}
	if !strings.Contains(out, "Login authenticates a user.") {
		t.Errorf("should include doc:\n%s", out)
	}

	// main.go calls Login → caller appears.
	out, err = e.Symbol("Login", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Called by") || !strings.Contains(out, "main.go") {
		t.Errorf("should list caller from main.go:\n%s", out)
	}
}

func TestSymbolQualifiedAndMissing(t *testing.T) {
	e, _ := setupRepo(t)
	out, err := e.Symbol("Server.Login", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "```go") {
		t.Errorf("--src should render a go code block:\n%s", out)
	}
	if !strings.Contains(out, "func (s *Server) Login") {
		t.Errorf("source body should be included:\n%s", out)
	}

	out, err = e.Symbol("DoesNotExist", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("missing symbol message:\n%s", out)
	}
}

func TestRefsGroupsByFile(t *testing.T) {
	e, _ := setupRepo(t)
	out, err := e.Refs("Login")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("refs should include main.go call site:\n%s", out)
	}
	if !strings.Contains(out, "call") {
		t.Errorf("refs should include kind:\n%s", out)
	}
}

func TestDepsShowsBothDirections(t *testing.T) {
	e, _ := setupRepo(t)
	out, err := e.Deps("Server.Login")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Depends on") || !strings.Contains(out, "Issue") {
		t.Errorf("deps should list callees:\n%s", out)
	}
	if !strings.Contains(out, "Depended on by") {
		t.Errorf("deps should list callers:\n%s", out)
	}
	if !strings.Contains(out, "strings") {
		t.Errorf("deps should list file imports:\n%s", out)
	}
}
