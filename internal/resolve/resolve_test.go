package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoCandidate(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	os.MkdirAll(dist, 0755)
	bin := filepath.Join(dist, "cortex")
	os.WriteFile(bin, []byte("x"), 0755)
	got, err := Find("", root, "", "", "")
	if err != nil || got != bin {
		t.Fatalf("got %q err=%v want %q", got, err, bin)
	}
}
func TestFindExplicitAndMissing(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "cortex")
	os.WriteFile(bin, []byte("x"), 0755)
	got, err := Find(bin, "/nowhere", "", "", "")
	if err != nil || got != bin {
		t.Fatalf("explicit got %q err=%v", got, err)
	}
	if _, err := Find("", t.TempDir(), "", "", ""); err == nil {
		t.Fatal("missing binary should fail")
	}
}
func TestShellSnippet(t *testing.T) {
	if s := ShellSnippet(); len(s) == 0 {
		t.Fatal("empty snippet")
	}
}
