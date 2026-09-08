package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MemoryState string

const (
	StateCurrent   MemoryState = "current"
	StateStale     MemoryState = "stale"
	StateOrphaned  MemoryState = "orphaned"
	StateUntracked MemoryState = "untracked"
	StateInvalid   MemoryState = "invalid"
)

type ValidationItem struct {
	Path   string
	State  MemoryState
	Detail string
}

type ValidationReport struct{ Items []ValidationItem }

func (r ValidationReport) Count(state MemoryState) int {
	n := 0
	for _, i := range r.Items {
		if i.State == state {
			n++
		}
	}
	return n
}

func Validate(root string) (ValidationReport, error) {
	files, err := LoadAll(root)
	if err != nil {
		return ValidationReport{}, err
	}
	var report ValidationReport
	for _, f := range files {
		meta, ok, invalid := parseGeneratedMeta(f.Content)
		if invalid {
			report.Items = append(report.Items, ValidationItem{f.RelPath, StateInvalid, "malformed generated frontmatter"})
			continue
		}
		if !ok {
			report.Items = append(report.Items, ValidationItem{f.RelPath, StateUntracked, "no machine-readable provenance"})
			continue
		}
		state, detail := checkMeta(root, meta)
		report.Items = append(report.Items, ValidationItem{f.RelPath, state, detail})
	}
	return report, nil
}

type generatedMeta struct {
	Sources []string
	Hash    string
	Status  string
}

func parseGeneratedMeta(content string) (generatedMeta, bool, bool) {
	if !strings.HasPrefix(content, "---\n") {
		return generatedMeta{}, false, false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return generatedMeta{}, false, true
	}
	block := content[4 : end+4]
	if !strings.Contains(block, "cortex: generated") {
		return generatedMeta{}, false, false
	}
	var m generatedMeta
	inSource := false
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "source:":
			inSource = true
		case strings.HasPrefix(line, "- ") && inSource:
			m.Sources = append(m.Sources, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		case strings.HasPrefix(line, "source_hash:"):
			m.Hash = strings.TrimSpace(strings.TrimPrefix(line, "source_hash:"))
			inSource = false
		case strings.HasPrefix(line, "status:"):
			m.Status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			inSource = false
		case line != "" && !strings.HasPrefix(line, "-"):
			inSource = false
		}
	}
	if len(m.Sources) == 0 || m.Hash == "" {
		return generatedMeta{}, false, true
	}
	return m, true, false
}

func checkMeta(root string, m generatedMeta) (MemoryState, string) {
	var manifest strings.Builder
	for _, rel := range m.Sources {
		clean := filepath.Clean(rel)
		if filepath.IsAbs(rel) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return StateInvalid, "source path escapes repository"
		}
		path := filepath.Join(root, clean)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return StateOrphaned, "missing source: " + rel
		}
		if err != nil {
			return StateInvalid, "cannot read source: " + rel
		}
		sum := sha256.Sum256(data)
		manifest.WriteString(rel + "\x00" + hex.EncodeToString(sum[:]) + "\n")
	}
	sum := sha256.Sum256([]byte(manifest.String()))
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if actual != m.Hash {
		return StateStale, fmt.Sprintf("source hash changed (recorded %s, current %s)", m.Hash, actual)
	}
	return StateCurrent, "source hashes match"
}

func FormatValidation(r ValidationReport) string {
	var b strings.Builder
	b.WriteString("# Memory Validation\n\n")
	for _, state := range []MemoryState{StateCurrent, StateStale, StateOrphaned, StateUntracked, StateInvalid} {
		b.WriteString(fmt.Sprintf("- %s: %d\n", state, r.Count(state)))
	}
	b.WriteString("\n")
	for _, i := range r.Items {
		if i.State != StateCurrent {
			b.WriteString(fmt.Sprintf("- `%s` — **%s**: %s\n", i.Path, i.State, i.Detail))
		}
	}
	return b.String()
}
