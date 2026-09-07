package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
)

func init() {
	register([]string{".c", ".h"}, &Language{Name: "c", Extract: extractC})
	register([]string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx", ".ipp"}, &Language{Name: "cpp", Extract: extractCpp})
}

func extractC(src []byte) Result {
	return extractWith("c", c.GetLanguage(), src, cVisit)
}

func extractCpp(src []byte) Result {
	return extractWith("cpp", cpp.GetLanguage(), src, cVisit)
}

func cVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "function_definition":
		name, parent, _ := cFunctionName(w, n)
		if name == "" {
			return ""
		}
		if parent == "" {
			parent = w.cur()
		}
		return addSym(w, n, KindFunction, name, parent, "")

	case "struct_specifier", "union_specifier":
		name := sanitizeIdent(nameOf(w, n))
		if name == "" {
			return "" // anonymous struct used via typedef below
		}
		return addSym(w, n, KindStruct, name, w.cur(), "")

	case "class_specifier":
		name := sanitizeIdent(nameOf(w, n))
		if name == "" {
			return ""
		}
		if base := n.ChildByFieldName("base_class_clause"); base != nil {
			for i := uint32(0); i < base.NamedChildCount(); i++ {
				c := base.NamedChild(int(i))
				if t := strings.Fields(txt(w, c)); len(t) > 0 {
					addRef(w, c, RefExtends, t[len(t)-1], "")
				}
			}
		}
		return addSym(w, n, KindClass, name, w.cur(), "")

	case "enum_specifier":
		name := sanitizeIdent(nameOf(w, n))
		if name == "" {
			return ""
		}
		return addSym(w, n, KindEnum, name, w.cur(), "")

	case "enumerator":
		addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""

	case "type_definition":
		// typedef struct {...} Name; — declarator may be the struct name
		d := n.ChildByFieldName("declarator")
		name := identifierLeaf(w, d)
		if name == "" {
			return ""
		}
		addSym(w, n, KindType, name, w.cur(), "")
		return ""

	case "alias_declaration": // C++ using X = ...;
		name := sanitizeIdent(nameOf(w, n))
		if name == "" {
			return ""
		}
		addSym(w, n, KindType, name, w.cur(), "")
		return ""

	case "field_declarator", "array_declarator":
		// class member fields; only inside field_declaration
		if p := n.Parent(); p == nil || p.Type() != "field_declaration" {
			return ""
		}
		name := identifierLeaf(w, n)
		if name == "" {
			return ""
		}
		addSym(w, n, KindProperty, name, w.cur(), "")
		return ""

	case "declaration":
		// C++ in-class method declarations: class_body > declaration >
		// function_declarator (definitions may live out-of-line).
		if p := n.Parent(); p == nil || p.Type() != "class_body" {
			return ""
		}
		d := n.ChildByFieldName("declarator")
		for d != nil && d.Type() != "function_declarator" {
			d = d.ChildByFieldName("declarator")
		}
		if d == nil {
			return ""
		}
		inner := d.ChildByFieldName("declarator")
		if inner == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, inner))
		name := sanitizeIdent(nm)
		if name == "" {
			return ""
		}
		kind := KindMethod
		if name == w.cur() {
			kind = KindConstructor
		}
		return addSym(w, n, kind, name, w.cur(), q)

	case "call_expression":
		f := n.ChildByFieldName("function")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "preproc_include":
		p := n.ChildByFieldName("path")
		if p == nil {
			return ""
		}
		path := strings.Trim(txt(w, p), "\"<>")
		sl, _ := posOf(n.StartPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: path, StartLine: sl, EndLine: sl})
	}
	return ""
}

// cFunctionName resolves the declared function name from a
// function_definition's (possibly pointer-wrapped) declarator, plus the
// enclosing class for out-of-line C++ method definitions (Foo::bar).
func cFunctionName(w *walkCtx, n *sitter.Node) (name, class, raw string) {
	d := n.ChildByFieldName("declarator")
	for d != nil && d.Type() != "function_declarator" {
		d = d.ChildByFieldName("declarator")
	}
	if d == nil {
		return "", "", ""
	}
	inner := d.ChildByFieldName("declarator")
	if inner == nil {
		return "", "", ""
	}
	raw = txt(w, inner)
	q, nm := splitQualified(raw)
	return sanitizeIdent(nm), q, raw
}
