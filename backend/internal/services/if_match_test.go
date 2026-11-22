package services

import "testing"

func TestIfMatchExactOnly(t *testing.T) {
	const etag = "rabc123"
	for header, want := range map[string]bool{
		`"rabc123"`:         true,
		`rabc123`:           true,
		` "rabc123" `:       true,
		`*`:                 false,
		`"*"`:               false,
		`W/"rabc123"`:       false,
		`W/rabc123`:         false,
		`"rold", "rabc123"`: false,
		`"rabc12"`:          false,
		``:                  false,
		`"rabc123`:          false,
	} {
		if got := ifMatch(header, etag); got != want {
			t.Errorf("ifMatch(%q) = %v, want %v", header, got, want)
		}
	}
}
