package lang

import (
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
)

func TestDocDump(t *testing.T) {
	p := sitter.NewParser()
	p.SetLanguage(python.GetLanguage())
	src := "def login(self, user):\n    \"\"\"Login a user.\"\"\"\n    return 1\n"
	tree := p.Parse(nil, []byte(src))
	fmt.Println("PY:", tree.RootNode().String())
	tree.Close()

	r := sitter.NewParser()
	r.SetLanguage(rust.GetLanguage())
	rsrc := "impl S {\n    /// Issue a token.\n    pub fn issue(&self) {}\n}\n"
	rt := r.Parse(nil, []byte(rsrc))
	fmt.Println("RS:", rt.RootNode().String())
	rt.Close()
}
