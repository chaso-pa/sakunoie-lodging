package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// CalendarDate is a date without a time or a time zone, written YYYY-MM-DD:
// the day a member says they visited a place.
//
// It is stored and read as text, never as a time.Time. A time.Time passes
// through the driver's time zone conversion (the DSN's loc), so "2026-10-08"
// parsed in one zone can come back as the 7th or the 9th when the server's
// zone differs or changes. Text goes in and out unchanged.
//
// It works for DATE columns (gps_visits.visited_date) and for DATETIME ones
// that only ever hold a date (posts.visited_at, stored as midnight).
type CalendarDate string

// ParseCalendarDate accepts exactly YYYY-MM-DD and rejects impossible dates.
func ParseCalendarDate(s string) (CalendarDate, error) {
	if _, err := time.Parse(DateLayout, s); err != nil {
		return "", fmt.Errorf("not a date in YYYY-MM-DD format: %q", s)
	}
	return CalendarDate(s), nil
}

func (d CalendarDate) String() string { return string(d) }

// Value sends the date as text, which the driver passes through untouched.
func (d CalendarDate) Value() (driver.Value, error) {
	return string(d), nil
}

// Scan reads the column. With parseTime=True the driver hands over a
// time.Time built from the stored text in the DSN's location, so its own
// year, month and day are the stored ones whatever that location is.
func (d *CalendarDate) Scan(src any) error {
	switch v := src.(type) {
	case time.Time:
		*d = CalendarDate(v.Format(DateLayout))
	case []byte:
		return d.scanText(string(v))
	case string:
		return d.scanText(v)
	default:
		return fmt.Errorf("cannot scan %T into CalendarDate", src)
	}
	return nil
}

func (d *CalendarDate) scanText(s string) error {
	if len(s) < len(DateLayout) {
		return fmt.Errorf("cannot scan %q into CalendarDate", s)
	}
	parsed, err := ParseCalendarDate(s[:len(DateLayout)])
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// GormDataType keeps GORM from treating the type as a plain string column
// when it builds statements.
func (CalendarDate) GormDataType() string { return "date" }
