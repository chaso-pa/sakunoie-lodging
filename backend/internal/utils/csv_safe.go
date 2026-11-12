package utils

import "unicode/utf8"

// CSVSafe neutralizes spreadsheet formula injection (contract §8): a cell
// that starts with = + - @ (ASCII or full-width), a tab, a carriage return or
// any other control character is prefixed with a single quote, so Excel and
// similar tools treat it as text. encoding/csv handles quoting separately.
func CSVSafe(s string) string {
	r, _ := utf8.DecodeRuneInString(s)
	switch {
	case s == "":
		return s
	case r == '=', r == '+', r == '-', r == '@',
		r == '＝', r == '＋', r == '－', r == '＠',
		r < 0x20, r == 0x7f:
		return "'" + s
	}
	return s
}
