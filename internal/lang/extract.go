package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// walkCtx carries extraction state through one file's tree walk.
type walkCtx struct {
	src   []byte
	res   *Result
	stack []string // innermost enclosing symbol display names
}

func (w *walkCtx) cur() string {
	if len(w.stack) == 0 {
		return ""
	}
	return w.stack[len(w.stack)-1]
}

func (w *walkCtx) push(name string) { w.stack = append(w.stack, name) }
func (w *walkCtx) pop()             { w.stack = w.stack[:len(w.stack)-1] }

// visitFn returns the symbol display name to push onto the stack while
// visiting this node's children, or "" when the node defines no symbol.
type visitFn func(n *sitter.Node, w *walkCtx) string

func walk(n *sitter.Node, w *walkCtx, visit visitFn) {
	if n == nil || n.IsNull() {
		return
	}
	pushed := visit(n, w)
	if pushed != "" {
		w.push(pushed)
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		walk(n.NamedChild(int(i)), w, visit)
	}
	if pushed != "" {
		w.pop()
	}
}

// ---- shared node helpers ----

func txt(w *walkCtx, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return n.Content(w.src)
}

func posOf(p sitter.Point) (line, col int) {
	return int(p.Row) + 1, int(p.Column)
}

// nameOf returns the content of the node's "name" field.
func nameOf(w *walkCtx, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	c := n.ChildByFieldName("name")
	if c == nil {
		return ""
	}
	return c.Content(w.src)
}

// identifierLeaf finds the identifier-ish text of a declarator subtree,
// unwrapping pointer/reference/function declarators.
func identifierLeaf(w *walkCtx, n *sitter.Node) string {
	for depth := 0; n != nil && depth < 16; depth++ {
		switch n.Type() {
		case "identifier", "field_identifier", "property_identifier",
			"type_identifier", "simple_identifier":
			return n.Content(w.src)
		case "function_declarator", "pointer_declarator", "init_declarator",
			"parenthesized_declarator", "array_declarator", "reference_declarator",
			"qualified_identifier", "destructuring_pattern", "attributed_declarator",
			"field_declarator":
			if c := n.ChildByFieldName("declarator"); c != nil {
				n = c
				continue
			}
			// qualified_identifier: use last segment
			var last *sitter.Node
			for i := uint32(0); i < n.NamedChildCount(); i++ {
				c := n.NamedChild(int(i))
				if c.Type() == "identifier" || c.Type() == "type_identifier" {
					last = c
				}
			}
			if last != nil {
				return last.Content(w.src)
			}
			n = n.NamedChild(0)
		case "declaration", "field_declaration", "variable_declarator":
			if c := n.ChildByFieldName("declarator"); c != nil {
				n = c
				continue
			}
			return ""
		default:
			return ""
		}
	}
	return ""
}

// addSym appends a symbol with derived signature/doc/chunk.
func addSym(w *walkCtx, n *sitter.Node, kind Kind, name, parent, qualifier string) string {
	if name == "" || n == nil {
		return ""
	}
	sl, sc := posOf(n.StartPoint())
	el, ec := posOf(n.EndPoint())
	s := Symbol{
		Name:      name,
		Kind:      kind,
		Parent:    parent,
		Qualifier: qualifier,
		Doc:       docAbove(w, n),
		StartLine: sl, StartCol: sc,
		EndLine: el, EndCol: ec,
	}
	s.Signature = signatureOf(w, n)
	if int(n.EndByte()-n.StartByte()) <= maxChunkBytes {
		s.Chunk = strings.TrimSpace(string(w.src[n.StartByte():n.EndByte()]))
	}
	w.res.Symbols = append(w.res.Symbols, s)
	if s.Parent != "" {
		return s.Parent + "." + s.Name
	}
	return s.Name
}

const maxChunkBytes = 4096

// signatureOf renders a compact one-line header for the node:
// source text up to the first '{', ';' or ':' at the top level,
// whitespace-collapsed and length-capped.
func signatureOf(w *walkCtx, n *sitter.Node) string {
	start := n.StartByte()
	end := n.EndByte()
	src := w.src[start:end]
	cut := len(src)
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '{', ';':
			cut = i
			goto done
		case '\n':
			// keep newlines only until the body starts; a blank line ends it
			if i+1 < len(src) && src[i+1] == '\n' {
				cut = i
				goto done
			}
		case ':':
			// Python-style bodies start after ':' — but only a label colon
			// followed by whitespace; "::" is a C++/Rust scope separator.
			if i+1 < len(src) && src[i+1] != ':' && (src[i+1] == ' ' || src[i+1] == '\t' || src[i+1] == '\n' || src[i+1] == '\r') {
				cut = i
				goto done
			}
		}
	}
done:
	sig := strings.TrimSpace(string(src[:cut]))
	sig = strings.Join(strings.Fields(sig), " ")
	if len(sig) > 200 {
		sig = sig[:200] + "…"
	}
	return sig
}

var commentNodeTypes = map[string]bool{
	"comment":        true, // go, js/ts, python, c, cpp
	"line_comment":   true, // rust, java, kotlin
	"block_comment":  true, // rust, java, kotlin
	"doc_comment":    true, // kotlin
	"documentation":  true, // some grammars
}

// docAbove collects contiguous comment lines immediately above the node.
func docAbove(w *walkCtx, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	expected := int(n.StartPoint().Row) - 1 // 0-based row the previous comment must end on
	var lines []string
	for sib := n.PrevSibling(); sib != nil && len(lines) < 60; sib = sib.PrevSibling() {
		if !commentNodeTypes[sib.Type()] {
			break
		}
		// Some grammars (e.g. Rust) swallow the trailing newline into the
		// comment token, pushing its end row past the text it covers.
		content := sib.Content(w.src)
		endRow := int(sib.EndPoint().Row)
		if trimmed := strings.TrimRight(content, "\r\n"); len(trimmed) != len(content) {
			endRow -= len(content) - len(trimmed)
		}
		if endRow != expected {
			break
		}
		var block []string
		for _, ln := range strings.Split(sib.Content(w.src), "\n") {
			ln = strings.TrimSpace(ln)
			ln = strings.TrimPrefix(ln, "//")
			ln = strings.TrimPrefix(ln, "/**")
			ln = strings.TrimPrefix(ln, "/*")
			ln = strings.TrimPrefix(ln, "*")
			ln = strings.TrimPrefix(ln, "/")
			ln = strings.TrimSuffix(ln, "*/")
			ln = strings.TrimSpace(ln)
			if ln != "" {
				block = append(block, ln)
			}
		}
		lines = append(block, lines...)
		expected = int(sib.StartPoint().Row) - 1
	}
	// When the node itself is a spec inside a declaration, the doc comment
	// may be attached to the declaration one level up.
	if len(lines) == 0 && n.Parent() != nil {
		switch n.Parent().Type() {
		case "type_declaration", "const_declaration", "var_declaration",
			"declaration", "lexical_declaration", "variable_declaration":
			return docAbove(w, n.Parent())
		}
	}
	doc := strings.Join(lines, "\n")
	if len(doc) > 1500 {
		doc = doc[:1500] + "…"
	}
	return doc
}

// addRef records a reference at the given node's position.
func addRef(w *walkCtx, n *sitter.Node, kind RefKind, name, qualifier string) {
	if name == "" {
		return
	}
	sl, sc := posOf(n.StartPoint())
	el, ec := posOf(n.EndPoint())
	w.res.Refs = append(w.res.Refs, Ref{
		Name: name, Qualifier: qualifier, Kind: kind,
		InSymbol:  w.cur(),
		StartLine: sl, StartCol: sc, EndLine: el, EndCol: ec,
	})
}

// splitQualified splits "a.b.c", "a::b" or "p->m" into qualifier and final
// name. A leading qualifier of "this", "self" or "super" is kept — it tells
// the retrieval engine the call likely targets the enclosing type.
func splitQualified(s string) (qualifier, name string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	sep, sepLen := -1, 1
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '.' || s[i] == '#':
			sep, sepLen = i, 1
		case i+1 < len(s) && s[i] == ':' && s[i+1] == ':':
			sep, sepLen = i, 2
		case i+1 < len(s) && s[i] == '-' && s[i+1] == '>':
			sep, sepLen = i, 2
		}
	}
	if sep <= 0 || sep+sepLen >= len(s) {
		return "", s
	}
	return s[:sep], s[sep+sepLen:]
}

// sanitizeIdent strips decorations from an extracted name.
func sanitizeIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`")
	if i := strings.IndexAny(s, "(<"); i > 0 {
		s = s[:i]
	}
	return s
}
