package utils

import (
	"fmt"
	"time"
)

// CivilDate is a calendar date with no time zone (a MariaDB DATE).
type CivilDate struct {
	Year  int
	Month time.Month
	Day   int
}

const civilDateLayout = "2006-01-02"

// ParseCivilDate parses "YYYY-MM-DD".
func ParseCivilDate(s string) (CivilDate, error) {
	t, err := time.Parse(civilDateLayout, s)
	if err != nil {
		return CivilDate{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return CivilDate{t.Year(), t.Month(), t.Day()}, nil
}

// CivilDateFromDB converts a scanned DATE column to a CivilDate.
//
// The driver scans DATE as midnight in the connection location, which
// internal/database forces to UTC, so Y/M/D are read in UTC. Never convert
// with t.In(loc): that shifts the calendar day for zones west of UTC.
func CivilDateFromDB(t time.Time) CivilDate {
	u := t.UTC()
	return CivilDate{u.Year(), u.Month(), u.Day()}
}

// CivilDateIn returns the calendar date of instant t in loc.
func CivilDateIn(t time.Time, loc *time.Location) CivilDate {
	l := t.In(loc)
	return CivilDate{l.Year(), l.Month(), l.Day()}
}

// ToDB returns the value to write into a DATE column (midnight UTC).
func (d CivilDate) ToDB() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// String formats the date as YYYY-MM-DD.
func (d CivilDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

// Compare returns -1, 0 or +1.
func (d CivilDate) Compare(o CivilDate) int {
	return d.ToDB().Compare(o.ToDB())
}

// EditTokenExpiry returns the last microsecond of the checkout date in the
// facility time zone, as a UTC instant (contract §4.6). A token is valid while
// now <= expiry.
func EditTokenExpiry(checkout CivilDate, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load facility time zone: %w", err)
	}
	return time.Date(checkout.Year, checkout.Month, checkout.Day, 23, 59, 59, 999999000, loc).UTC(), nil
}
