package lang

import (
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

var (
	parsersMu sync.Mutex
	parsers   = map[string]*sitter.Parser{}
)

// parseWith returns a parsed tree using a cached parser for the grammar.
// Tree-sitter parsers are not safe for concurrent use; indexing is
// single-goroutine in V1, but tests may run in parallel, so guard anyway.
func parseWith(key string, lg *sitter.Language, src []byte) *sitter.Tree {
	parsersMu.Lock()
	defer parsersMu.Unlock()
	p := parsers[key]
	if p == nil {
		p = sitter.NewParser()
		p.SetLanguage(lg)
		parsers[key] = p
	}
	return p.Parse(nil, src)
}

func extractWith(key string, lg *sitter.Language, src []byte, visit visitFn) Result {
	tree := parseWith(key, lg, src)
	if tree == nil {
		return Result{}
	}
	defer tree.Close()
	w := &walkCtx{src: src, res: &Result{}}
	walk(tree.RootNode(), w, visit)
	return *w.res
}
