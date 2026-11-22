package utils

import (
	"testing"
	"time"
)

func TestEditTokenExpiry(t *testing.T) {
	cases := []struct {
		name     string
		tz       string
		checkout string
		want     string // UTC
	}{
		{"tokyo", "Asia/Tokyo", "2026-12-03", "2026-12-03T14:59:59.999999Z"},
		{"honolulu", "Pacific/Honolulu", "2026-12-03", "2026-12-04T09:59:59.999999Z"},
		// DST starts 2027-03-14 in New York (EST -5 -> EDT -4).
		{"new_york_before_dst", "America/New_York", "2027-03-13", "2027-03-14T04:59:59.999999Z"},
		{"new_york_dst_day", "America/New_York", "2027-03-14", "2027-03-15T03:59:59.999999Z"},
		// DST ends 2027-11-07 (EDT -4 -> EST -5).
		{"new_york_dst_end_day", "America/New_York", "2027-11-07", "2027-11-08T04:59:59.999999Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParseCivilDate(tc.checkout)
			if err != nil {
				t.Fatal(err)
			}
			got, err := EditTokenExpiry(d, tc.tz)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := time.Parse(time.RFC3339Nano, tc.want)
			if !got.Equal(want) {
				t.Fatalf("got %s, want %s", got.Format(time.RFC3339Nano), tc.want)
			}
			if got.Location() != time.UTC {
				t.Fatalf("expiry must be UTC, got %s", got.Location())
			}
			// ±1µs boundary: the expiry instant is still the checkout date
			// locally, one microsecond later is the next day.
			loc, _ := time.LoadLocation(tc.tz)
			if CivilDateIn(got, loc) != d {
				t.Fatalf("expiry is not on the checkout date locally")
			}
			if CivilDateIn(got.Add(time.Microsecond), loc) == d {
				t.Fatalf("expiry+1µs must be the next local day")
			}
		})
	}
}

func TestEditTokenExpiryUnknownZone(t *testing.T) {
	if _, err := EditTokenExpiry(CivilDate{2026, 12, 3}, "Mars/Olympus"); err == nil {
		t.Fatal("expected error for unknown zone")
	}
}

func TestCivilDateFromDB(t *testing.T) {
	// DATE scanned by the driver with loc=UTC.
	scanned := time.Date(2026, 12, 3, 0, 0, 0, 0, time.UTC)
	if got := CivilDateFromDB(scanned); got.String() != "2026-12-03" {
		t.Fatalf("got %s", got)
	}
	// Converting with In(loc) west of UTC would give the previous day; the
	// helper must not do that.
	hon, _ := time.LoadLocation("Pacific/Honolulu")
	if got := CivilDateFromDB(scanned.In(hon)); got.String() != "2026-12-03" {
		t.Fatalf("got %s", got)
	}
}

func TestParseCivilDate(t *testing.T) {
	if _, err := ParseCivilDate("2026-02-30"); err == nil {
		t.Fatal("expected error for invalid date")
	}
	d, err := ParseCivilDate("2026-02-28")
	if err != nil || d.String() != "2026-02-28" {
		t.Fatalf("got %v %v", d, err)
	}
	if d.Compare(CivilDate{2026, 3, 1}) != -1 {
		t.Fatal("compare")
	}
}
