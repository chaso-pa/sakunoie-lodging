package middlewares

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Rate-limit classes (RATE_LIMIT_*_PER_MIN).
const (
	LimitCreate = "create"
	LimitManage = "manage"
	LimitSurvey = "survey"
)

// ClassifyRoute maps a Gin route template to a rate-limit class ("" = none).
func ClassifyRoute(method, route string) string {
	switch {
	case method == http.MethodPost && route == "/api/v1/guest-registrations":
		return LimitCreate
	case strings.HasPrefix(route, "/api/v1/guest-registrations/manage"):
		return LimitManage
	case method == http.MethodPost && route == "/api/v1/surveys":
		return LimitSurvey
	}
	return ""
}

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// ConfigureClientIP makes c.ClientIP() trust only an X-Real-IP header that a
// TRUSTED_PROXIES peer sent. X-Forwarded-For is never read: the Next rewrite
// forwards the client's XFF unchanged without appending the peer, so a
// trusted-hop XFF is client-controlled. The edge proxy must overwrite
// X-Real-IP with the peer address (see README "Client IP").
func ConfigureClientIP(r *gin.Engine, trusted []string) error {
	r.RemoteIPHeaders = []string{"X-Real-IP"}
	return r.SetTrustedProxies(trusted)
}

// RateLimiter is an in-memory per-IP token bucket per class. The key is
// c.ClientIP() as configured by ConfigureClientIP. IPs are never logged.
type RateLimiter struct {
	mu         sync.Mutex
	perMin     map[string]int
	buckets    map[string]*bucket
	maxBuckets int
	ttl        time.Duration
	lastSweep  time.Time
	now        func() time.Time
	logger     *slog.Logger
}

// defaultMaxBuckets caps memory under an address-rotation flood (~100 B per
// bucket).
const defaultMaxBuckets = 100_000

func NewRateLimiter(perMin map[string]int, logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		perMin:     perMin,
		buckets:    map[string]*bucket{},
		maxBuckets: defaultMaxBuckets,
		ttl:        10 * time.Minute,
		now:        time.Now,
		logger:     logger,
	}
}

func (rl *RateLimiter) allow(class, ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	if now.Sub(rl.lastSweep) > time.Minute {
		rl.sweep(now)
	}
	key := class + "|" + ip
	b, ok := rl.buckets[key]
	if !ok {
		if len(rl.buckets) >= rl.maxBuckets {
			rl.evict(now)
		}
		n := rl.perMin[class]
		b = &bucket{limiter: rate.NewLimiter(rate.Limit(float64(n)/60.0), n)}
		rl.buckets[key] = b
	}
	b.lastSeen = now
	return b.limiter.AllowN(now, 1)
}

func (rl *RateLimiter) sweep(now time.Time) {
	for k, b := range rl.buckets {
		if now.Sub(b.lastSeen) > rl.ttl {
			delete(rl.buckets, k)
		}
	}
	rl.lastSweep = now
}

// evict runs when the map is full: it drops expired buckets, and if that is
// not enough, arbitrary ones (map order) down to 90% of the cap. An evicted
// client merely starts with a fresh bucket.
func (rl *RateLimiter) evict(now time.Time) {
	rl.sweep(now)
	target := rl.maxBuckets * 9 / 10
	for k := range rl.buckets {
		if len(rl.buckets) <= target {
			break
		}
		delete(rl.buckets, k)
	}
	rl.logger.Warn("rate limiter bucket cap reached", "max_buckets", rl.maxBuckets)
}

// Middleware must run after routing (it reads c.FullPath()).
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		class := ClassifyRoute(c.Request.Method, c.FullPath())
		if class == "" || rl.perMin[class] <= 0 {
			c.Next()
			return
		}
		if !rl.allow(class, c.ClientIP()) {
			rl.logger.Warn("rate limited",
				"request_id", utils.RequestID(c.Request.Context()),
				"route", c.FullPath(),
			)
			c.Header("Retry-After", strconv.Itoa(60/max(rl.perMin[class], 1)+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				models.EnvelopeFor(c.Request.Context(), http.StatusTooManyRequests, "rate_limited", "too many requests"))
			return
		}
		c.Next()
	}
}
