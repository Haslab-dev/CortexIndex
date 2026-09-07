package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

func init() {
	register([]string{".py", ".pyi"}, &Language{Name: "python", Extract: extractPython})
}

func extractPython(src []byte) Result {
	return extractWith("python", python.GetLanguage(), src, pyVisit)
}

func pyVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "class_definition":
		name := sanitizeIdent(nameOf(w, n))
		if sc := n.ChildByFieldName("superclasses"); sc != nil {
			for i := uint32(0); i < sc.NamedChildCount(); i++ {
				c := sc.NamedChild(int(i))
				if c.Type() == "identifier" || c.Type() == "attribute" {
					_, nm := splitQualified(txt(w, c))
					addRef(w, c, RefExtends, sanitizeIdent(nm), "")
				}
			}
		}
		return addSym(w, n, KindClass, name, w.cur(), "")

	case "function_definition":
		name := sanitizeIdent(nameOf(w, n))
		d := addSym(w, n, KindFunction, name, w.cur(), "")
		if d != "" {
			pyDocstring(w, n)
		}
		return d

	case "call":
		f := n.ChildByFieldName("function")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "assignment":
		if w.cur() != "" {
			return "" // locals are noise
		}
		l := n.ChildByFieldName("left")
		if l == nil || l.Type() != "identifier" {
			return ""
		}
		addSym(w, n, KindVariable, l.Content(w.src), "", "")
		return ""

	case "import_statement":
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			switch c.Type() {
			case "dotted_name": // plain `import a.b.c`
				sl, _ := posOf(c.StartPoint())
				w.res.Imports = append(w.res.Imports, Import{Path: txt(w, c), StartLine: sl, EndLine: sl})
			case "aliased_import": // `import a.b as x`
				path := ""
				if dn := c.ChildByFieldName("name"); dn != nil {
					path = txt(w, dn)
				}
				w.res.Imports = append(w.res.Imports, Import{Path: path})
			}
		}

	case "import_from_statement":
		path := txt(w, n.ChildByFieldName("module_name"))
		modName := n.ChildByFieldName("module_name")
		var names []string
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			if modName != nil && c.ID() == modName.ID() {
				continue
			}
			switch c.Type() {
			case "dotted_name":
				names = append(names, txt(w, c))
			case "aliased_import":
				if nn := c.ChildByFieldName("name"); nn != nil {
					names = append(names, txt(w, nn))
				}
			}
		}
		sl, _ := posOf(n.StartPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: path, Names: names, StartLine: sl, EndLine: sl})
	}
	return ""
}

// pyDocstring attaches the function/class docstring to the most recently
// added symbol when it has no comment-style doc already.
func pyDocstring(w *walkCtx, n *sitter.Node) {
	body := n.ChildByFieldName("body") // this IS the block node
	if body == nil || body.Type() != "block" {
		return
	}
	stmt := body.NamedChild(0)
	if stmt == nil || stmt.Type() != "expression_statement" {
		return
	}
	s := stmt.NamedChild(0)
	if s == nil || s.Type() != "string" {
		return
	}
	doc := strings.TrimSpace(txt(w, s))
	doc = strings.TrimPrefix(doc, "r")
	doc = strings.TrimPrefix(doc, "b")
	doc = strings.Trim(doc, "\"'")
	if len(doc) > 1500 {
		doc = doc[:1500] + "…"
	}
	if i := len(w.res.Symbols) - 1; i >= 0 && w.res.Symbols[i].Doc == "" {
		w.res.Symbols[i].Doc = doc
	}
}
