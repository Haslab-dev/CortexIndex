package memory

import (
	"fmt"
	"sort"
	"strings"
)

type WorkRecord struct {
	File      string
	ID        string
	Scope     string
	Status    string
	UpdatedAt string
	Sections  map[string]string
}

var workStatuses = map[string]bool{"open": true, "active": true, "blocked": true, "completed": true, "abandoned": true}

func ParseWorkRecord(relPath, content string) (WorkRecord, bool, error) {
	block, body, ok := frontmatter(content)
	if !ok || !strings.Contains(block, "cortex: work") {
		return WorkRecord{}, false, nil
	}
	w := WorkRecord{File: relPath, Scope: "task", Status: "active", Sections: map[string]string{}}
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "cortex: work" {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return w, true, fmt.Errorf("%s: malformed work frontmatter", relPath)
		}
		key, value := strings.TrimSpace(line[:colon]), cleanScalar(line[colon+1:])
		switch key {
		case "id":
			w.ID = value
		case "scope":
			w.Scope = value
		case "status":
			w.Status = value
		case "updated_at":
			w.UpdatedAt = value
		}
	}
	w.Sections = markdownSections(body)
	if w.ID == "" || w.UpdatedAt == "" || !workStatuses[w.Status] || w.Sections["Goal"] == "" {
		return w, true, fmt.Errorf("%s: work requires id, updated_at, valid status, and a Goal section", relPath)
	}
	return w, true, nil
}

func LoadWorkRecords(root string) ([]WorkRecord, error) {
	files, err := LoadAll(root)
	if err != nil {
		return nil, err
	}
	var out []WorkRecord
	for _, f := range files {
		if w, ok, err := ParseWorkRecord(f.RelPath, f.Content); ok && err == nil {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

func markdownSections(body string) map[string]string {
	out := map[string]string{}
	current := ""
	var lines []string
	flush := func() {
		if current != "" {
			out[current] = strings.TrimSpace(strings.Join(lines, "\n"))
		}
		lines = nil
	}
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(trim, "## "))
			continue
		}
		if current != "" {
			lines = append(lines, line)
		}
	}
	flush()
	return out
}

func FormatWork(w WorkRecord) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("### %s [%s]\n\n", w.File, w.Status))
	b.WriteString(fmt.Sprintf("- Work ID: `%s`\n- Scope: `%s`\n- Updated: `%s`\n\n", w.ID, w.Scope, w.UpdatedAt))
	for _, name := range []string{"Goal", "Constraints", "Plan", "Decision", "Outcome", "Open Questions"} {
		if text := w.Sections[name]; text != "" {
			b.WriteString("**" + name + ":**\n" + text + "\n\n")
		}
	}
	return b.String()
}
