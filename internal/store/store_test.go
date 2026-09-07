package store

import (
	"path/filepath"
	"testing"

	"cortex/internal/lang"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func goResult() lang.Result {
	return lang.Result{
		Symbols: []lang.Symbol{
			{Name: "Server", Kind: lang.KindStruct, Signature: "type Server struct", Doc: "serves auth", StartLine: 3, StartCol: 0, EndLine: 5, EndCol: 1},
			{Name: "Login", Kind: lang.KindMethod, Parent: "Server", Signature: "func (s *Server) Login(user string) error", StartLine: 7, StartCol: 0, EndLine: 10, EndCol: 1},
			{Name: "NewServer", Kind: lang.KindFunction, Signature: "func NewServer() *Server", StartLine: 12, StartCol: 0, EndLine: 12, EndCol: 30},
		},
		Refs: []lang.Ref{
			{Name: "Issue", Qualifier: "s.store", Kind: lang.RefCall, InSymbol: "Server.Login", StartLine: 8, StartCol: 14},
			{Name: "ToUpper", Qualifier: "strings", Kind: lang.RefCall, InSymbol: "Server.Login", StartLine: 9, StartCol: 2},
			{Name: "Server", Kind: lang.RefNew, StartLine: 12, StartCol: 24},
		},
		Imports: []lang.Import{
			{Path: "fmt", StartLine: 1, EndLine: 1},
			{Path: "strings", Names: []string{"str"}, StartLine: 2, EndLine: 2},
		},
	}
}

func TestReplaceFileAndQuery(t *testing.T) {
	s := openTestStore(t)
	f := File{Path: "internal/auth/server.go", Language: "go", Size: 100, Hash: "abc", MTime: 1, Parsed: true}
	if err := s.ReplaceFile(f, "package auth\n// content here", goResult()); err != nil {
		t.Fatalf("replace: %v", err)
	}

	// Exact name lookup.
	rows, err := s.SearchSymbolsExact("login", QueryParams{})
	if err != nil {
		t.Fatalf("exact: %v", err)
	}
	if len(rows) != 1 || rows[0].Display() != "Server.Login" {
		t.Fatalf("exact rows = %+v", rows)
	}
	if rows[0].File != "internal/auth/server.go" || rows[0].StartLine != 7 {
		t.Errorf("row metadata wrong: %+v", rows[0])
	}

	// FTS by camel part ("serv" is a part of "Server").
	rows, err = s.SearchSymbolsFTS(`"serv"*`, QueryParams{})
	if err != nil {
		t.Fatalf("fts: %v", err)
	}
	if len(rows) == 0 {
		t.Errorf("FTS should find Server via parts")
	}

	// File content FTS.
	hits, err := s.SearchFilesFTS(`"content"`, QueryParams{})
	if err != nil {
		t.Fatalf("file fts: %v", err)
	}
	if len(hits) != 1 || hits[0].Path != "internal/auth/server.go" {
		t.Fatalf("file hits = %+v", hits)
	}
	if hits[0].Snippet == "" {
		t.Errorf("snippet should be populated")
	}

	// Refs by name include enclosing symbol.
	refs, err := s.RefsByName("Issue", 0)
	if err != nil {
		t.Fatalf("refs: %v", err)
	}
	if len(refs) != 1 || refs[0].FromName != "Server.Login" {
		t.Fatalf("refs = %+v", refs)
	}
	if refs[0].Kind != string(lang.RefCall) || refs[0].Line != 8 {
		t.Errorf("ref metadata wrong: %+v", refs[0])
	}

	// Imports.
	imps, err := s.ImportsForFile("internal/auth/server.go")
	if err != nil || len(imps) != 2 || imps[0].Path != "fmt" {
		t.Fatalf("imports = %+v err=%v", imps, err)
	}

	// Chunks: re-replace with chunk data, then read it back via a fresh lookup.
	res := goResult()
	res.Symbols[1].Chunk = "func (s *Server) Login(user string) error { return nil }"
	if err := s.ReplaceFile(f, "package auth\n// content here", res); err != nil {
		t.Fatalf("replace 2: %v", err)
	}
	fresh, err := s.SearchSymbolsExact("login", QueryParams{})
	if err != nil || len(fresh) == 0 {
		t.Fatalf("fresh: %v %v", fresh, err)
	}
	chunk, ok, err := s.ChunkForSymbol(fresh[0].ID)
	if err != nil || !ok || chunk == "" {
		t.Fatalf("chunk ok=%v err=%v", ok, err)
	}
}

func TestReplaceFileIsIdempotent(t *testing.T) {
	s := openTestStore(t)
	f := File{Path: "a.go", Language: "go", Size: 10, Hash: "h", MTime: 1, Parsed: true}
	for i := 0; i < 3; i++ {
		if err := s.ReplaceFile(f, "package a", goResult()); err != nil {
			t.Fatalf("replace %d: %v", i, err)
		}
	}
	files, symbols, refs, imports, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if files != 1 || symbols != 3 || refs != 3 || imports != 2 {
		t.Errorf("counts after 3 replaces: files=%d symbols=%d refs=%d imports=%d", files, symbols, refs, imports)
	}
	// FTS rows track symbols 1:1 (a duplicate would surface in searches).
	got, err := s.SearchSymbolsExact("login", QueryParams{})
	if err != nil || len(got) != 1 {
		t.Errorf("exact search after re-replace: %d rows err=%v", len(got), err)
	}
}

func TestRemoveFileCleansEverything(t *testing.T) {
	s := openTestStore(t)
	f := File{Path: "a.go", Language: "go", Size: 10, Hash: "h", MTime: 1, Parsed: true}
	if err := s.ReplaceFile(f, "package a", goResult()); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveFile("a.go"); err != nil {
		t.Fatal(err)
	}
	files, symbols, refs, imports, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if files != 0 || symbols != 0 || refs != 0 || imports != 0 {
		t.Errorf("orphans after remove: files=%d symbols=%d refs=%d imports=%d", files, symbols, refs, imports)
	}
	hits, err := s.SearchFilesFTS(`"package"`, QueryParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Errorf("files_fts orphans: %+v", hits)
	}
}

func TestAllFilePathsAndWipe(t *testing.T) {
	s := openTestStore(t)
	for _, p := range []string{"a.go", "b.go"} {
		f := File{Path: p, Language: "go", Size: 1, Hash: p, MTime: 1, Parsed: false}
		if err := s.ReplaceFile(f, "package a", lang.Result{}); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := s.AllFilePaths()
	if err != nil || len(paths) != 2 {
		t.Fatalf("paths = %v err=%v", paths, err)
	}
	if err := s.Wipe(); err != nil {
		t.Fatal(err)
	}
	files, _, _, _, err := s.Stats()
	if err != nil || files != 0 {
		t.Fatalf("after wipe files=%d err=%v", files, err)
	}
}

func TestSanitizeFTS(t *testing.T) {
	cases := map[string]string{
		"AuthService.login": `"AuthService" OR "login"`,
		"how does auth":     `"how" OR "does" OR "auth"`,
		"":                  "",
		"  single  ":        `"single"`,
	}
	for in, want := range cases {
		if got := SanitizeFTS(in); got != want {
			t.Errorf("SanitizeFTS(%q) = %q, want %q", in, got, want)
		}
	}
}
