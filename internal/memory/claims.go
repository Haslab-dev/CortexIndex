package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Claim is one explicit, human-readable memory record. Markdown remains the
// canonical store; claims add typed metadata without hiding the body.
type Claim struct {
	File        string
	ID          string
	Type        string
	Scope       string
	Text        string
	Confidence  *float64
	Status      string
	UpdatedAt   string
	Sources     []string
	Evidence    []Evidence
	Supersedes  []string
	Contradicts []string
}

// Evidence identifies the repository material supporting a claim.
type Evidence struct {
	Raw       string
	Path      string
	StartLine int
	EndLine   int
}

var claimTypes = map[string]bool{
	"fact": true, "decision": true, "constraint": true, "architecture": true,
	"module": true, "convention": true, "behavior": true, "preference": true,
}

var claimScopes = map[string]bool{
	"repository": true, "team": true, "user": true, "task": true,
}

var claimStatuses = map[string]bool{
	"active": true, "deprecated": true, "superseded": true, "proposed": true,
}

// ParseClaim parses the intentionally small claim frontmatter format. The
// boolean is false for ordinary legacy Markdown and generated bootstrap files.
func ParseClaim(relPath, content string) (Claim, bool, error) {
	block, body, ok := frontmatter(content)
	if !ok {
		return Claim{}, false, nil
	}
	recognized := hasClaimMarker(block)
	if !recognized {
		return Claim{}, false, nil
	}
	c := Claim{File: filepath.ToSlash(relPath), Scope: "repository", Status: "active"}
	lists := map[string]*[]string{
		"source": &c.Sources, "sources": &c.Sources,
		"supersedes": &c.Supersedes, "contradicts": &c.Contradicts,
	}
	var list *[]string
	evidenceOpen := false
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			item := cleanScalar(strings.TrimPrefix(line, "- "))
			if evidenceOpen {
				e, err := parseEvidence(item)
				if err != nil {
					return c, true, fmt.Errorf("%s: %w", relPath, err)
				}
				c.Evidence = append(c.Evidence, e)
				continue
			}
			if list != nil {
				*list = append(*list, item)
				continue
			}
			return c, true, fmt.Errorf("%s: list item without a list field", relPath)
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return c, true, fmt.Errorf("%s: malformed frontmatter line %q", relPath, line)
		}
		key := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		list = nil
		evidenceOpen = false
		switch key {
		case "id":
			c.ID = cleanScalar(value)
		case "type":
			c.Type = strings.ToLower(cleanScalar(value))
		case "scope":
			c.Scope = strings.ToLower(cleanScalar(value))
		case "status":
			c.Status = strings.ToLower(cleanScalar(value))
		case "updated_at":
			c.UpdatedAt = cleanScalar(value)
		case "confidence":
			if value == "" {
				return c, true, fmt.Errorf("%s: confidence is empty", relPath)
			}
			n, err := strconv.ParseFloat(cleanScalar(value), 64)
			if err != nil || n < 0 || n > 1 {
				return c, true, fmt.Errorf("%s: confidence must be between 0 and 1", relPath)
			}
			c.Confidence = &n
		case "evidence":
			if value != "" {
				for _, item := range parseInlineList(value) {
					e, err := parseEvidence(item)
					if err != nil {
						return c, true, fmt.Errorf("%s: %w", relPath, err)
					}
					c.Evidence = append(c.Evidence, e)
				}
			} else {
				// Evidence is represented separately because it may carry line ranges.
				evidenceOpen = true
			}
		case "source", "sources", "supersedes", "contradicts":
			if value != "" {
				for _, item := range parseInlineList(value) {
					if key == "source" || key == "sources" {
						c.Sources = append(c.Sources, item)
					} else {
						*lists[key] = append(*lists[key], item)
					}
				}
			} else {
				list = lists[key]
			}
		case "cortex":
			if cleanScalar(value) != "claim" {
				return c, true, fmt.Errorf("%s: cortex marker must be claim", relPath)
			}
		default:
			// Unknown fields are intentionally ignored for forward compatibility.
		}
	}
	c.Text = strings.TrimSpace(body)
	if err := validateClaim(c); err != nil {
		return c, true, fmt.Errorf("%s: %w", relPath, err)
	}
	return c, true, nil
}

func hasClaimMarker(block string) bool {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "cortex: claim" || strings.HasPrefix(line, "type:") {
			return true
		}
	}
	return false
}

func validateClaim(c Claim) error {
	if c.ID == "" {
		return fmt.Errorf("missing id")
	}
	if strings.ContainsAny(c.ID, " \t\r\n") {
		return fmt.Errorf("id must not contain whitespace")
	}
	if !claimTypes[c.Type] {
		return fmt.Errorf("unsupported type %q", c.Type)
	}
	if !claimScopes[c.Scope] {
		return fmt.Errorf("unsupported scope %q", c.Scope)
	}
	if !claimStatuses[c.Status] {
		return fmt.Errorf("unsupported status %q", c.Status)
	}
	if c.UpdatedAt == "" {
		return fmt.Errorf("missing updated_at")
	}
	if _, err := time.Parse("2006-01-02", c.UpdatedAt); err != nil {
		return fmt.Errorf("updated_at must be YYYY-MM-DD")
	}
	if c.Text == "" {
		return fmt.Errorf("claim body is empty")
	}
	for _, source := range c.Sources {
		if err := validateRepoPath(source); err != nil {
			return fmt.Errorf("source %q: %w", source, err)
		}
	}
	for _, e := range c.Evidence {
		if err := validateRepoPath(e.Path); err != nil {
			return fmt.Errorf("evidence %q: %w", e.Raw, err)
		}
	}
	return nil
}

func validateRepoPath(path string) error {
	clean := filepath.Clean(filepath.FromSlash(path))
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path must stay inside repository")
	}
	return nil
}

func parseEvidence(raw string) (Evidence, error) {
	raw = cleanScalar(raw)
	e := Evidence{Raw: raw, Path: raw}
	marker := strings.LastIndex(raw, "#L")
	if marker < 0 {
		return e, nil
	}
	e.Path = raw[:marker]
	lines := raw[marker+2:]
	lines = strings.TrimPrefix(lines, "L")
	parts := strings.Split(lines, "-")
	if len(parts) > 2 || parts[0] == "" {
		return Evidence{}, fmt.Errorf("invalid evidence line range %q", raw)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil || start < 1 {
		return Evidence{}, fmt.Errorf("invalid evidence line range %q", raw)
	}
	e.StartLine, e.EndLine = start, start
	if len(parts) == 2 {
		end, err := strconv.Atoi(strings.TrimPrefix(parts[1], "L"))
		if err != nil || end < start {
			return Evidence{}, fmt.Errorf("invalid evidence line range %q", raw)
		}
		e.EndLine = end
	}
	if e.Path == "" {
		return Evidence{}, fmt.Errorf("evidence path is empty")
	}
	return e, nil
}

func frontmatter(content string) (string, string, bool) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content, false
	}
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return "", content, false
	}
	end += 4
	return content[4:end], content[end+4:], true
}

func cleanScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		return value[1 : len(value)-1]
	}
	return value
}

func parseInlineList(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = cleanScalar(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// LoadClaims returns valid claims in deterministic file order. Invalid or
// legacy files are left for Validate to report and never poison retrieval.
func LoadClaims(root string) ([]Claim, error) {
	files, err := LoadAll(root)
	if err != nil {
		return nil, err
	}
	var claims []Claim
	for _, f := range files {
		claim, ok, err := ParseClaim(f.RelPath, f.Content)
		if ok && err == nil {
			claims = append(claims, claim)
		}
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].File != claims[j].File {
			return claims[i].File < claims[j].File
		}
		return claims[i].ID < claims[j].ID
	})
	return claims, nil
}

// ValidateClaims checks duplicate IDs and relationship targets after parsing.
func ValidateClaims(root string) ([]ValidationItem, error) {
	files, err := LoadAll(root)
	if err != nil {
		return nil, err
	}
	var items []ValidationItem
	var claims []Claim
	ids := map[string]string{}
	for _, f := range files {
		claim, ok, parseErr := ParseClaim(f.RelPath, f.Content)
		if !ok {
			continue
		}
		if parseErr != nil {
			items = append(items, ValidationItem{Path: f.RelPath, State: StateInvalid, Detail: parseErr.Error()})
			continue
		}
		if previous, exists := ids[claim.ID]; exists {
			items = append(items, ValidationItem{Path: f.RelPath, State: StateInvalid, Detail: fmt.Sprintf("duplicate claim id %q (already in %s)", claim.ID, previous)})
			continue
		}
		ids[claim.ID] = f.RelPath
		claims = append(claims, claim)
	}
	for _, c := range claims {
		missing := missingRelations(c, ids)
		if len(missing) > 0 {
			items = append(items, ValidationItem{Path: c.File, State: StateInvalid, Detail: "missing related claim: " + strings.Join(missing, ", ")})
			continue
		}
		items = append(items, ValidationItem{Path: c.File, State: StateCurrent, Detail: fmt.Sprintf("claim %s (%s), status %s", c.ID, c.Type, c.Status)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items, nil
}

func missingRelations(c Claim, ids map[string]string) []string {
	var missing []string
	for _, id := range append(append([]string{}, c.Supersedes...), c.Contradicts...) {
		if _, ok := ids[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}
