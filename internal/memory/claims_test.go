package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseClaim(t *testing.T) {
	content := `---
id: auth-boundary
type: constraint
scope: repository
confidence: 0.8
status: active
updated_at: 2026-09-09
source:
  - internal/auth/service.go
evidence: [internal/auth/service.go#L10-L22]
---

Handlers must call AuthService.
`
	claim, ok, err := ParseClaim("decisions/auth.md", content)
	if err != nil || !ok {
		t.Fatalf("ParseClaim() = %#v, %v, %v", claim, ok, err)
	}
	if claim.ID != "auth-boundary" || claim.Type != "constraint" || claim.Confidence == nil || *claim.Confidence != 0.8 {
		t.Fatalf("unexpected claim: %#v", claim)
	}
	if len(claim.Evidence) != 1 || claim.Evidence[0].StartLine != 10 || claim.Evidence[0].EndLine != 22 {
		t.Fatalf("unexpected evidence: %#v", claim.Evidence)
	}
}

func TestParseClaimLegacyAndInvalid(t *testing.T) {
	if _, ok, err := ParseClaim("project.md", "# Project\n\nplain memory\n"); ok || err != nil {
		t.Fatalf("legacy Markdown should not be a claim: ok=%v err=%v", ok, err)
	}
	if _, ok, err := ParseClaim("bad.md", "---\nid: bad\ntype: constraint\nconfidence: 2\nupdated_at: 2026-09-09\n---\ntext\n"); !ok || err == nil {
		t.Fatalf("invalid confidence should fail: ok=%v err=%v", ok, err)
	}
}

func TestValidateClaimsRelationships(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".cortex", "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, ".cortex", "decisions", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("old.md", "---\nid: old\ntype: decision\nupdated_at: 2026-09-09\n---\nOld decision.\n")
	write("new.md", "---\nid: new\ntype: decision\nupdated_at: 2026-09-09\nsupersedes: [old]\n---\nNew decision.\n")
	items, err := ValidateClaims(root)
	if err != nil || len(items) != 2 {
		t.Fatalf("ValidateClaims() = %#v, %v", items, err)
	}
	for _, item := range items {
		if item.State != StateCurrent {
			t.Fatalf("unexpected item: %#v", item)
		}
	}
}
