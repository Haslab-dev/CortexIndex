package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	tsgo "github.com/smacker/go-tree-sitter/typescript/typescript"
)

func init() {
	register([]string{".ts", ".mts", ".cts"}, &Language{Name: "typescript", Extract: extractTS})
	register([]string{".tsx"}, &Language{Name: "tsx", Extract: extractTSX})
	register([]string{".js", ".mjs", ".cjs", ".jsx"}, &Language{Name: "javascript", Extract: extractJS})
}

func extractTS(src []byte) Result {
	return extractWith("ts", tsgo.GetLanguage(), src, tsVisit)
}

func extractTSX(src []byte) Result {
	return extractWith("tsx", tsx.GetLanguage(), src, tsVisit)
}

func extractJS(src []byte) Result {
	// tree-sitter-javascript includes JSX rules, so .jsx parses natively.
	return extractWith("js", javascript.GetLanguage(), src, tsVisit)
}

func tsVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "function_declaration", "generator_function_declaration", "function_signature":
		return addSym(w, n, KindFunction, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "class_declaration", "abstract_class_declaration":
		return addSym(w, n, KindClass, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "interface_declaration":
		return addSym(w, n, KindInterface, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "type_alias_declaration":
		return addSym(w, n, KindType, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "enum_declaration":
		return addSym(w, n, KindEnum, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "method_definition", "abstract_method_signature", "method_signature":
		name := sanitizeIdent(nameOf(w, n))
		if name == "" { // computed property keys
			return ""
		}
		return addSym(w, n, KindMethod, name, w.cur(), "")

	case "public_field_definition":
		name := sanitizeIdent(nameOf(w, n))
		return addSym(w, n, KindProperty, name, w.cur(), "")

	case "variable_declarator":
		nameNode := n.ChildByFieldName("name")
		if nameNode == nil || nameNode.Type() != "identifier" {
			return "" // destructuring patterns
		}
		if w.cur() != "" {
			return "" // local variables inside functions are noise
		}
		name := nameNode.Content(w.src)
		val := n.ChildByFieldName("value")
		if val != nil {
			switch val.Type() {
			case "arrow_function", "function", "function_expression":
				return addSym(w, n, KindFunction, name, "", "")
			}
		}
		kind := KindVariable
		if d := n.Parent(); d != nil && d.Type() == "lexical_declaration" {
			head := w.src[d.StartByte():min(d.EndByte(), d.StartByte()+8)]
			if strings.HasPrefix(string(head), "const") {
				kind = KindConstant
			}
		}
		addSym(w, n, kind, name, "", "")
		return ""

	case "call_expression":
		f := n.ChildByFieldName("function")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "new_expression":
		f := n.ChildByFieldName("constructor")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefNew, sanitizeIdent(nm), q)

	case "extends_clause", "heritage_clause":
		// TS uses extends_clause; older JS grammar uses heritage_clause
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			switch c.Type() {
			case "identifier", "member_expression":
				addRef(w, c, RefExtends, sanitizeIdent(txt(w, c)), "")
			}
		}

	case "implements_clause":
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			if c.Type() == "type_identifier" || c.Type() == "generic_type" {
				addRef(w, c, RefImplements, sanitizeIdent(txt(w, c)), "")
			}
		}

	case "import_statement":
		path := strings.Trim(txt(w, n.ChildByFieldName("source")), "\"'`")
		var names []string
		// import_clause is a plain child in both JS and TS grammars, not a field
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			if c.Type() != "import_clause" {
				continue
			}
			for j := uint32(0); j < c.NamedChildCount(); j++ {
				item := c.NamedChild(int(j))
				switch item.Type() {
				case "identifier": // default import binding
					names = append(names, item.Content(w.src))
				case "namespace_import":
					for k := uint32(0); k < item.NamedChildCount(); k++ {
						nn := item.NamedChild(int(k))
						if nn.Type() == "identifier" {
							names = append(names, nn.Content(w.src))
						}
					}
				case "named_imports":
					for k := uint32(0); k < item.NamedChildCount(); k++ {
						sp := item.NamedChild(int(k))
						if sp.Type() == "import_specifier" {
							names = append(names, nameOf(w, sp))
						}
					}
				}
			}
		}
		sl, _ := posOf(n.StartPoint())
		el, _ := posOf(n.EndPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: path, Names: names, StartLine: sl, EndLine: el})
	}
	return ""
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
