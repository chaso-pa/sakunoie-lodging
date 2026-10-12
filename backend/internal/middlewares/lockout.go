package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// FailureLockout blocks a client IP for blockFor after maxFailures
// consecutive 401 responses from the wrapped route. A successful response
// clears the count, and so does blockFor passing without another failure.
//
// Requests without a real client address (see HasClientAddress) pass
// through uncounted: they all share one address, so a lockout there would
// let anyone block every guest's restore. See the comment on the limits in
// rate_limit.go for why that is safe.
//
// State is in memory, like the rate limiters.
type FailureLockout struct {
	maxFailures int
	blockFor    time.Duration
	now         func() time.Time

	mu      sync.Mutex
	entries map[string]*lockoutEntry
}

type lockoutEntry struct {
	failures     int
	lastFailure  time.Time
	blockedUntil time.Time
}

func NewFailureLockout(maxFailures int, blockFor time.Duration) *FailureLockout {
	return &FailureLockout{
		maxFailures: maxFailures,
		blockFor:    blockFor,
		now:         time.Now,
		entries:     make(map[string]*lockoutEntry),
	}
}

func (l *FailureLockout) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasClientAddress(c) {
			c.Next()
			return
		}
		key := c.ClientIP()
		if l.blocked(key) {
			abortWithProblem(c, http.StatusTooManyRequests, "too many failed attempts")
			return
		}

		c.Next()

		switch status := c.Writer.Status(); {
		case status == http.StatusUnauthorized:
			l.recordFailure(key)
		case status < http.StatusBadRequest:
			l.reset(key)
		}
	}
}

func (l *FailureLockout) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune()
	e, ok := l.entries[key]
	return ok && l.now().Before(e.blockedUntil)
}

func (l *FailureLockout) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.entries[key]
	if !ok {
		e = &lockoutEntry{}
		l.entries[key] = e
	}
	e.failures++
	e.lastFailure = now
	if e.failures >= l.maxFailures {
		e.failures = 0
		e.blockedUntil = now.Add(l.blockFor)
	}
}

func (l *FailureLockout) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// prune drops entries whose block and failure streak have both lapsed.
// Callers hold l.mu.
func (l *FailureLockout) prune() {
	now := l.now()
	for key, e := range l.entries {
		if now.After(e.blockedUntil) && now.Sub(e.lastFailure) > l.blockFor {
			delete(l.entries, key)
		}
	}
}
