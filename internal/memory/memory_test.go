package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitIdempotent(t *testing.T) {
	root := t.TempDir()
	created, err := Init(root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(created) == 0 {
		t.Errorf("first init should create files, got %v", created)
	}
	for _, must := range []string{"config.md", "project.md", "architecture.md", "conventions.md"} {
		if _, err := os.Stat(filepath.Join(root, DirName, must)); err != nil {
			t.Errorf("missing scaffold file %s: %v", must, err)
		}
	}

	// Second init must not create anything or clobber edits.
	projectPath := filepath.Join(root, DirName, "project.md")
	if err := os.WriteFile(projectPath, []byte("# My custom project memory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created2, err := Init(root)
	if err != nil {
		t.Fatalf("init 2: %v", err)
	}
	if len(created2) != 0 {
		t.Errorf("second init should create nothing, got %v", created2)
	}
	data, _ := os.ReadFile(projectPath)
	if string(data) != "# My custom project memory\n" {
		t.Errorf("user edit clobbered: %q", data)
	}
}

func TestLoadConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `---
ignore:
  - "**/*_gen.go"
  - "testdata/*"
max-file-size: 2097152
languages: [go, typescript]
---

Free-form notes here.
`
	if err := os.WriteFile(filepath.Join(root, DirName, "config.md"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	c := LoadConfig(root)
	if len(c.Ignore) != 2 || c.Ignore[0] != "**/*_gen.go" {
		t.Errorf("ignore = %v", c.Ignore)
	}
	if c.MaxFileSize != 2097152 {
		t.Errorf("max-file-size = %d", c.MaxFileSize)
	}
	if len(c.Languages) != 2 || c.Languages[1] != "typescript" {
		t.Errorf("languages = %v", c.Languages)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	c := LoadConfig(t.TempDir())
	if c.MaxFileSize != 0 || len(c.Ignore) != 0 {
		t.Errorf("missing config should give zero values: %+v", c)
	}
}

func TestLoadAllAndRender(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	// Add a module file.
	modules := filepath.Join(root, DirName, "modules")
	if err := os.MkdirAll(modules, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modules, "auth.md"), []byte("# Auth\n\nOwns sessions.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := LoadAll(root)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.RelPath)
	}
	// project/architecture/conventions + modules/auth.md; no config.md, no index/.
	for _, want := range []string{"architecture.md", "conventions.md", "modules/auth.md", "project.md"} {
		found := false
		for _, p := range paths {
			if p == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %s in %v", want, paths)
		}
	}
	for _, p := range paths {
		if p == "config.md" || strings.HasPrefix(p, "index/") {
			t.Errorf("unwanted file in memory listing: %s", p)
		}
	}

	out, err := RenderAll(root)
	if err != nil || !strings.Contains(out, "# Auth") {
		t.Errorf("RenderAll missing module content: %v %q", err, out)
	}
}

func TestRenderAllEmpty(t *testing.T) {
	out, err := RenderAll(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cortex init") {
		t.Errorf("empty memory message should mention init: %q", out)
	}
}
