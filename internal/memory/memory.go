// Package memory owns the durable, human-readable Markdown knowledge under
// `.cortex/`: the init scaffold, config parsing, and the reader used by the
// context engine. Markdown is the memory; SQLite is derived state (PRD §31).
package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirName is the hidden project directory created inside a repository.
const DirName = ".cortex"

// MemoryPaths bundles the well-known files of the memory directory.
type MemoryPaths struct {
	Root string // repo root (absolute)
}

// Dir returns the absolute path of the .cortex directory.
func (m MemoryPaths) Dir() string { return filepath.Join(m.Root, DirName) }

// IndexPath returns the database location.
func (m MemoryPaths) IndexPath() string {
	return filepath.Join(m.Dir(), "index", "codebase.db")
}

// Scaffold describes every file `cortex init` ensures exists.
type Scaffold struct {
	RelPath string
	Content string
}

// ScaffoldFiles returns the full `cortex init` template set.
func ScaffoldFiles() []Scaffold {
	return []Scaffold{
		{RelPath: "config.md", Content: configTemplate},
		{RelPath: "project.md", Content: projectTemplate},
		{RelPath: "architecture.md", Content: architectureTemplate},
		{RelPath: "conventions.md", Content: conventionsTemplate},
		{RelPath: filepath.Join("decisions", ".keep"), Content: ""},
		{RelPath: filepath.Join("modules", ".keep"), Content: ""},
		{RelPath: filepath.Join("index", ".gitignore"), Content: "*\n!.gitignore\n"},
	}
}

// Init creates the memory scaffold, never overwriting user content
// (idempotent; PRD §24). Returns the list of files it created.
func Init(root string) ([]string, error) {
	m := MemoryPaths{Root: root}
	var created []string
	for _, f := range ScaffoldFiles() {
		abs := filepath.Join(m.Dir(), filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return created, err
		}
		if _, err := os.Stat(abs); err == nil {
			continue
		}
		if err := os.WriteFile(abs, []byte(f.Content), 0o644); err != nil {
			return created, err
		}
		created = append(created, f.RelPath)
	}
	return created, nil
}

// Exists reports whether the repo has a .cortex directory.
func Exists(root string) bool {
	info, err := os.Stat(filepath.Join(root, DirName))
	return err == nil && info.IsDir()
}

// Config holds the parsed options from config.md frontmatter.
type Config struct {
	Ignore      []string
	MaxFileSize int64
	Languages   []string

	ignoreOpen bool // parser state: inside an "ignore:" list block
}

// LoadConfig parses the minimal frontmatter block of .cortex/config.md:
//
//	---
//	ignore:
//	  - "**/*_gen.go"
//	max-file-size: 2097152
//	languages: [go, typescript]
//	---
//
// Unknown keys are ignored; the rest of the file is free-form notes.
func LoadConfig(root string) Config {
	c := Config{MaxFileSize: 0} // 0 = indexer default
	data, err := os.ReadFile(filepath.Join(root, DirName, "config.md"))
	if err != nil {
		return c
	}
	text := string(data)
	if !strings.HasPrefix(text, "---") {
		return c
	}
	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return c
	}
	block := text[3 : end+3]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "- ") && c.ignoreOpen:
			c.Ignore = append(c.Ignore, strings.Trim(strings.TrimPrefix(line, "- "), `"`))
		case strings.HasPrefix(line, "ignore:"):
			c.ignoreOpen = true
		case strings.HasPrefix(line, "languages:"):
			c.ignoreOpen = false
			raw := strings.Trim(strings.TrimPrefix(line, "languages:"), " []")
			for _, l := range strings.Split(raw, ",") {
				if l = strings.TrimSpace(l); l != "" {
					c.Languages = append(c.Languages, l)
				}
			}
		case strings.HasPrefix(line, "max-file-size:"):
			c.ignoreOpen = false
			var n int64
			if _, err := fmtSscan(strings.TrimPrefix(line, "max-file-size:"), &n); err == nil {
				c.MaxFileSize = n
			}
		default:
			if line != "" && !strings.HasPrefix(line, "- ") {
				c.ignoreOpen = false
			}
		}
	}
	return c
}

// MemoryFile is one Markdown knowledge file with its relative path.
type MemoryFile struct {
	RelPath string // e.g. "modules/auth.md"
	Content string
}

// LoadAll returns every Markdown file under .cortex (excluding config and
// the index dir), sorted with top-level files first.
func LoadAll(root string) ([]MemoryFile, error) {
	dir := filepath.Join(root, DirName)
	var out []MemoryFile
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint: nil is fine; unreadable entries are skipped
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "config.md" || strings.HasPrefix(rel, "index/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		out = append(out, MemoryFile{RelPath: rel, Content: string(data)})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out, nil
}

// RenderAll prints the whole memory tree for `cortex memory`.
func RenderAll(root string) (string, error) {
	files, err := LoadAll(root)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if len(files) == 0 {
		return "No memory files found. Run `cortex init` first.\n", nil
	}
	for _, f := range files {
		b.WriteString("## " + f.RelPath + "\n\n")
		b.WriteString(strings.TrimRight(f.Content, "\n"))
		b.WriteString("\n\n")
	}
	return b.String(), nil
}
