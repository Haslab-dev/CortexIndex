package store

import (
	"strings"
)

// SanitizeFTS turns a free-text query into a safe FTS5 MATCH expression:
// each token is double-quoted (FTS5 string syntax) and tokens are OR-ed so
// partial keyword overlap still matches; a trailing "*" enables prefix
// matching on the last token when the query looks like an identifier stub.
func SanitizeFTS(query string) string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '.', '/', '-', '_', ':', '(', ')', ',', ';', '"', '\'', '*':
			return true
		}
		return false
	})
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, `"`, `""`)
		parts = append(parts, `"`+f+`"`)
	}
	return strings.Join(parts, " OR ")
}

// SanitizeFTSPrefix sanitizes a query and makes every token a prefix term,
// for type-as-you-search behavior on short inputs.
func SanitizeFTSPrefix(query string) string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '.', '/', '-', '_', ':', '(', ')', ',', ';', '"', '\'', '*':
			return true
		}
		return false
	})
	if len(fields) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, `"`, `""`)
		parts = append(parts, `"`+f+`"* `)
	}
	return strings.Join(parts, "OR ")
}
