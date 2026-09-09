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
	claimIDs := map[string]string{}
	var claims []Claim
	for _, f := range files {
		if claim, ok, parseErr := ParseClaim(f.RelPath, f.Content); ok {
			if parseErr != nil {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateInvalid, parseErr.Error()})
				continue
			}
			if previous, exists := claimIDs[claim.ID]; exists {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateInvalid, fmt.Sprintf("duplicate claim id %q (already in %s)", claim.ID, previous)})
				continue
			}
			claimIDs[claim.ID] = f.RelPath
			claims = append(claims, claim)
			continue
		}
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
	for _, claim := range claims {
		if missing := missingRelations(claim, claimIDs); len(missing) > 0 {
			report.Items = append(report.Items, ValidationItem{claim.File, StateInvalid, "missing related claim: " + strings.Join(missing, ", ")})
			continue
		}
		detail := fmt.Sprintf("claim %s (%s), status %s", claim.ID, claim.Type, claim.Status)
		if claim.Confidence != nil {
			detail += fmt.Sprintf(", confidence %.2f", *claim.Confidence)
		}
		report.Items = append(report.Items, ValidationItem{claim.File, StateCurrent, detail})
	}
	for _, f := range files {
		if work, ok, parseErr := ParseWorkRecord(f.RelPath, f.Content); ok {
			if parseErr != nil {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateInvalid, parseErr.Error()})
			} else {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateCurrent, fmt.Sprintf("work %s, status %s", work.ID, work.Status)})
			}
		}
		if pref, ok, parseErr := ParsePreference(f.RelPath, f.Content); ok {
			if parseErr != nil {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateInvalid, parseErr.Error()})
			} else {
				report.Items = append(report.Items, ValidationItem{f.RelPath, StateCurrent, fmt.Sprintf("preference %s, scope %s, confidence %.2f", pref.ID, pref.Scope, pref.Confidence)})
			}
		}
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
