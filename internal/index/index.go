// Package index walks a repository and keeps the derived SQLite index in
// sync: full builds for `cortex index`, hash-diff incremental updates for
// `cortex update` and the `watch` loop.
package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cortex/internal/lang"
	"cortex/internal/store"
)

// Options controls an indexing run.
type Options struct {
	Root       string // repo root (absolute)
	ExtraIgnores [] string // from config.md
	MaxFileSize int64  // skip files larger than this
}

// Stats summarizes one indexing run.
type Stats struct {
	Indexed  int // files (re)parsed+stored
	Skipped  int // unchanged files skipped
	Deleted  int // removed from index
	Failed   int // read errors
	Symbols  int
	Files    int // total files in index after run
	Duration time.Duration
}

const defaultMaxFileSize = 1 << 20 // 1 MiB

// defaultIgnores lists directory names and file patterns never indexed.
var defaultIgnoreDirs = map[string]bool{
	".git": true, ".cortex": true, ".hg": true, ".svn": true,
	"node_modules": true, "vendor": true, "bower_components": true,
	"dist": true, "build": true, "out": true, "target": true,
	".next": true, ".nuxt": true, ".turbo": true, ".cache": true,
	".venv": true, "venv": true, "__pycache__": true, ".mypy_cache": true,
	".pytest_cache": true, "coverage": true, ".idea": true, ".vscode": true,
	"Pods": true, "DerivedData": true, ".gradle": true, ".terraform": true,
}

var defaultIgnoreSuffixes = []string{
	".min.js", ".min.css", ".bundle.js", ".chunk.js",
	".lock", ".sum", ".pb.go", "_pb2.py", ".generated.cs",
	".ico", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".pdf",
	".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
	".woff", ".woff2", ".ttf", ".otf", ".eot",
	".mp3", ".mp4", ".mov", ".avi", ".webm",
	".exe", ".dll", ".so", ".dylib", ".a", ".o", ".obj",
	".class", ".jar", ".pyc", ".pyo", ".wasm", ".bin", ".dat",
	".env", ".env.local", ".pem", ".key", ".p12", ".crt",
	".DS_Store", ".sqlite", ".db",
}

// shouldIndex reports whether a repo-relative path passes ignore rules.
func (o Options) shouldIndex(relPath string, isDir bool) bool {
	if isDir {
		return true
	}
	base := filepath.Base(relPath)
	lower := strings.ToLower(base)
	for _, suf := range defaultIgnoreSuffixes {
		if strings.HasSuffix(lower, suf) {
			return false
		}
	}
	for _, pat := range o.ExtraIgnores {
		if ok, _ := filepath.Match(pat, relPath); ok {
			return false
		}
		if ok, _ := filepath.Match(pat, base); ok {
			return false
		}
	}
	return true
}

// hashFile computes sha256 of a file's contents.
func hashFile(path string) (string, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data)), nil
}

// walkFiles returns all indexable files under root (repo-relative, /-separated).
func walkFiles(root string, o Options) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: skip silently, keep walking
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if defaultIgnoreDirs[d.Name()] && rel != "." {
				return filepath.SkipDir
			}
			for _, pat := range o.ExtraIgnores {
				if ok, _ := filepath.Match(pat, rel); ok {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if o.shouldIndex(rel, false) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// Full rebuilds the entire index from scratch (PRD: `cortex index`).
func Full(o Options, st *store.Store) (*Stats, error) {
	start := time.Now()
	if err := st.Wipe(); err != nil {
		return nil, err
	}
	stats, err := scan(o, st, nil)
	if err != nil {
		return nil, err
	}
	stats.Duration = time.Since(start)
	return stats, nil
}

// Incremental updates only changed/new/deleted files (PRD: `cortex update`).
// known may be nil to load from the store.
func Incremental(o Options, st *store.Store) (*Stats, error) {
	start := time.Now()
	known, err := st.AllFilePaths()
	if err != nil {
		return nil, err
	}
	stats, err := scan(o, st, known)
	if err != nil {
		return nil, err
	}
	stats.Duration = time.Since(start)
	return stats, nil
}

// scan walks the repo and applies changes; known == nil means full rebuild.
func scan(o Options, st *store.Store, known map[string]string) (*Stats, error) {
	maxSize := o.MaxFileSize
	if maxSize <= 0 {
		maxSize = defaultMaxFileSize
	}
	paths, err := walkFiles(o.Root, o)
	if err != nil {
		return nil, err
	}
	stats := &Stats{}
	seen := map[string]bool{}

	for _, rel := range paths {
		abs := filepath.Join(o.Root, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil {
			stats.Failed++
			continue
		}
		if info.Size() > maxSize {
			continue
		}
		seen[rel] = true
		hash, _, err := hashFile(abs)
		if err != nil {
			stats.Failed++
			continue
		}
		if known != nil {
			if h, ok := known[rel]; ok && h == hash {
				stats.Skipped++
				continue
			}
		}
		if err := indexOne(o.Root, rel, abs, info, hash, st); err != nil {
			stats.Failed++
			continue
		}
		stats.Indexed++
	}

	// Deletion sweep: indexed paths no longer on disk.
	for p := range known {
		if !seen[p] {
			if err := st.RemoveFile(p); err != nil {
				return stats, err
			}
			stats.Deleted++
		}
	}

	files, symbols, _, _, err := st.Stats()
	if err != nil {
		return stats, err
	}
	stats.Files = files
	stats.Symbols = symbols
	return stats, nil
}

// indexOne parses (or content-indexes) one file and stores it.
func indexOne(root, rel, abs string, info os.FileInfo, hash string, st *store.Store) error {
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	content := string(data)
	f := store.File{
		Path:     rel,
		Language: "",
		Size:     info.Size(),
		Hash:     hash,
		MTime:    info.ModTime().UnixNano(),
		Parsed:   false,
	}

	var res lang.Result
	if lg := lang.Detect(rel); lg != nil {
		f.Language = lg.Name
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Parser panics must never take down indexing (PRD §27).
					res = lang.Result{}
					f.Parsed = false
				}
			}()
			res = lg.Extract(data)
			f.Parsed = true
		}()
	} else if lang.IsTextExt(rel) {
		f.Language = strings.TrimPrefix(filepath.Ext(rel), ".")
	} else {
		// Unknown extension: index as text if it decodes cleanly, else skip content.
		f.Language = "text"
	}

	return st.ReplaceFile(f, content, res)
}

// FormatStats renders an indexing run as agent-readable Markdown.
func FormatStats(kind string, s *Stats) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Index %s\n\n", kind))
	b.WriteString(fmt.Sprintf("- Files indexed: %d\n", s.Indexed))
	b.WriteString(fmt.Sprintf("- Files skipped (unchanged): %d\n", s.Skipped))
	b.WriteString(fmt.Sprintf("- Files removed: %d\n", s.Deleted))
	if s.Failed > 0 {
		b.WriteString(fmt.Sprintf("- Read/parse failures: %d\n", s.Failed))
	}
	b.WriteString(fmt.Sprintf("- Total files in index: %d\n", s.Files))
	b.WriteString(fmt.Sprintf("- Total symbols: %d\n", s.Symbols))
	b.WriteString(fmt.Sprintf("- Duration: %s\n", s.Duration.Round(time.Millisecond)))
	return b.String()
}
