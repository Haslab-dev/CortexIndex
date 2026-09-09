package memory

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Preference struct {
	File           string
	ID             string
	Key            string
	Scope          string
	Status         string
	Confidence     float64
	UpdatedAt      string
	Provider       string
	Source         string
	Package        string
	PackageVersion string
	Text           string
}

func ParsePreference(relPath, content string) (Preference, bool, error) {
	block, body, ok := frontmatter(content)
	if !ok || !strings.Contains(block, "cortex: preference") {
		return Preference{}, false, nil
	}
	p := Preference{File: relPath, Scope: "repository", Status: "proposed", Provider: "manual"}
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "cortex: preference" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return p, true, fmt.Errorf("%s: malformed preference frontmatter", relPath)
		}
		key, value := strings.TrimSpace(line[:colon]), cleanScalar(line[colon+1:])
		switch key {
		case "id":
			p.ID = value
		case "key":
			p.Key = value
		case "package":
			p.Package = value
		case "package_version":
			p.PackageVersion = value
		case "scope":
			p.Scope = strings.ToLower(value)
		case "status":
			p.Status = strings.ToLower(value)
		case "updated_at":
			p.UpdatedAt = value
		case "provider":
			p.Provider = value
		case "source":
			p.Source = value
		case "confidence":
			n, err := strconv.ParseFloat(value, 64)
			if err != nil || n < 0 || n > 1 {
				return p, true, fmt.Errorf("%s: confidence must be between 0 and 1", relPath)
			}
			p.Confidence = n
		}
	}
	p.Text = strings.TrimSpace(body)
	if p.Key == "" {
		p.Key = p.ID
	}
	validScopes := map[string]bool{"repository": true, "team": true, "user": true}
	validStatuses := map[string]bool{"active": true, "proposed": true, "disabled": true, "deprecated": true, "superseded": true}
	if p.ID == "" || p.Text == "" || p.UpdatedAt == "" || !validScopes[p.Scope] || !validStatuses[p.Status] {
		return p, true, fmt.Errorf("%s: invalid preference metadata or empty body", relPath)
	}
	return p, true, nil
}

func LoadPreferences(root string) ([]Preference, error) {
	files, err := LoadAll(root)
	if err != nil {
		return nil, err
	}
	var out []Preference
	for _, f := range files {
		if p, ok, err := ParsePreference(f.RelPath, f.Content); ok && err == nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

// TastePreference is the provider-neutral export shape accepted from a Taste-like system.
type TastePreference struct {
	ID         string
	Scope      string
	Text       string
	Confidence float64
	UpdatedAt  string
	Source     string
}

// NormalizeTaste converts an imported preference to a reviewable local record.
func NormalizeTaste(in TastePreference, relPath string) (Preference, error) {
	if in.ID == "" || in.Text == "" || in.UpdatedAt == "" || in.Confidence < 0 || in.Confidence > 1 {
		return Preference{}, fmt.Errorf("invalid Taste preference %q", in.ID)
	}
	if in.Scope == "" {
		in.Scope = "user"
	}
	return Preference{File: relPath, ID: in.ID, Key: in.ID, Scope: in.Scope, Status: "proposed", Confidence: in.Confidence, UpdatedAt: in.UpdatedAt, Provider: "taste", Source: in.Source, Text: in.Text}, nil
}

func FormatPreference(p Preference) string {
	return fmt.Sprintf("### %s [%s]\n\n%s\n\n- Preference ID: `%s`\n- Key: `%s`\n- Scope: `%s`\n- Confidence: %.2f\n- Provider: `%s`\n- Package: `%s`\n- Status: `%s`\n- Updated: `%s`\n\n", p.File, p.Status, p.Text, p.ID, p.Key, p.Scope, p.Confidence, p.Provider, p.Package, p.Status, p.UpdatedAt)
}
