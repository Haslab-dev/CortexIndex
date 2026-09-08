package context

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"cortex/internal/memory"
	"cortex/internal/store"
)

// Intent controls context shape.
type Intent int

const (
	IntentTask Intent = iota
	IntentOverview
)

// DetectIntent recognizes broad repository-orientation requests.
func DetectIntent(task string) Intent {
	low := strings.ToLower(task)
	markers := []string{
		"overview", "summarize", "summary", "architecture", "repository structure",
		"codebase structure", "entry points", "project tour", "familiarize",
		"how is this project organized", "important modules", "major modules",
	}
	for _, marker := range markers {
		if strings.Contains(low, marker) {
			return IntentOverview
		}
	}
	return IntentTask
}

type overviewScore struct {
	sym    store.SymbolRow
	counts store.ReferenceCounts
	score  float64
}

func buildOverview(task string, o Options) string {
	sections := []scoredSection{}
	if m := overviewMemory(o.Root); m.body != "" {
		sections = append(sections, m)
	}
	if o.HasIndex && o.Store != nil {
		sections = append(sections, overviewModules(o.Store))
		sections = append(sections, overviewEntries(o.Store))
		sections = append(sections, overviewFlows(o.Store))
		sections = append(sections, overviewFiles(o.Store))
		sections = append(sections, overviewStats(o.Store))
	} else {
		sections = append(sections, scoredSection{name: "fallback", priority: 2, body: "## Index Status\n\n_No structural index is available. Run `cortex index` for symbols and relationships._\n\n"})
	}
	return renderOverview(task, sections, o.Budget)
}

func overviewMemory(root string) scoredSection {
	files, err := memory.LoadAll(root)
	if err != nil {
		return scoredSection{}
	}
	wanted := map[string]bool{"project.md": true, "architecture.md": true, "conventions.md": true}
	var b strings.Builder
	b.WriteString("## Project Memory\n\n")
	count := 0
	for _, f := range files {
		if !wanted[f.RelPath] && !strings.HasPrefix(f.RelPath, "decisions/") {
			continue
		}
		content := substantiveMemory(f.Content)
		if content == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", f.RelPath, content))
		count++
	}
	if count == 0 {
		return scoredSection{}
	}
	return scoredSection{name: "memory", body: b.String(), priority: 4}
}

func substantiveMemory(content string) string {
	content = stripComments(content)
	lines := strings.Split(content, "\n")
	var keep []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "<!--") || strings.HasPrefix(trim, "# <") {
			continue
		}
		bad := []string{"what is this project?", "primary languages, frameworks", "keep this small", "add repository-wide", "what is this module?", "what this module owns"}
		low := strings.ToLower(trim)
		ignored := false
		for _, phrase := range bad {
			if strings.Contains(low, phrase) {
				ignored = true
				break
			}
		}
		if !ignored {
			keep = append(keep, line)
		}
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

func overviewModules(st *store.Store) scoredSection {
	mods, err := st.ModuleSummaries()
	if err != nil || len(mods) == 0 {
		return scoredSection{}
	}
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].SymbolCount != mods[j].SymbolCount {
			return mods[i].SymbolCount > mods[j].SymbolCount
		}
		return mods[i].Path < mods[j].Path
	})
	var b strings.Builder
	b.WriteString("## Modules\n\n")
	for _, m := range mods {
		b.WriteString(fmt.Sprintf("### %s\n\n- Files: %d\n- Symbols: %d\n- Imports: %d\n\n", m.Path, m.FileCount, m.SymbolCount, m.ImportCount))
	}
	return scoredSection{name: "modules", body: b.String(), priority: 3}
}

func overviewEntries(st *store.Store) scoredSection {
	syms, err := st.ListSymbols(store.QueryParams{Limit: 100000})
	if err != nil {
		return scoredSection{}
	}
	counts, _ := st.SymbolReferenceCounts()
	var ranked []overviewScore
	for _, s := range syms {
		if isNoisePath(s.File) {
			continue
		}
		c := counts[s.ID]
		score := float64(c.Incoming*2+c.Outgoing) + 1
		if s.Parent == "" {
			score += 1
		}
		if isExportedName(s.Name) {
			score += 1
		}
		low := strings.ToLower(s.Name)
		for _, marker := range []string{"main", "run", "start", "serve", "execute", "handle", "init"} {
			if low == marker || strings.Contains(low, marker) {
				score += 3
			}
		}
		for _, dir := range []string{"cmd/", "app/", "server/", "api/", "routes/"} {
			if strings.Contains(s.File, dir) {
				score += 2
			}
		}
		if isGenericName(s.Name) {
			score -= 1.5
		}
		ranked = append(ranked, overviewScore{sym: s, counts: c, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].sym.Display() < ranked[j].sym.Display()
	})
	if len(ranked) > 12 {
		ranked = ranked[:12]
	}
	if len(ranked) == 0 {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Entry Points\n\n")
	for _, r := range ranked {
		b.WriteString(fmt.Sprintf("### %s (%s)\n\n- Location: `%s:%d`\n- Signature: `%s`\n- References: %d incoming, %d outgoing\n\n", r.sym.Display(), r.sym.Kind, r.sym.File, r.sym.StartLine, r.sym.Signature, r.counts.Incoming, r.counts.Outgoing))
	}
	return scoredSection{name: "entries", body: b.String(), priority: 5}
}

func overviewFlows(st *store.Store) scoredSection {
	syms, err := st.ListSymbols(store.QueryParams{Limit: 100000})
	if err != nil {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Key Flows\n\n")
	count := 0
	for _, s := range syms {
		if isNoisePath(s.File) || s.Kind == string("struct") || s.Kind == string("class") {
			continue
		}
		refs, err := st.RefsFromSymbol(s.ID)
		if err != nil || len(refs) == 0 {
			continue
		}
		var labels []string
		for _, r := range refs {
			label := r.Name
			if r.Qualifier != "" {
				label = r.Qualifier + "." + r.Name
			}
			labels = append(labels, label)
			if len(labels) == 4 {
				break
			}
		}
		b.WriteString(fmt.Sprintf("- `%s` → %s\n", s.Display(), strings.Join(labels, ", ")))
		count++
		if count == 10 {
			break
		}
	}
	if count == 0 {
		return scoredSection{}
	}
	b.WriteString("\n")
	return scoredSection{name: "flows", body: b.String(), priority: 2}
}

func overviewFiles(st *store.Store) scoredSection {
	files, err := st.ListFiles(store.QueryParams{Limit: 100000})
	if err != nil {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Relevant Files\n\n")
	seen := map[string]bool{}
	for _, f := range files {
		module := filepath.ToSlash(filepath.Dir(f.Path))
		if module == "." || seen[module] {
			continue
		}
		seen[module] = true
		b.WriteString(fmt.Sprintf("- `%s/`\n", module))
		if len(seen) == 12 {
			break
		}
	}
	b.WriteString("\n")
	return scoredSection{name: "files", body: b.String(), priority: 1}
}

func overviewStats(st *store.Store) scoredSection {
	files, symbols, refs, imports, err := st.Stats()
	if err != nil {
		return scoredSection{}
	}
	return scoredSection{name: "stats", body: fmt.Sprintf("## Index Summary\n\n- Files: %d\n- Symbols: %d\n- References: %d\n- Imports: %d\n\n", files, symbols, refs, imports), priority: 4}
}

func renderOverview(task string, sections []scoredSection, budget int) string {
	order := []string{"memory", "modules", "entries", "flows", "files", "stats", "fallback"}
	for {
		var b strings.Builder
		b.WriteString("# Codebase Overview\n\n")
		b.WriteString(fmt.Sprintf("> Request: %s\n\n", task))
		for _, name := range order {
			for _, s := range sections {
				if s.name == name {
					b.WriteString(s.body)
				}
			}
		}
		if estTokenLen(b.String()) <= budget || len(sections) == 0 {
			b.WriteString(fmt.Sprintf("\n---\n*~%d tokens estimated.*\n", estTokenLen(b.String())))
			return b.String()
		}
		worst := 0
		for i := range sections {
			if sections[i].priority < sections[worst].priority {
				worst = i
			}
		}
		sections = append(sections[:worst], sections[worst+1:]...)
	}
}

func isNoisePath(path string) bool {
	low := strings.ToLower(filepath.ToSlash(path))
	if filepath.Base(path) == ".DS_Store" || strings.HasSuffix(low, ".db") {
		return true
	}
	for _, marker := range []string{"_test.", "/test/", "/tests/", "/fixtures/", "/mocks/", "/snapshots/", "/examples/", "/benchmarks/", "generated", "/gen/", "template"} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

func isExportedName(name string) bool { return name != "" && name[0] >= 'A' && name[0] <= 'Z' }

func isGenericName(name string) bool {
	low := strings.ToLower(name)
	for _, v := range []string{"helper", "util", "utils", "test", "setup", "main"} {
		if low == v {
			return true
		}
	}
	return false
}
