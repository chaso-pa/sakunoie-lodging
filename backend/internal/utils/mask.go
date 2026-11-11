package utils

import (
	"strings"
	"unicode/utf8"
)

// MaskName keeps the first character of each space-separated word.
// "Test Guest" -> "T*** G***". Names without spaces keep one character.
func MaskName(s string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	out := make([]string, len(words))
	for i, w := range words {
		r, _ := utf8.DecodeRuneInString(w)
		out[i] = string(r) + "***"
	}
	return strings.Join(out, " ")
}

// MaskPhone keeps the last two digits. "090-1234-5678" -> "****78".
func MaskPhone(s string) string {
	var digits []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) < 4 {
		return "****"
	}
	return "****" + string(digits[len(digits)-2:])
}

// MaskEmail keeps the first character of the local part and the domain.
// "guest@example.test" -> "g***@example.test".
func MaskEmail(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 0 {
		return "***"
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(r) + "***" + s[at:]
}

// MaskAddress keeps the first three characters. "長野県佐久市…" -> "長野県***".
func MaskAddress(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) == 0 {
		return ""
	}
	return string(r[:min(3, len(r))]) + "***"
}
