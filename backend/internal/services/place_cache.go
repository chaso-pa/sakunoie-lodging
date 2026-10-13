package services

import (
	"context"
	"sync"
	"time"
)

// placeCache is a small in-memory cache with a lifetime per entry and a cap
// on the number of entries. Only the OSM provider uses one: Google's terms
// do not allow caching its results.
type placeCache struct {
	ttl time.Duration
	max int
	now func() time.Time

	mu      sync.Mutex
	entries map[string]placeCacheEntry
}

type placeCacheEntry struct {
	places  []Place
	expires time.Time
}

func newPlaceCache(ttl time.Duration, maxEntries int) *placeCache {
	return &placeCache{ttl: ttl, max: maxEntries, now: time.Now, entries: map[string]placeCacheEntry{}}
}

// get returns a copy of the cached places, so callers can reorder or trim
// what they get without touching the cache.
func (c *placeCache) get(key string) ([]Place, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || !c.now().Before(entry.expires) {
		return nil, false
	}
	return append([]Place(nil), entry.places...), true
}

// put stores places under key, keeping the cache within its cap: expired
// entries go first, then the ones closest to expiry.
func (c *placeCache) put(key string, places []Place) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for k, e := range c.entries {
		if !now.Before(e.expires) {
			delete(c.entries, k)
		}
	}
	if _, replacing := c.entries[key]; !replacing {
		for len(c.entries) >= c.max {
			oldest, found := "", false
			var oldestExpiry time.Time
			for k, e := range c.entries {
				if !found || e.expires.Before(oldestExpiry) {
					oldest, oldestExpiry, found = k, e.expires, true
				}
			}
			delete(c.entries, oldest)
		}
	}
	c.entries[key] = placeCacheEntry{places: append([]Place(nil), places...), expires: now.Add(c.ttl)}
}

func (c *placeCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// upstreamThrottle spaces out the calls to one upstream service: each call
// starts at least interval after the one before, whichever member it is
// for. A call that would have to wait longer than maxWait is refused, so a
// burst queues briefly instead of piling up behind the limit.
type upstreamThrottle struct {
	interval time.Duration
	maxWait  time.Duration
	now      func() time.Time
	sleep    func(ctx context.Context, d time.Duration) error

	mu   sync.Mutex
	next time.Time // earliest start of the next call
}

func newUpstreamThrottle(interval, maxWait time.Duration) *upstreamThrottle {
	return &upstreamThrottle{interval: interval, maxWait: maxWait, now: time.Now, sleep: sleepContext}
}

// wait blocks until it is this call's turn. It returns ErrPlaceBusy without
// waiting when the turn is too far away, and the context's error when the
// request ends first (the turn is then given up, not handed on, which only
// makes the spacing more conservative).
func (t *upstreamThrottle) wait(ctx context.Context) error {
	if t == nil || t.interval <= 0 {
		return nil
	}
	t.mu.Lock()
	now := t.now()
	start := t.next
	if start.Before(now) {
		start = now
	}
	delay := start.Sub(now)
	if delay > t.maxWait {
		t.mu.Unlock()
		return ErrPlaceBusy
	}
	t.next = start.Add(t.interval)
	t.mu.Unlock()

	if delay <= 0 {
		return nil
	}
	return t.sleep(ctx, delay)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
