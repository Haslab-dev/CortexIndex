// Package lang defines the common extraction model shared by every
// tree-sitter language extractor, plus language detection by file extension.
package lang

import (
	"path/filepath"
	"strings"
)

// Kind is the coarse symbol classification used across the index.
type Kind string

const (
	KindFunction    Kind = "function"
	KindMethod      Kind = "method"
	KindClass       Kind = "class"
	KindInterface   Kind = "interface"
	KindStruct      Kind = "struct"
	KindEnum        Kind = "enum"
	KindType        Kind = "type"
	KindTrait       Kind = "trait"
	KindImpl        Kind = "impl"
	KindConstant    Kind = "constant"
	KindVariable    Kind = "variable"
	KindModule      Kind = "module"
	KindProperty    Kind = "property"
	KindConstructor Kind = "constructor"
)

// RefKind classifies a relationship captured in source.
type RefKind string

const (
	RefCall       RefKind = "call"
	RefNew        RefKind = "new"
	RefExtends    RefKind = "extends"
	RefImplements RefKind = "implements"
)

// Symbol is a named definition extracted from source.
// Lines are 1-based; columns are 0-based (tree-sitter native).
type Symbol struct {
	Name      string
	Kind      Kind
	Parent    string // enclosing class/interface/impl, "" when top-level
	Qualifier string // e.g. trait for a Rust impl, receiver alias info
	Signature string // compact header line
	Doc       string // leading doc comment / docstring
	Chunk     string // source text of the symbol when small enough

	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// Display returns the qualified display name, e.g. "Server.Handle".
func (s Symbol) Display() string {
	if s.Parent != "" {
		return s.Parent + "." + s.Name
	}
	return s.Name
}

// Ref is a reference to a name somewhere in a file (call target,
// inheritance edge, ...). InSymbol is the innermost containing symbol's
// display name within the same file, "" at top level.
type Ref struct {
	Name      string
	Qualifier string
	Kind      RefKind
	InSymbol  string

	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// Import is a single import/use/include statement.
type Import struct {
	Path  string
	Names []string

	StartLine int
	EndLine   int
}

// Result is everything extracted from one parsed file.
type Result struct {
	Symbols []Symbol
	Refs    []Ref
	Imports []Import
}

// Language describes one supported source language.
type Language struct {
	Name    string
	Extract func(src []byte) Result
}

var byExt = map[string]*Language{}

func register(exts []string, l *Language) {
	for _, e := range exts {
		byExt[e] = l
	}
}

// Detect returns the language for a file path, or nil when the extension
// is not a parseable source language.
func Detect(path string) *Language {
	ext := strings.ToLower(filepath.Ext(path))
	return byExt[ext]
}

// IsTextExt reports whether the extension belongs to a textual,
// non-source file that should be content-indexed only (no AST parsing).
func IsTextExt(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".rst",
		".json", ".jsonc", ".yaml", ".yml", ".toml", ".ini", ".cfg",
		".sql", ".css", ".scss", ".html", ".htm", ".xml", ".svg",
		".sh", ".bash", ".zsh", ".env", ".gitignore", ".editorconfig",
		".mod", ".sum", ".lock", ".plist", ".properties", ".proto":
		return true
	}
	return false
}

// NameParts splits an identifier into lowercase search parts:
// camelCase, PascalCase, snake_case and dot/colon separators all split.
func NameParts(name string) []string {
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_' || r == '.' || r == ':' || r == '$' || r == '-' || r == ' ':
			flush()
		case r >= 'A' && r <= 'Z':
			// New camel word when the previous rune is lowercase/digit, or
			// when an uppercase run ends here (next rune is lowercase):
			// "AuthService" → auth|service, "HTTPServer" → http|server.
			if i > 0 {
				prev := runes[i-1]
				prevLower := (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9')
				nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
				if prevLower || nextLower {
					flush()
				}
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return parts
}
