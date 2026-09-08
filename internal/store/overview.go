package store

import (
	"sort"
	"strings"
)

// ReferenceCounts summarizes incoming/outgoing reference volume for a symbol.
type ReferenceCounts struct {
	Incoming int
	Outgoing int
}

// ListFiles returns indexed files in deterministic path order.
func (s *Store) ListFiles(p QueryParams) ([]File, error) {
	q := `SELECT path, language, size, hash, mtime, parsed FROM files`
	args := []any{}
	if p.File != "" {
		q += ` WHERE path LIKE ?`
		args = append(args, p.File+"%")
	}
	q += ` ORDER BY path LIMIT ?`
	args = append(args, p.limit())
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		var f File
		var parsed int
		if err := rows.Scan(&f.Path, &f.Language, &f.Size, &f.Hash, &f.MTime, &parsed); err != nil {
			return nil, err
		}
		f.Parsed = parsed == 1
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListSymbols returns indexed symbols in deterministic file/range order.
func (s *Store) ListSymbols(p QueryParams) ([]SymbolRow, error) {
	q := `SELECT id, file_path, name, kind, signature, parent, doc,
		start_line, start_col, end_line, end_col, 0 FROM symbols`
	args := []any{}
	where := ""
	if p.Kind != "" {
		where = ` WHERE kind = ?`
		args = append(args, p.Kind)
	}
	if p.File != "" {
		if where == "" {
			where = " WHERE file_path LIKE ?"
		} else {
			where += ` AND file_path LIKE ?`
		}
		args = append(args, p.File+"%")
	}
	q += where + ` ORDER BY file_path, start_line, start_col, id LIMIT ?`
	args = append(args, p.limit())
	return s.symbolRows(q, args...)
}

// AllImports returns every import in deterministic source order.
func (s *Store) AllImports() ([]ImportRow, error) {
	rows, err := s.db.Query(`SELECT file_path, module, names, line FROM imports ORDER BY file_path, line, module`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ImportRow
	for rows.Next() {
		var r ImportRow
		var names string
		if err := rows.Scan(&r.File, &r.Path, &names, &r.Line); err != nil {
			return nil, err
		}
		if names != "" {
			r.Names = splitComma(names)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func splitComma(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// SymbolReferenceCounts returns aggregate counts without N+1 queries.
func (s *Store) SymbolReferenceCounts() (map[int64]ReferenceCounts, error) {
	rows, err := s.db.Query(`
		SELECT id, incoming, outgoing FROM (
			SELECT sym.id,
				(SELECT COUNT(*) FROM refs r WHERE r.name = sym.name) AS incoming,
				(SELECT COUNT(*) FROM refs r WHERE r.from_symbol_id = sym.id) AS outgoing
			FROM symbols sym
		)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]ReferenceCounts{}
	for rows.Next() {
		var id int64
		var c ReferenceCounts
		if err := rows.Scan(&id, &c.Incoming, &c.Outgoing); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}

// ModuleFileStats aggregates files and symbols by their first two path
// components (or one component for shallow paths).
type ModuleFileStats struct {
	Path        string
	FileCount   int
	SymbolCount int
	ImportCount int
}

// ModuleSummaries derives stable module boundaries from indexed paths.
func (s *Store) ModuleSummaries() ([]ModuleFileStats, error) {
	files, err := s.ListFiles(QueryParams{Limit: 100000})
	if err != nil {
		return nil, err
	}
	syms, err := s.ListSymbols(QueryParams{Limit: 100000})
	if err != nil {
		return nil, err
	}
	imps, err := s.AllImports()
	if err != nil {
		return nil, err
	}
	by := map[string]*ModuleFileStats{}
	get := func(path string) *ModuleFileStats {
		if by[path] == nil {
			by[path] = &ModuleFileStats{Path: path}
		}
		return by[path]
	}
	for _, f := range files {
		get(modulePath(f.Path)).FileCount++
	}
	for _, s := range syms {
		get(modulePath(s.File)).SymbolCount++
	}
	for _, i := range imps {
		get(modulePath(i.File)).ImportCount++
	}
	out := make([]ModuleFileStats, 0, len(by))
	for _, v := range by {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// modulePath is intentionally path-only; language package semantics remain
// in the extractor and the overview can remain language-neutral.
func modulePath(path string) string {
	parts := splitPath(path)
	if len(parts) <= 1 {
		return "."
	}
	if parts[0] == "cmd" || parts[0] == "internal" || parts[0] == "pkg" || parts[0] == "src" {
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return parts[0]
}

func splitPath(s string) []string { return strings.Split(s, "/") }
