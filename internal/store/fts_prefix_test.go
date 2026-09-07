package store

import "testing"

func TestSanitizeFTSPrefix(t *testing.T) {
	cases := map[string]string{
		"AuthService": `"AuthService"* `,
		"a b":         `"a"* OR "b"* `,
		"":            "",
	}
	for in, want := range cases {
		if got := SanitizeFTSPrefix(in); got != want {
			t.Errorf("SanitizeFTSPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}
