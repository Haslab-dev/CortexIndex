package memory

import "testing"

func TestParsePreference(t *testing.T) {
	content := "---\ncortex: preference\nid: table-tests\nscope: team\nstatus: active\nconfidence: 0.8\nprovider: taste\nupdated_at: 2026-09-09\n---\n\nPrefer table-driven tests.\n"
	p, ok, err := ParsePreference("preferences/testing.md", content)
	if err != nil || !ok || p.ID != "table-tests" || p.Confidence != 0.8 || p.Provider != "taste" {
		t.Fatalf("p=%#v ok=%v err=%v", p, ok, err)
	}
}

func TestNormalizeTasteCreatesProposal(t *testing.T) {
	p, err := NormalizeTaste(TastePreference{ID: "concise", Scope: "user", Text: "Prefer concise output.", Confidence: 0.8, UpdatedAt: "2026-09-09", Source: "taste://user/concise"}, "preferences/concise.md")
	if err != nil || p.Status != "proposed" || p.Provider != "taste" || p.Confidence != 0.8 {
		t.Fatalf("p=%#v err=%v", p, err)
	}
}

func TestParsePreferenceRejectsConfidence(t *testing.T) {
	content := "---\ncortex: preference\nid: bad\nupdated_at: 2026-09-09\nconfidence: 1.2\n---\n\nNo.\n"
	if _, ok, err := ParsePreference("bad.md", content); !ok || err == nil {
		t.Fatalf("expected confidence error: ok=%v err=%v", ok, err)
	}
}
