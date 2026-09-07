// Package lexsearch implements the no-index fallback: a plain lexical scan
// over repository text files (PRD §27 — the agent must never be blocked by
// a missing or broken index).
package lexsearch

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Hit is one matching line.
type Hit struct {
	Path string
	Line int
	Text string
}

const (
	maxFileSize   = 512 << 10 // 512 KiB per file in fallback mode
	maxCandidates = 4000
)

func isSearchable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case "", ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".kt",
		".c", ".h", ".cpp", ".cc", ".hpp", ".md", ".txt", ".yaml", ".yml",
		".toml", ".json", ".sql", ".sh", ".rb", ".php", ".swift", ".m", ".mm",
		".proto", ".css", ".html", ".xml", ".kts", ".scala", ".dart":
		return true
	}
	return false
}

func isSkipDir(name string) bool {
	switch name {
	case ".git", ".cortex", "node_modules", "vendor", "dist", "build", "out",
		"target", "__pycache__", ".venv", "venv", ".idea", ".vscode":
		return true
	}
	return false
}

// Scan greps the repo for all keywords (OR semantics), returning the most
// keyword-dense lines.
func Scan(root string, keywords []string, limit int) ([]Hit, error) {
	if limit <= 0 {
		limit = 10
	}
	type hitScore struct {
		hit   Hit
		score int
	}
	var candidates []hitScore

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSearchable(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileSize {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		lines := strings.Split(string(data), "\n")
		for i, ln := range lines {
			low := strings.ToLower(ln)
			score := 0
			for _, kw := range keywords {
				if strings.Contains(low, kw) {
					score++
				}
			}
			if score > 0 {
				candidates = append(candidates, hitScore{
					hit:   Hit{Path: rel, Line: i + 1, Text: strings.TrimSpace(ln)},
					score: score,
				})
				if len(candidates) > maxCandidates {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].hit.Path != candidates[j].hit.Path {
			return candidates[i].hit.Path < candidates[j].hit.Path
		}
		return candidates[i].hit.Line < candidates[j].hit.Line
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]Hit, len(candidates))
	for i, c := range candidates {
		out[i] = c.hit
	}
	return out, nil
}
