package lang

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

func init() {
	register([]string{".rs"}, &Language{Name: "rust", Extract: extractRust})
}

func extractRust(src []byte) Result {
	return extractWith("rust", rust.GetLanguage(), src, rsVisit)
}

func rsVisit(n *sitter.Node, w *walkCtx) string {
	switch n.Type() {
	case "function_item":
		return addSym(w, n, KindFunction, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "struct_item", "union_item":
		return addSym(w, n, KindStruct, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "enum_item":
		return addSym(w, n, KindEnum, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "trait_item":
		return addSym(w, n, KindTrait, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "impl_item":
		typeName := sanitizeIdent(txt(w, n.ChildByFieldName("type")))
		traitName := sanitizeIdent(txt(w, n.ChildByFieldName("trait")))
		return addSym(w, n, KindImpl, typeName, w.cur(), traitName)

	case "type_item":
		addSym(w, n, KindType, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""

	case "const_item":
		addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""

	case "static_item":
		addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""

	case "mod_item":
		return addSym(w, n, KindModule, sanitizeIdent(nameOf(w, n)), w.cur(), "")

	case "call_expression":
		f := n.ChildByFieldName("function")
		if f == nil {
			return ""
		}
		q, nm := splitQualified(txt(w, f))
		addRef(w, n, RefCall, sanitizeIdent(nm), q)

	case "use_declaration":
		w.res.Imports = append(w.res.Imports, rustUseImport(w, n))

	case "enum_variant":
		addSym(w, n, KindConstant, sanitizeIdent(nameOf(w, n)), w.cur(), "")
		return ""
	}
	return ""
}

// rustUseImport flattens `use a::b::{c, d as e};` into path text + leaf names.
func rustUseImport(w *walkCtx, n *sitter.Node) Import {
	sl, _ := posOf(n.StartPoint())
	el, _ := posOf(n.EndPoint())
	raw := strings.TrimSpace(txt(w, n))
	raw = strings.TrimPrefix(raw, "use ")
	raw = strings.TrimSuffix(raw, ";")
	imp := Import{StartLine: sl, EndLine: el}

	if open := strings.Index(raw, "{"); open >= 0 && strings.Contains(raw, "}") {
		inner := raw[open+1 : strings.LastIndex(raw, "}")]
		imp.Path = strings.TrimSpace(strings.TrimSuffix(raw[:open], "::"))
		for _, part := range strings.Split(inner, ",") {
			part = strings.TrimSpace(part)
			if part == "" || part == "self" {
				continue
			}
			if as := strings.SplitN(part, " as ", 2); len(as) == 2 {
				part = strings.TrimSpace(as[1])
			}
			imp.Names = append(imp.Names, lastRustSegment(part))
		}
	} else {
		if as := strings.SplitN(raw, " as ", 2); len(as) == 2 {
			raw = strings.TrimSpace(as[0])
		}
		imp.Path = raw
		imp.Names = []string{lastRustSegment(raw)}
	}
	return imp
}

func lastRustSegment(s string) string {
	if i := strings.LastIndex(s, "::"); i >= 0 {
		return s[i+2:]
	}
	return s
}
