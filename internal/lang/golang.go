package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func init() {
	register([]string{".go"}, &Language{Name: "go", Extract: extractGo})
}

func extractGo(src []byte) Result {
	return extractWith("go", golang.GetLanguage(), src, goVisit)
}

func goVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "function_declaration":
		return addSym(w, n, KindFunction, sanitizeIdent(nameOf(w, n)), "", "")

	case "method_declaration":
		return addSym(w, n, KindMethod, sanitizeIdent(nameOf(w, n)), goReceiverType(w, n), "")

	case "method_spec", "method_elem":
		// interface method signatures (grammar versions differ)
		return addSym(w, n, KindMethod, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "type_spec":
		name := sanitizeIdent(nameOf(w, n))
		kind := KindType
		if t := n.ChildByFieldName("type"); t != nil {
			switch t.Type() {
			case "struct_type":
				kind = KindStruct
			case "interface_type":
				kind = KindInterface
			}
		}
		return addSym(w, n, kind, name, "", "")

	case "const_spec":
		if goTopLevelSpec(n) {
			return addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), "", "")
		}
	case "var_spec":
		if goTopLevelSpec(n) {
			return addSym(w, n, KindVariable, sanitizeIdent(nameOf(w, n)), "", "")
		}

	case "call_expression":
		f := n.ChildByFieldName("function")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "import_spec":
		path := strings.Trim(txt(w, n.ChildByFieldName("path")), "\"`")
		var names []string
		if a := n.ChildByFieldName("name"); a != nil {
			names = append(names, a.Content(w.src))
		}
		sl, _ := posOf(n.StartPoint())
		el, _ := posOf(n.EndPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: path, Names: names, StartLine: sl, EndLine: el})
	}
	return ""
}

// goReceiverType extracts "Server" from method_declaration(receiver (*Server)).
func goReceiverType(w *walkCtx, n *sitter.Node) string {
	r := n.ChildByFieldName("receiver")
	if r == nil {
		return ""
	}
	// parameter_list > parameter_declaration > type
	for i := uint32(0); i < r.NamedChildCount(); i++ {
		p := r.NamedChild(int(i))
		if t := p.ChildByFieldName("type"); t != nil {
			return strings.TrimPrefix(strings.TrimSpace(t.Content(w.src)), "*")
		}
	}
	return ""
}

// goTopLevelSpec reports whether a const/var spec sits at package level
// (source_file > const_declaration > spec) rather than inside a function.
func goTopLevelSpec(n *sitter.Node) bool {
	d := n.Parent()
	if d == nil {
		return false
	}
	g := d.Parent()
	return g != nil && g.Type() == "source_file"
}
