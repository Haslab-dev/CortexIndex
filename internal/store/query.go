package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetFile returns a file row by path.
func (s *Store) GetFile(path string) (File, bool, error) {
	var f File
	var parsed int
	err := s.db.QueryRow(`SELECT path, language, size, hash, mtime, parsed FROM files WHERE path = ?`, path).
		Scan(&f.Path, &f.Language, &f.Size, &f.Hash, &f.MTime, &parsed)
	if err == sql.ErrNoRows {
		return f, false, nil
	}
	if err != nil {
		return f, false, err
	}
	f.Parsed = parsed == 1
	return f, true, nil
}

// AllFilePaths returns every indexed path with its hash (incremental diff).
func (s *Store) AllFilePaths() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT path, hash FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var p, h string
		if err := rows.Scan(&p, &h); err != nil {
			return nil, err
		}
		out[p] = h
	}
	return out, rows.Err()
}

// Stats reports index sizes for status output.
func (s *Store) Stats() (files, symbols, refs, imports int, err error) {
	type cnt struct {
		n int
	}
	for i, q := range []string{
		`SELECT COUNT(*) FROM files`,
		`SELECT COUNT(*) FROM symbols`,
		`SELECT COUNT(*) FROM refs`,
		`SELECT COUNT(*) FROM imports`,
	} {
		var c cnt
		if err := s.db.QueryRow(q).Scan(&c.n); err != nil {
			return 0, 0, 0, 0, err
		}
		switch i {
		case 0:
			files = c.n
		case 1:
			symbols = c.n
		case 2:
			refs = c.n
		case 3:
			imports = c.n
		}
	}
	return
}

// QueryParams bounds a search call.
type QueryParams struct {
	Limit int
	Kind  string // optional kind filter
	File  string // optional path prefix filter (directory scoping)
}

func (p QueryParams) limit() int {
	if p.Limit <= 0 {
		return 40
	}
	return p.Limit
}

// SearchSymbolsFTS full-text search over symbol names/parts/docs.
func (s *Store) SearchSymbolsFTS(match string, p QueryParams) ([]SymbolRow, error) {
	q := `SELECT s.id, s.file_path, s.name, s.kind, s.signature, s.parent, s.doc,
			s.start_line, s.start_col, s.end_line, s.end_col,
			bm25(symbols_fts, 8.0, 4.0, 1.0, 2.0, 1.0) AS score
		FROM symbols_fts f JOIN symbols s ON s.id = f.rowid
		WHERE symbols_fts MATCH ?`
	args := []any{match}
	if p.Kind != "" {
		q += ` AND s.kind = ?`
		args = append(args, p.Kind)
	}
	if p.File != "" {
		q += ` AND s.file_path LIKE ?`
		args = append(args, p.File+"%")
	}
	q += ` ORDER BY score LIMIT ?`
	args = append(args, p.limit())
	return s.symbolRows(q, args...)
}

// SearchSymbolsExact finds symbols whose name equals the query
// (case-insensitive), preferring shorter names and method matches.
func (s *Store) SearchSymbolsExact(name string, p QueryParams) ([]SymbolRow, error) {
	q := `SELECT id, file_path, name, kind, signature, parent, doc,
			start_line, start_col, end_line, end_col, 0
		FROM symbols WHERE lower(name) = lower(?)`
	args := []any{name}
	if p.Kind != "" {
		q += ` AND kind = ?`
		args = append(args, p.Kind)
	}
	if p.File != "" {
		q += ` AND file_path LIKE ?`
		args = append(args, p.File+"%")
	}
	q += ` ORDER BY length(name), file_path LIMIT ?`
	args = append(args, p.limit())
	return s.symbolRows(q, args...)
}

// SearchSymbolsPrefix finds symbols whose name starts with the query.
func (s *Store) SearchSymbolsPrefix(prefix string, p QueryParams) ([]SymbolRow, error) {
	q := `SELECT id, file_path, name, kind, signature, parent, doc,
			start_line, start_col, end_line, end_col, 0
		FROM symbols WHERE lower(name) LIKE lower(?) || '%'`
	args := []any{prefix}
	if p.Kind != "" {
		q += ` AND kind = ?`
		args = append(args, p.Kind)
	}
	if p.File != "" {
		q += ` AND file_path LIKE ?`
		args = append(args, p.File+"%")
	}
	q += ` ORDER BY length(name), name, file_path LIMIT ?`
	args = append(args, p.limit())
	return s.symbolRows(q, args...)
}

// GetSymbolByID loads one symbol row.
func (s *Store) GetSymbolByID(id int64) (SymbolRow, error) {
	rows, err := s.symbolRows(`SELECT id, file_path, name, kind, signature, parent, doc,
			start_line, start_col, end_line, end_col, 0
		FROM symbols WHERE id = ?`, id)
	if err != nil {
		return SymbolRow{}, err
	}
	if len(rows) == 0 {
		return SymbolRow{}, fmt.Errorf("symbol %d not found", id)
	}
	return rows[0], nil
}

func (s *Store) symbolRows(q string, args ...any) ([]SymbolRow, error) {
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SymbolRow
	for rows.Next() {
		var r SymbolRow
		if err := rows.Scan(&r.ID, &r.File, &r.Name, &r.Kind, &r.Signature, &r.Parent, &r.Doc,
			&r.StartLine, &r.StartCol, &r.EndLine, &r.EndCol, new(float64)); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SearchFilesFTS full-text search over file contents with snippets.
func (s *Store) SearchFilesFTS(match string, p QueryParams) ([]FileHit, error) {
	q := `SELECT path, snippet(files_fts, 1, '[', ']', '…', 12),
			bm25(files_fts) AS score
		FROM files_fts WHERE files_fts MATCH ?`
	args := []any{match}
	if p.File != "" {
		q += ` AND path LIKE ?`
		args = append(args, p.File+"%")
	}
	q += ` ORDER BY score LIMIT ?`
	args = append(args, p.limit())
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileHit
	for rows.Next() {
		var h FileHit
		if err := rows.Scan(&h.Path, &h.Snippet, &h.Score); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// RefsByName returns references to the given name across the repo,
// resolving the enclosing symbol display name when possible.
func (s *Store) RefsByName(name string, limit int) ([]RefRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT r.id, r.file_path, r.name, r.qualifier, r.kind, r.line, r.col,
			COALESCE(r.from_symbol_id, 0),
			COALESCE(sym.name, ''),
			COALESCE(sym.parent, '')
		FROM refs r
		LEFT JOIN symbols sym ON sym.id = r.from_symbol_id
		WHERE r.name = ?
		ORDER BY r.file_path, r.line
		LIMIT ?`, name, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

// RefsFromSymbol returns references made inside a symbol's body.
func (s *Store) RefsFromSymbol(symbolID int64) ([]RefRow, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.file_path, r.name, r.qualifier, r.kind, r.line, r.col,
			r.from_symbol_id,
			COALESCE(sym.name, ''),
			COALESCE(sym.parent, '')
		FROM refs r
		LEFT JOIN symbols sym ON sym.id = r.from_symbol_id
		WHERE r.from_symbol_id = ?
		ORDER BY r.line`, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRefs(rows)
}

func scanRefs(rows *sql.Rows) ([]RefRow, error) {
	var out []RefRow
	for rows.Next() {
		var r RefRow
		var parent, name string
		if err := rows.Scan(&r.ID, &r.File, &r.Name, &r.Qualifier, &r.Kind, &r.Line, &r.Col,
			&r.FromID, &name, &parent); err != nil {
			return nil, err
		}
		if name != "" {
			if parent != "" {
				r.FromName = parent + "." + name
			} else {
				r.FromName = name
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ImportsForFile returns the imports recorded for a file.
func (s *Store) ImportsForFile(path string) ([]ImportRow, error) {
	rows, err := s.db.Query(`SELECT file_path, module, names, line FROM imports WHERE file_path = ? ORDER BY line`, path)
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
			r.Names = strings.Split(names, ",")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ChunkForSymbol returns the stored source text of a symbol, if any.
func (s *Store) ChunkForSymbol(symbolID int64) (string, bool, error) {
	var content string
	err := s.db.QueryRow(`SELECT content FROM chunks WHERE symbol_id = ?`, symbolID).Scan(&content)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return content, true, nil
}

// FileCountUnder counts indexed files whose path starts with prefix.
func (s *Store) FileCountUnder(prefix string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE path LIKE ? || '%'`, prefix).Scan(&n)
	return n, err
}
