// Package context builds the flagship `cortex context "<task>"` response:
// layered retrieval (memory → structural → lexical → source) with
// relevance ranking and a token budget, rendered as Markdown for LLM
// consumption (PRD §14, §15, §21–§23, §27).
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cortex/internal/lexsearch"
	"cortex/internal/memory"
	"cortex/internal/store"
)

// Options controls a context build.
type Options struct {
	Budget         int    // approximate token budget for the whole response
	Root           string // repo root
	HasIndex       bool   // false → lexical fallback only
	Store          *store.Store
	MaxSymbols     int // cap on symbols returned (default 8)
	WithSources    int // how many top symbols get source bodies (default 3)
	Intent         Intent
	ExplicitIntent bool
}

// DefaultBudget is the default approximate token budget.
const DefaultBudget = 6000

// Build renders task-specific context. It never fails hard: any layer that
// comes up empty is skipped, and a missing index falls back to lexical
// repository search (PRD §27).
func Build(task string, o Options) string {
	if o.Budget <= 0 {
		o.Budget = DefaultBudget
	}
	if o.MaxSymbols <= 0 {
		o.MaxSymbols = 8
	}
	if o.WithSources <= 0 {
		o.WithSources = 3
	}
	if !o.ExplicitIntent {
		o.Intent = DetectIntent(task)
	}
	if o.Intent == IntentOverview {
		return buildOverview(task, o)
	}
	keywords := Keywords(task)

	var sections []scoredSection

	// Layer 1 — Markdown memory.
	memoryHit := memorySection(o.Root, keywords)
	if memoryHit.body != "" {
		sections = append(sections, memoryHit)
	}

	if o.HasIndex && o.Store != nil {
		// Layer 2/5 — structural retrieval + source extraction.
		syms := symbolCandidates(o, keywords)
		sections = append(sections, symbolsSection(o, syms))
		sections = append(sections, sourcesSection(o, syms))
		sections = append(sections, filesSection(o, keywords))
		// Layer 3 supplement: lexical scan when structural retrieval came
		// up empty (rare-task vocabulary, config strings, etc.).
		if len(syms) == 0 {
			sections = append(sections, lexicalFallback(o.Root, keywords))
		}
	}

	// Layer 3 fallback — lexical scan when no/poor index results.
	if len(sections) <= 1 && !o.HasIndex {
		sections = append(sections, lexicalFallback(o.Root, keywords))
	}

	// Token budget: trim lowest-priority source, then memory.
	out := render(task, sections, o.Budget)

	if !o.HasIndex {
		out += "\n> No index available — showing lexical scan only. Run `cortex index` for structural results.\n"
	}
	return out
}

// ---- keyword extraction ----

var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "but": true, "by": true, "can": true, "do": true, "does": true,
	"for": true, "from": true, "how": true, "i": true, "in": true, "is": true,
	"it": true, "me": true, "my": true, "of": true, "on": true, "or": true,
	"the": true, "this": true, "to": true, "up": true, "was": true, "what": true,
	"where": true, "which": true, "who": true, "why": true, "with": true,
	"work": true, "works": true, "working": true, "use": true, "used": true,
	"using": true, "add": true, "change": true, "fix": true, "find": true,
}

// Keywords extracts lowercase search terms, splitting camelCase identifiers.
func Keywords(task string) []string {
	fields := strings.FieldsFunc(task, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_')
	})
	seen := map[string]bool{}
	var out []string
	for _, f := range fields {
		if len(f) > 1 && !stopwords[strings.ToLower(f)] {
			low := strings.ToLower(f)
			if !seen[low] {
				seen[low] = true
				out = append(out, low)
			}
			// camelCase split parts are individually valuable
			for _, p := range splitCamel(f) {
				if len(p) > 2 && !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func splitCamel(s string) []string {
	var parts []string
	var cur strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' && i > 0 {
			parts = append(parts, strings.ToLower(cur.String()))
			cur.Reset()
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		parts = append(parts, strings.ToLower(cur.String()))
	}
	return parts
}

// ---- ranking ----

type score struct {
	sym      store.SymbolRow
	points   float64
	reasons  []string
	callers  int
	calledBy int
}

// symbolCandidates finds and ranks symbols for the task keywords.
func symbolCandidates(o Options, keywords []string) []score {
	byID := map[int64]*score{}
	add := func(sym store.SymbolRow, pts float64, why string) {
		s, ok := byID[sym.ID]
		if !ok {
			s = &score{sym: sym}
			byID[sym.ID] = s
		}
		s.points += pts
		if why != "" && len(s.reasons) < 4 {
			s.reasons = append(s.reasons, why)
		}
	}

	for _, kw := range keywords {
		collect := func(kw string, namePts, ftsPts, callPts float64) {
			for _, r := range must(o.Store.SearchSymbolsExact(kw, store.QueryParams{Limit: 15})) {
				add(r, namePts, "name match: "+kw)
			}
			for _, r := range must(o.Store.SearchSymbolsFTS(`"`+kw+`"*`, store.QueryParams{Limit: 15})) {
				add(r, ftsPts, "text match: "+kw)
			}
			if refs, err := o.Store.RefsByName(kw, 40); err == nil {
				for _, ref := range refs {
					if ref.FromID == 0 {
						continue
					}
					if sym, err := o.Store.GetSymbolByID(ref.FromID); err == nil {
						add(sym, callPts, "calls "+kw)
					}
				}
			}
		}
		collect(kw, 5.0, 2.0, 1.5)
	}

	// Stemming fallback: natural-language words like "authentication" must
	// still reach "auth"-named symbols. Retry with 5- and 4-char prefixes at
	// lower weight when full words found nothing.
	if len(byID) == 0 {
		for _, kw := range keywords {
			for _, n := range []int{5, 4} {
				if len(kw) > n {
					for _, r := range must(o.Store.SearchSymbolsExact(kw[:n], store.QueryParams{Limit: 15})) {
						add(r, 3.0, "stem match: "+kw[:n])
					}
					for _, r := range must(o.Store.SearchSymbolsFTS(`"`+kw[:n]+`"*`, store.QueryParams{Limit: 15})) {
						add(r, 1.0, "stem text: "+kw[:n])
					}
				}
			}
		}
	}

	// call-degree bonus (referenced symbols matter more); test symbols are
	// down-ranked — they rarely answer "how does X work" questions.
	var out []score
	for _, s := range byID {
		if strings.HasSuffix(s.sym.File, "_test.go") {
			s.points *= 0.4
		}
		if refs, err := o.Store.RefsByName(s.sym.Name, 1); err == nil && len(refs) > 0 {
			s.points += 0.5
			s.callers++
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].points != out[j].points {
			return out[i].points > out[j].points
		}
		return out[i].sym.Display() < out[j].sym.Display()
	})
	if len(out) > o.MaxSymbols {
		out = out[:o.MaxSymbols]
	}
	return out
}

func must(rows []store.SymbolRow, err error) []store.SymbolRow {
	if err != nil {
		return nil
	}
	return rows
}

// ---- sections ----

type scoredSection struct {
	name     string
	body     string
	priority int // higher = trimmed last
}

// memorySection renders relevant durable knowledge (Layer 1).
func memorySection(root string, keywords []string) scoredSection {
	files, err := memory.LoadAll(root)
	if err != nil || len(files) == 0 {
		return scoredSection{}
	}
	var picked []memory.MemoryFile
	for _, f := range files {
		low := strings.ToLower(f.Content)
		hits := 0
		for _, kw := range keywords {
			if strings.Contains(low, kw) {
				hits++
			}
		}
		if hits > 0 {
			picked = append(picked, f)
			if len(picked) >= 3 {
				break
			}
		}
	}
	if len(picked) == 0 {
		// conventions are cheap and always useful
		for _, f := range files {
			if f.RelPath == "conventions.md" && strings.TrimSpace(stripComments(f.Content)) != "" {
				picked = append(picked, f)
				break
			}
		}
	}
	if len(picked) == 0 {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Project Memory\n\n")
	for _, f := range picked {
		content := strings.TrimRight(stripComments(f.Content), "\n")
		if content == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", f.RelPath, content))
	}
	return scoredSection{name: "memory", body: b.String(), priority: 1}
}

// symbolsSection renders the ranked symbol table (Layer 2).
func symbolsSection(o Options, syms []score) scoredSection {
	if len(syms) == 0 {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Relevant Symbols\n\n")
	for _, s := range syms {
		b.WriteString(fmt.Sprintf("### %s (%s) — relevance %.2f\n\n", s.sym.Display(), s.sym.Kind, s.points))
		b.WriteString(fmt.Sprintf("- Location: `%s:%d:%d`\n", s.sym.File, s.sym.StartLine, s.sym.StartCol+1))
		if s.sym.Signature != "" {
			b.WriteString(fmt.Sprintf("- Signature: `%s`\n", s.sym.Signature))
		}
		if s.sym.Doc != "" {
			b.WriteString(fmt.Sprintf("- Doc: %s\n", firstLine(s.sym.Doc)))
		}
		if len(s.reasons) > 0 {
			b.WriteString(fmt.Sprintf("- Why: %s\n", strings.Join(s.reasons, "; ")))
		}
		// one-hop callees
		if refs, err := o.Store.RefsFromSymbol(s.sym.ID); err == nil && len(refs) > 0 {
			seen := map[string]bool{}
			var calls []string
			for _, r := range refs {
				label := r.Name
				if r.Qualifier != "" {
					label = r.Qualifier + "." + r.Name
				}
				if !seen[label] {
					seen[label] = true
					calls = append(calls, label)
				}
			}
			b.WriteString(fmt.Sprintf("- Calls: %s\n", strings.Join(calls, ", ")))
		}
		b.WriteString("\n")
	}
	return scoredSection{name: "symbols", body: b.String(), priority: 3}
}

// sourcesSection renders source for the top-ranked symbols (Layer 5).
func sourcesSection(o Options, syms []score) scoredSection {
	n := o.WithSources
	if n > len(syms) {
		n = len(syms)
	}
	var b strings.Builder
	wrote := false
	for i := 0; i < n; i++ {
		s := syms[i]
		src, ok, _ := o.Store.ChunkForSymbol(s.sym.ID)
		if !ok || src == "" {
			src = readLines(o.Root, s.sym.File, s.sym.StartLine, s.sym.EndLine)
		}
		if src == "" {
			continue
		}
		if !wrote {
			b.WriteString("## Source\n\n")
			wrote = true
		}
		b.WriteString(fmt.Sprintf("### %s — `%s:%d`\n\n```%s\n%s\n```\n\n",
			s.sym.Display(), s.sym.File, s.sym.StartLine, langHint(s.sym.File),
			strings.TrimRight(src, "\n")))
	}
	if !wrote {
		return scoredSection{}
	}
	return scoredSection{name: "sources", body: b.String(), priority: 0} // trimmed first
}

// filesSection lists files relevant to the keywords (content FTS).
func filesSection(o Options, keywords []string) scoredSection {
	if len(keywords) == 0 {
		return scoredSection{}
	}
	// Try full keywords first, then shorten prefixes: natural-language
	// words ("authentication") must still match inflected tokens in code
	// ("authenticates") even though FTS5 prefix matching only works
	// left-to-right within one token.
	var hits []store.FileHit
	for width := 0; width <= 8 && len(hits) == 0; width += 2 {
		var terms []string
		for _, kw := range keywords {
			term := kw
			if width > 0 {
				if len(kw) <= width+3 {
					break
				}
				term = kw[:len(kw)-width]
			}
			terms = append(terms, `"`+strings.ReplaceAll(term, `"`, `""`)+`"*`)
		}
		if len(terms) == 0 {
			break
		}
		hits, _ = o.Store.SearchFilesFTS(strings.Join(terms, " OR "), store.QueryParams{Limit: 8})
	}
	if len(hits) == 0 {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Relevant Files\n\n")
	for _, h := range hits {
		b.WriteString(fmt.Sprintf("- `%s`\n", h.Path))
	}
	b.WriteString("\n")
	return scoredSection{name: "files", body: b.String(), priority: 1}
}

// lexicalFallback is the no-index path (PRD §27).
func lexicalFallback(root string, keywords []string) scoredSection {
	if len(keywords) == 0 {
		return scoredSection{}
	}
	hits, err := lexsearch.Scan(root, keywords, 8)
	if err != nil || len(hits) == 0 {
		return scoredSection{}
	}
	var b strings.Builder
	b.WriteString("## Lexical Scan (no index available)\n\n")
	for _, h := range hits {
		b.WriteString(fmt.Sprintf("- `%s:%d` — %s\n", h.Path, h.Line, h.Text))
	}
	b.WriteString("\n")
	return scoredSection{name: "fallback", body: b.String(), priority: 0}
}

// ---- rendering with budget ----

// sectionOrder is the fixed output order; priority is the trim order
// (lower priority = trimmed first).
var sectionOrder = []struct {
	name     string
	priority int
}{
	{"memory", 1},
	{"symbols", 3},
	{"sources", 0},
	{"files", 1},
	{"fallback", 2},
}

// render assembles sections in canonical order, trimming lowest-priority
// sections until under budget. Sources trim first, memory/files trim last.
func render(task string, sections []scoredSection, budget int) string {
	avail := make([]scoredSection, 0, len(sections))
	for _, want := range sectionOrder {
		for _, s := range sections {
			if s.name == want.name && s.body != "" {
				avail = append(avail, s)
				break
			}
		}
	}

	for {
		var b strings.Builder
		b.WriteString("# Codebase Context\n\n")
		b.WriteString(fmt.Sprintf("> Task: %s\n\n", task))
		for _, s := range avail {
			b.WriteString(s.body)
		}
		if estTokenLen(b.String()) <= budget || len(avail) == 0 {
			b.WriteString(fmt.Sprintf("\n---\n*~%d tokens estimated.*\n", estTokenLen(b.String())))
			return b.String()
		}
		// drop the lowest-priority section present
		worst := 0
		for i, s := range avail {
			if s.priority < avail[worst].priority {
				worst = i
			}
		}
		avail = append(avail[:worst], avail[worst+1:]...)
	}
}

// estTokenLen approximates tokens as chars/4 — good enough for budgeting.
func estTokenLen(s string) int { return len(s) / 4 }

// ---- small helpers ----

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// stripComments removes HTML comments from Markdown templates so empty
// scaffolding doesn't leak into context.
func stripComments(s string) string {
	for {
		start := strings.Index(s, "<!--")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "-->")
		if end < 0 {
			return s[:start]
		}
		s = s[:start] + s[start+end+3:]
	}
}

func readLines(root, path string, start, end int) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

func langHint(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".ts"):
		return "typescript"
	case strings.HasSuffix(path, ".tsx"):
		return "tsx"
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return "javascript"
	case strings.HasSuffix(path, ".py"):
		return "python"
	case strings.HasSuffix(path, ".rs"):
		return "rust"
	case strings.HasSuffix(path, ".java"):
		return "java"
	case strings.HasSuffix(path, ".kt"):
		return "kotlin"
	case strings.HasSuffix(path, ".c"):
		return "c"
	case strings.HasSuffix(path, ".cpp"), strings.HasSuffix(path, ".cc"), strings.HasSuffix(path, ".hpp"):
		return "cpp"
	}
	return ""
}
