package utils

import "testing"

func TestMask(t *testing.T) {
	cases := []struct {
		fn       func(string) string
		in, want string
	}{
		{MaskName, "Test Guest A", "T*** G*** A***"},
		{MaskName, "  試験 太郎 ", "試*** 太***"},
		{MaskName, "", ""},
		{MaskPhone, "090-1234-5678", "****78"},
		{MaskPhone, "12", "****"},
		{MaskEmail, "test-guest@example.test", "t***@example.test"},
		{MaskEmail, "invalid", "***"},
		{MaskEmail, "@example.test", "***"},
		{MaskAddress, "長野県佐久市テスト1-2-3", "長野県***"},
		{MaskAddress, " ab ", "ab***"},
		{MaskAddress, "", ""},
	}
	for _, tc := range cases {
		if got := tc.fn(tc.in); got != tc.want {
			t.Errorf("mask(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
