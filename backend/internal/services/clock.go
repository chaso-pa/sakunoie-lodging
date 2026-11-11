package services

import "time"

// Clock is injectable so tests can cross token-expiry boundaries.
type Clock interface {
	Now() time.Time
}

// SystemClock returns UTC with microsecond precision (DATETIME(6)).
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }
