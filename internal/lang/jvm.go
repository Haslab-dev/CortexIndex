package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/kotlin"
)

func init() {
	register([]string{".java"}, &Language{Name: "java", Extract: extractJava})
	register([]string{".kt", ".kts"}, &Language{Name: "kotlin", Extract: extractKotlin})
}

func extractJava(src []byte) Result {
	return extractWith("java", java.GetLanguage(), src, javaVisit)
}

func extractKotlin(src []byte) Result {
	return extractWith("kotlin", kotlin.GetLanguage(), src, ktVisit)
}

func javaVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "class_declaration", "record_declaration":
		for _, ref := range javaTypeRefs(w, n) {
			addRef(w, ref.node, ref.kind, ref.name, "")
		}
		return addSym(w, n, KindClass, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "interface_declaration", "annotation_type_declaration":
		return addSym(w, n, KindInterface, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "enum_declaration":
		return addSym(w, n, KindEnum, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "enum_constant":
		addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""

	case "method_declaration":
		return addSym(w, n, KindMethod, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "constructor_declaration":
		return addSym(w, n, KindConstructor, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "variable_declarator":
		// field (class member) or local variable; keep only class members
		if p := n.Parent(); p != nil && p.Type() == "field_declaration" {
			addSym(w, n, KindProperty, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		}
		return ""

	case "method_invocation":
		name := sanitizeIdent(nameOf(w, n))
		qual := ""
		if o := n.ChildByFieldName("object"); o != nil {
			qual = txt(w, o)
		}
		addRef(w, n, RefCall, name, qual)

	case "object_creation_expression":
		t := n.ChildByFieldName("type")
		if t != nil {
			q, nm := splitQualified(txt(w, t))
			addRef(w, t, RefNew, sanitizeIdent(nm), q)
		}

	case "import_declaration":
		raw := strings.TrimSpace(txt(w, n))
		raw = strings.TrimSuffix(raw, ";")
		raw = strings.TrimPrefix(raw, "import ")
		raw = strings.TrimPrefix(raw, "static ")
		sl, _ := posOf(n.StartPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: raw, StartLine: sl, EndLine: sl})
	}
	return ""
}

type javaRef struct {
	node *sitter.Node
	kind RefKind
	name string
}

func javaTypeRefs(w *walkCtx, n *sitter.Node) []javaRef {
	var out []javaRef
	if sup := n.ChildByFieldName("superclass"); sup != nil {
		_, nm := splitQualified(txt(w, sup))
		nm = strings.TrimPrefix(sanitizeIdent(nm), "extends ")
		out = append(out, javaRef{sup, RefExtends, nm})
	}
	if ifs := n.ChildByFieldName("interfaces"); ifs != nil {
		for i := uint32(0); i < ifs.NamedChildCount(); i++ {
			c := ifs.NamedChild(int(i))
			_, nm := splitQualified(txt(w, c))
			out = append(out, javaRef{c, RefImplements, sanitizeIdent(nm)})
		}
	}
	return out
}

func ktVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "class_declaration":
		name := ktName(w, n)
		if name == "" {
			return ""
		}
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			if c.Type() == "delegation_specifier" {
				for j := uint32(0); j < c.NamedChildCount(); j++ {
					sc := c.NamedChild(int(j))
					switch sc.Type() {
					case "user_type", "constructor_invocation":
						_, nm := splitQualified(txt(w, sc))
						addRef(w, sc, RefExtends, sanitizeIdent(nm), "")
					}
				}
			}
		}
		return addSym(w, n, KindClass, name, w.cur(), "")

	case "object_declaration":
		name := ktName(w, n)
		if name == "" {
			return "" // anonymous / companion objects
		}
		return addSym(w, n, KindClass, name, w.cur(), "")

	case "function_declaration":
		name := ktName(w, n)
		if name == "" {
			return ""
		}
		return addSym(w, n, KindFunction, name, w.cur(), "")

	case "property_declaration":
		// val/var members and top-level properties; old grammar uses
		// variable_declaration (not variable_declarator) with a bare
		// simple_identifier child. Locals (inside a function body) are noise.
		if p := n.Parent(); p != nil && p.Type() != "class_body" && p.Type() != "source_file" {
			return ""
		}
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			switch c.Type() {
			case "variable_declaration", "variable_declarator":
				name := ktName(w, c)
				if name != "" {
					addSym(w, c, KindProperty, name, w.cur(), "")
				}
			}
		}
		return ""

	case "call_expression":
		// No "callee" field in this grammar revision: the callee is the
		// first named child that is not the call_suffix.
		var callee *sitter.Node
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(int(i))
			if c.Type() == "call_suffix" || c.Type() == "value_arguments" {
				continue
			}
			callee = c
			break
		}
		if callee == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, callee))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "import_header":
		// import_header > identifier (dotted text)
		var path string
		for i := uint32(0); i < n.NamedChildCount(); i++ {
			if c := n.NamedChild(int(i)); c.Type() == "identifier" {
				path = txt(w, c)
				break
			}
		}
		sl, _ := posOf(n.StartPoint())
		w.res.Imports = append(w.res.Imports, Import{Path: path, StartLine: sl, EndLine: sl})
	}
	return ""
}

// ktName extracts a declaration name from the old tree-sitter-kotlin
// grammar, which has no "name" field: the name is the first simple or type
// identifier child that is not a modifier/parameter/body wrapper.
func ktName(w *walkCtx, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(int(i))
		switch c.Type() {
		case "simple_identifier", "type_identifier":
			return sanitizeIdent(c.Content(w.src))
		case "modifiers", "annotation", "use_site_annotation", "type_parameters",
			"binding_pattern_kind", "parameter_modifiers":
			continue // scan past decorations to the name
		default:
			// structural children (params, body, supertypes...) mean the
			// name either came before them or does not exist.
			return ""
		}
	}
	return ""
}
