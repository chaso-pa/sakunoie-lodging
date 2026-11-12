package utils

import (
	"bytes"
	"encoding/csv"
	"testing"
)

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"Test Guest":       "Test Guest",
		"=SUM(1)":          "'=SUM(1)",
		"+81-00-0000":      "'+81-00-0000",
		"-1+1":             "'-1+1",
		"@cmd":             "'@cmd",
		"\t=1":             "'\t=1",
		"\r=1":             "'\r=1",
		"\n=1":             "'\n=1",
		"＝1+1":             "'＝1+1",
		"＋1":               "'＋1",
		"－1":               "'－1",
		"＠x":               "'＠x",
		"a=1":              "a=1",
		"佐久の家":             "佐久の家",
		"2026-12-01":       "2026-12-01",
		"Test, \"quoted\"": "Test, \"quoted\"",
	}
	for in, want := range cases {
		if got := CSVSafe(in); got != want {
			t.Errorf("CSVSafe(%q) = %q, want %q", in, got, want)
		}
	}
}

// Embedded quotes and newlines survive a csv.Writer round trip.
func TestCSVSafeQuoting(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	row := []string{CSVSafe("=HYPERLINK(\"x\")"), CSVSafe("line1\nline2"), CSVSafe(`say "hi"`)}
	if err := w.Write(row); err != nil {
		t.Fatal(err)
	}
	w.Flush()
	got, err := csv.NewReader(&buf).Read()
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "'=HYPERLINK(\"x\")" || got[1] != "line1\nline2" || got[2] != `say "hi"` {
		t.Fatalf("round trip: %q", got)
	}
}
