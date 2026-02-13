package secret

import "testing"

func TestParseType(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"login_password", true},
		{"text", true},
		{"binary", true},
		{"card", true},
		{"unknown", false},
		{"", false},
	}
	for _, c := range cases {
		_, err := ParseType(c.in)
		if c.valid && err != nil {
			t.Fatalf("expected valid for %q, got %v", c.in, err)
		}
		if !c.valid && err == nil {
			t.Fatalf("expected error for %q", c.in)
		}
	}
}
