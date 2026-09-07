// Package store owns all persistence for Cortex: the derived SQLite index
// (files, symbols, refs, imports, chunks) and its FTS5 search tables.
// Everything here is rebuildable state — Markdown memory lives elsewhere.
package store

import (
	"database/sql"
	"fmt"
	"strings"

	"cortex/internal/lang"

	_ "modernc.org/sqlite"
)

const schemaVersion = "1"

// File describes one indexed file.
type File struct {
	Path     string // repo-relative, slash-separated
	Language string
	Size     int64
	Hash     string // sha256 hex of content
	MTime    int64  // unix nanos
	Parsed   bool   // true when tree-sitter extraction ran
}

// SymbolRow is an indexed symbol with its database identity.
type SymbolRow struct {
	ID        int64
	File      string
	Name      string
	Kind      string
	Signature string
	Parent    string
	Doc       string
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int

	Chunk string `json:"-"`
}

// Display returns "Parent.Name" or "Name".
func (s SymbolRow) Display() string {
	if s.Parent != "" {
		return s.Parent + "." + s.Name
	}
	return s.Name
}

// RefRow is an indexed reference (call/new/extends/implements).
type RefRow struct {
	ID        int64
	File      string
	Name      string
	Qualifier string
	Kind      string
	Line      int
	Col       int
	FromID    int64
	FromName  string // display name of the enclosing symbol, resolved on read
}

// ImportRow is an indexed import statement.
type ImportRow struct {
	File  string
	Path  string
	Names []string
	Line  int
}

// FileHit is a lexical (FTS) match over file contents.
type FileHit struct {
	Path    string
	Snippet string
	Score   float64
}

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// Open creates (or opens) the index database at path, ensuring the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) initSchema() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS files (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			path     TEXT NOT NULL UNIQUE,
			language TEXT NOT NULL,
			size     INTEGER NOT NULL,
			hash     TEXT NOT NULL,
			mtime    INTEGER NOT NULL,
			parsed   INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS symbols (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path   TEXT NOT NULL,
			name        TEXT NOT NULL,
			kind        TEXT NOT NULL,
			signature   TEXT NOT NULL DEFAULT '',
			parent      TEXT NOT NULL DEFAULT '',
			doc         TEXT NOT NULL DEFAULT '',
			start_line  INTEGER NOT NULL,
			start_col   INTEGER NOT NULL,
			end_line    INTEGER NOT NULL,
			end_col     INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols(kind)`,
		`CREATE TABLE IF NOT EXISTS refs (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path      TEXT NOT NULL,
			name           TEXT NOT NULL,
			qualifier      TEXT NOT NULL DEFAULT '',
			kind           TEXT NOT NULL,
			line           INTEGER NOT NULL,
			col            INTEGER NOT NULL,
			from_symbol_id INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_name ON refs(name)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_file ON refs(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_from ON refs(from_symbol_id)`,
		`CREATE TABLE IF NOT EXISTS imports (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			file_path TEXT NOT NULL,
			module    TEXT NOT NULL,
			names     TEXT NOT NULL DEFAULT '',
			line      INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_imports_file ON imports(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_imports_module ON imports(module)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			symbol_id  INTEGER PRIMARY KEY,
			file_path  TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line   INTEGER NOT NULL,
			content    TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS memory_metadata (
			path      TEXT PRIMARY KEY,
			hash      TEXT NOT NULL,
			synced_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
			name, parts, kind, file_path, doc
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(path, content)`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO meta(key, value) VALUES ('schema_version', ?)`, schemaVersion); err != nil {
		return err
	}
	return tx.Commit()
}

// Wipe deletes all derived rows (used by `index --full`).
func (s *Store) Wipe() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range []string{"symbols_fts", "files_fts", "chunks", "imports", "refs", "symbols", "files"} {
		if _, err := tx.Exec("DELETE FROM " + t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReplaceFile atomically replaces all derived data for one file.
// parsed=false means the file was content-indexed only (parse failed,
// unknown language, or text file) — symbols/refs/imports are empty then.
func (s *Store) ReplaceFile(f File, content string, res lang.Result) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove previous rows for this path.
	var oldSymIDs []int64
	rows, err := tx.Query(`SELECT id FROM symbols WHERE file_path = ?`, f.Path)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		oldSymIDs = append(oldSymIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range oldSymIDs {
		if _, err := tx.Exec(`DELETE FROM symbols_fts WHERE rowid = ?`, id); err != nil {
			return err
		}
	}
	for _, q := range []string{
		`DELETE FROM symbols WHERE file_path = ?`,
		`DELETE FROM chunks WHERE file_path = ?`,
		`DELETE FROM refs WHERE file_path = ?`,
		`DELETE FROM imports WHERE file_path = ?`,
	} {
		if _, err := tx.Exec(q, f.Path); err != nil {
			return err
		}
	}
	var oldFileID int64
	hadFile := false
	if err := tx.QueryRow(`SELECT id FROM files WHERE path = ?`, f.Path).Scan(&oldFileID); err == nil {
		hadFile = true
	} else if err != sql.ErrNoRows {
		return err
	}
	if hadFile {
		if _, err := tx.Exec(`DELETE FROM files_fts WHERE rowid = ?`, oldFileID); err != nil {
			return err
		}
		if _, err := tx.Exec(`DELETE FROM files WHERE id = ?`, oldFileID); err != nil {
			return err
		}
	}

	// Insert file row + content FTS.
	parsed := 0
	if f.Parsed {
		parsed = 1
	}
	res2, err := tx.Exec(`INSERT INTO files(path, language, size, hash, mtime, parsed) VALUES(?,?,?,?,?,?)`,
		f.Path, f.Language, f.Size, f.Hash, f.MTime, parsed)
	if err != nil {
		return fmt.Errorf("insert file %s: %w", f.Path, err)
	}
	fileID, err := res2.LastInsertId()
	if err != nil {
		return err
	}
	if content != "" {
		if _, err := tx.Exec(`INSERT INTO files_fts(rowid, path, content) VALUES(?,?,?)`, fileID, f.Path, content); err != nil {
			return err
		}
	}

	// Insert symbols (collect ranges for ref containment resolution).
	var spans []symSpan
	insSym, err := tx.Prepare(`INSERT INTO symbols(file_path, name, kind, signature, parent, doc, start_line, start_col, end_line, end_col) VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insSym.Close()
	for _, sym := range res.Symbols {
		r, err := insSym.Exec(f.Path, sym.Name, string(sym.Kind), sym.Signature, sym.Parent, sym.Doc,
			sym.StartLine, sym.StartCol, sym.EndLine, sym.EndCol)
		if err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.Display(), err)
		}
		id, err := r.LastInsertId()
		if err != nil {
			return err
		}
		parts := strings.Join(lang.NameParts(sym.Name), " ")
		if _, err := tx.Exec(`INSERT INTO symbols_fts(rowid, name, parts, kind, file_path, doc) VALUES(?,?,?,?,?,?)`,
			id, sym.Name, parts, string(sym.Kind), f.Path, sym.Doc); err != nil {
			return err
		}
		spans = append(spans, symSpan{id: id, start: sym.StartLine, end: sym.EndLine, displayName: sym.Display()})
		if sym.Chunk != "" {
			if _, err := tx.Exec(`INSERT OR REPLACE INTO chunks(symbol_id, file_path, start_line, end_line, content) VALUES(?,?,?,?,?)`,
				id, f.Path, sym.StartLine, sym.EndLine, sym.Chunk); err != nil {
				return err
			}
		}
	}

	// Insert refs with innermost-containing-symbol resolution.
	insRef, err := tx.Prepare(`INSERT INTO refs(file_path, name, qualifier, kind, line, col, from_symbol_id) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insRef.Close()
	for _, r := range res.Refs {
		var fromID any
		if best := innermost(spans, r.StartLine); best != nil {
			fromID = best.id
		}
		if _, err := insRef.Exec(f.Path, r.Name, r.Qualifier, string(r.Kind), r.StartLine, r.StartCol, fromID); err != nil {
			return err
		}
	}

	// Insert imports.
	insImp, err := tx.Prepare(`INSERT INTO imports(file_path, module, names, line) VALUES(?,?,?,?)`)
	if err != nil {
		return err
	}
	defer insImp.Close()
	for _, imp := range res.Imports {
		names := strings.Join(imp.Names, ",")
		if _, err := insImp.Exec(f.Path, imp.Path, names, imp.StartLine); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// symSpan is a symbol's line range used to attribute refs to their
// innermost enclosing symbol.
type symSpan struct {
	id          int64
	start, end  int
	displayName string
}

// innermost returns the smallest symbol span containing line.
func innermost(spans []symSpan, line int) *symSpan {
	var best *symSpan
	for i := range spans {
		sp := &spans[i]
		if line >= sp.start && line <= sp.end {
			if best == nil || (sp.end-sp.start) < (best.end-best.start) {
				best = sp
			}
		}
	}
	return best
}

// RemoveFile deletes every trace of a file (deletion sweep).
func (s *Store) RemoveFile(path string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM symbols_fts WHERE rowid IN (SELECT id FROM symbols WHERE file_path = ?)`, path); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM symbols WHERE file_path = ?`,
		`DELETE FROM chunks WHERE file_path = ?`,
		`DELETE FROM refs WHERE file_path = ?`,
		`DELETE FROM imports WHERE file_path = ?`,
	} {
		if _, err := tx.Exec(q, path); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM files_fts WHERE rowid IN (SELECT id FROM files WHERE path = ?)`, path); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM files WHERE path = ?`, path); err != nil {
		return err
	}
	return tx.Commit()
}
