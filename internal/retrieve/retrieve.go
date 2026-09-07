// Package retrieve implements the retrieval commands: search, symbol,
// refs, deps. All output is Markdown optimized for LLM consumption,
// smallest useful unit first (PRD §31.5).
package retrieve

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cortex/internal/store"
)

// Engine binds retrieval queries to an open store.
type Engine struct {
	St   *store.Store
	Root string // repo root for reading source from disk
}

// New creates an Engine.
func New(st *store.Store, root string) *Engine { return &Engine{St: st, Root: root} }

// ---- search ----

// Search runs layered symbol + file search for a free-text query.
func (e *Engine) Search(query string, limit int) (string, error) {
	if limit <= 0 {
		limit = 20
	}
	match := store.SanitizeFTS(query)
	if match == "" {
		return "No searchable terms in query.\n", nil
	}

	var b strings.Builder
	b.WriteString("# Search: " + query + "\n\n")

	// Symbols: exact name hits rank first, then FTS.
	exact, err := e.St.SearchSymbolsExact(strings.TrimSpace(query), store.QueryParams{Limit: limit})
	if err != nil {
		return "", err
	}
	fts, err := e.St.SearchSymbolsFTS(match, store.QueryParams{Limit: limit})
	if err != nil {
		return "", err
	}
	seen := map[int64]bool{}
	var symbols []store.SymbolRow
	for _, r := range exact {
		if !seen[r.ID] {
			seen[r.ID] = true
			symbols = append(symbols, r)
		}
	}
	for _, r := range fts {
		if !seen[r.ID] {
			seen[r.ID] = true
			symbols = append(symbols, r)
		}
	}
	if len(symbols) > 0 {
		b.WriteString("## Symbols\n\n")
		for _, s := range symbols {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — `%s:%d`\n", s.Display(), s.Kind, s.File, s.StartLine))
			if s.Signature != "" {
				b.WriteString("  ```\n  " + s.Signature + "\n  ```\n")
			}
		}
		b.WriteString("\n")
	}

	// Files via content FTS.
	hits, err := e.St.SearchFilesFTS(match, store.QueryParams{Limit: limit})
	if err != nil {
		return "", err
	}
	if len(hits) > 0 {
		b.WriteString("## Files (content matches)\n\n")
		for _, h := range hits {
			b.WriteString(fmt.Sprintf("- `%s`\n  > %s\n", h.Path, strings.ReplaceAll(h.Snippet, "\n", " ")))
		}
		b.WriteString("\n")
	}

	if len(symbols) == 0 && len(hits) == 0 {
		b.WriteString("_No matches. Try `cortex index` if the index is empty, or broaden the query._\n")
	}
	return b.String(), nil
}

// ---- symbol ----

// SymbolDetail renders one symbol with its call context; when withSource
// is set, the stored chunk (or the file slice) is included.
func (e *Engine) Symbol(name string, withSource bool) (string, error) {
	rows, err := e.lookupSymbol(name)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return fmt.Sprintf("Symbol %q not found. Try `cortex search %s`.\n", name, name), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Symbol: %s\n\n", name))
	for i, s := range rows {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		b.WriteString(fmt.Sprintf("## %s (%s)\n\n", s.Display(), s.Kind))
		b.WriteString(fmt.Sprintf("- Location: `%s:%d:%d`\n", s.File, s.StartLine, s.StartCol+1))
		if s.Parent != "" {
			b.WriteString(fmt.Sprintf("- Parent: %s\n", s.Parent))
		}
		if s.Signature != "" {
			b.WriteString(fmt.Sprintf("- Signature: `%s`\n", s.Signature))
		}
		if s.Doc != "" {
			b.WriteString("- Doc:\n\n")
			for _, line := range strings.Split(s.Doc, "\n") {
				b.WriteString("  > " + line + "\n")
			}
			b.WriteString("\n")
		}

		// Calls made by this symbol.
		refs, err := e.St.RefsFromSymbol(s.ID)
		if err == nil && len(refs) > 0 {
			b.WriteString("- Calls:\n")
			seen := map[string]bool{}
			for _, r := range refs {
				label := r.Name
				if r.Qualifier != "" {
					label = r.Qualifier + "." + r.Name
				}
				if seen[label] {
					continue
				}
				seen[label] = true
				b.WriteString(fmt.Sprintf("  - %s (`%s:%d`)\n", label, r.File, r.Line))
			}
			b.WriteString("\n")
		}

		// Callers across the repo.
		callers, err := e.St.RefsByName(s.Name, 50)
		if err == nil {
			var usable []store.RefRow
			for _, c := range callers {
				if c.FromName != "" && c.FromName != s.Display() {
					usable = append(usable, c)
				}
			}
			if len(usable) > 0 {
				b.WriteString("- Called by:\n")
				for _, c := range usable {
					b.WriteString(fmt.Sprintf("  - %s (`%s:%d`)\n", c.FromName, c.File, c.Line))
				}
				b.WriteString("\n")
			}
		}

		if withSource {
			src, ok, _ := e.St.ChunkForSymbol(s.ID)
			if !ok {
				src = e.readSlice(s.File, s.StartLine, s.EndLine)
			}
			if src != "" {
				b.WriteString("```" + e.langHint(s.File) + "\n" + strings.TrimRight(src, "\n") + "\n```\n")
			}
		}
	}
	return b.String(), nil
}

// lookupSymbol resolves a name to symbols: exact display match first
// ("Server.Login"), then exact name, then FTS, then prefix.
func (e *Engine) lookupSymbol(name string) ([]store.SymbolRow, error) {
	// qualified "Parent.Name"
	if dot := strings.LastIndex(name, "."); dot > 0 {
		parent, leaf := name[:dot], name[dot+1:]
		all, err := e.St.SearchSymbolsExact(leaf, store.QueryParams{Limit: 10})
		if err == nil {
			var filtered []store.SymbolRow
			for _, r := range all {
				if strings.EqualFold(r.Parent, parent) {
					filtered = append(filtered, r)
				}
			}
			if len(filtered) > 0 {
				return filtered, nil
			}
		}
	}
	if exact, err := e.St.SearchSymbolsExact(name, store.QueryParams{Limit: 10}); err == nil && len(exact) > 0 {
		return exact, nil
	}
	if fts, err := e.St.SearchSymbolsFTS(store.SanitizeFTS(name), store.QueryParams{Limit: 10}); err == nil && len(fts) > 0 {
		return fts, nil
	}
	return e.St.SearchSymbolsPrefix(name, store.QueryParams{Limit: 10})
}

// ---- refs ----

// Refs lists all references to a name with locations.
func (e *Engine) Refs(name string) (string, error) {
	refs, err := e.St.RefsByName(name, 200)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# References to %s\n\n", name))
	if len(refs) == 0 {
		b.WriteString("_No references found._\n")
		return b.String(), nil
	}
	byFile := map[string][]store.RefRow{}
	var files []string
	for _, r := range refs {
		if _, ok := byFile[r.File]; !ok {
			files = append(files, r.File)
		}
		byFile[r.File] = append(byFile[r.File], r)
	}
	sort.Strings(files)
	for _, f := range files {
		b.WriteString(fmt.Sprintf("## %s\n\n", f))
		for _, r := range byFile[f] {
			kind := r.Kind
			line := fmt.Sprintf("- `%s:%d`", f, r.Line)
			if r.FromName != "" {
				b.WriteString(fmt.Sprintf("%s — %s in **%s**\n", line, kind, r.FromName))
			} else {
				b.WriteString(fmt.Sprintf("%s — %s\n", line, kind))
			}
		}
	}
	return b.String(), nil
}

// ---- deps ----

// Deps reports what a symbol depends on (callees, file imports) and what
// depends on it (callers).
func (e *Engine) Deps(name string) (string, error) {
	rows, err := e.lookupSymbol(name)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return fmt.Sprintf("Symbol %q not found.\n", name), nil
	}
	s := rows[0]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Dependencies: %s\n\n", s.Display()))

	// Callees.
	refs, err := e.St.RefsFromSymbol(s.ID)
	if err != nil {
		return "", err
	}
	if len(refs) > 0 {
		b.WriteString("## Depends on (callees)\n\n")
		seen := map[string]bool{}
		for _, r := range refs {
			label := r.Name
			if r.Qualifier != "" {
				label = r.Qualifier + "." + r.Name
			}
			key := label + ":" + r.Kind
			if seen[key] {
				continue
			}
			seen[key] = true
			b.WriteString(fmt.Sprintf("- %s (%s)\n", label, r.Kind))
		}
		b.WriteString("\n")
	}

	// Imports of the file containing the symbol.
	imps, err := e.St.ImportsForFile(s.File)
	if err == nil && len(imps) > 0 {
		b.WriteString(fmt.Sprintf("## Imports of `%s`\n\n", s.File))
		for _, imp := range imps {
			if len(imp.Names) > 0 {
				b.WriteString(fmt.Sprintf("- `%s` (%s)\n", imp.Path, strings.Join(imp.Names, ", ")))
			} else {
				b.WriteString(fmt.Sprintf("- `%s`\n", imp.Path))
			}
		}
		b.WriteString("\n")
	}

	// Callers.
	callers, err := e.St.RefsByName(s.Name, 100)
	if err != nil {
		return "", err
	}
	var usable []store.RefRow
	seen := map[string]bool{}
	for _, c := range callers {
		if c.FromName != "" && c.FromName != s.Display() && !seen[c.FromName] {
			seen[c.FromName] = true
			usable = append(usable, c)
		}
	}
	if len(usable) > 0 {
		b.WriteString("## Depended on by (callers)\n\n")
		for _, c := range usable {
			b.WriteString(fmt.Sprintf("- %s (`%s:%d`)\n", c.FromName, c.File, c.Line))
		}
		b.WriteString("\n")
	}

	if len(refs) == 0 && len(usable) == 0 && (imps == nil || len(imps) == 0) {
		b.WriteString("_No dependency information found._\n")
	}
	return b.String(), nil
}

// readSlice returns lines [start,end] of a file (1-based, inclusive).
func (e *Engine) readSlice(path string, start, end int) string {
	data, err := os.ReadFile(filepath.Join(e.Root, filepath.FromSlash(path)))
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

func (e *Engine) langHint(path string) string {
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
