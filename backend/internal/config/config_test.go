package config

import "testing"

func TestInviteCodeWarnings(t *testing.T) {
	cases := map[string]int{
		ExampleInviteCode:        1, // the example value; it is exactly 12 characters
		"SHORT-CODE":             1,
		"abcdefghijk":            1, // 11 characters
		"abcdefghijkl":           0, // 12 characters
		"k3Vq-91xT-Zp0d-Lm7R-aa": 0,
		"  padded  ":             1,
	}
	for code, want := range cases {
		cfg := &Config{InviteCode: code}
		if got := cfg.InviteCodeWarnings(); len(got) != want {
			t.Errorf("InviteCodeWarnings(%q) = %q, want %d warnings", code, got, want)
		}
	}
}
