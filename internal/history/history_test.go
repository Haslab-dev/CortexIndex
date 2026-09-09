package history

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHistoryRealRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		} else {
			return string(out)
		}
		return ""
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test User")
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("auth.go", "package auth\n\nfunc Login() {}\n")
	run("add", "auth.go")
	run("commit", "-q", "-m", "Add login flow")
	write("auth.go", "package auth\n\nfunc Login() {}\nfunc Logout() {}\n")
	run("add", "auth.go")
	run("commit", "-q", "-m", "Add logout flow")

	client := New(root)
	commits, err := client.Log(context.Background(), LogOptions{Limit: 1})
	if err != nil || len(commits) != 1 || commits[0].Subject != "Add logout flow" {
		t.Fatalf("Log() = %#v, %v", commits, err)
	}
	commits, err = client.Log(context.Background(), LogOptions{Query: "login", File: "auth.go"})
	if err != nil || len(commits) != 1 || commits[0].Subject != "Add login flow" {
		t.Fatalf("filtered Log() = %#v, %v", commits, err)
	}
	ref := commits[0].Hash
	detail, err := client.Show(context.Background(), ref, ShowOptions{})
	if err != nil || detail.Subject != "Add login flow" || len(detail.Files) != 1 || detail.Files[0].Path != "auth.go" {
		t.Fatalf("Show() = %#v, %v", detail, err)
	}
	if strings.Contains(detail.Diff, "secret") {
		t.Fatal("unexpected diff in non-diff Show")
	}
}

func TestHistoryRejectsNonRepositoryAndBadRef(t *testing.T) {
	client := New(t.TempDir())
	if _, err := client.Log(context.Background(), LogOptions{}); err == nil || !strings.Contains(err.Error(), "not a Git repository") {
		t.Fatalf("expected non-repository error, got %v", err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	client = New(root)
	if _, err := client.Show(context.Background(), "--bad", ShowOptions{}); err == nil || !strings.Contains(err.Error(), "invalid commit reference") {
		t.Fatalf("expected bad ref error, got %v", err)
	}
}
