package middlewares

import (
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	limiter "github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// Per-minute limits of the two unauthenticated auth endpoints.
//
// The per-client limits apply when the request carries a real client
// address. In the default deployment it does not: the Next.js proxy forwards
// no client address, so every guest reaches this server as 127.0.0.1. Keyed
// by that one address, 5 and 3 per minute would be a single allowance for
// all guests together, and anyone could keep everyone else out of signup
// and restore by using it up. Requests without a client address therefore
// share a higher ceiling instead, and the failed-restore lockout
// (FailureLockout) is not applied to them.
//
// The looser shared case is safe, do not tighten it back:
//   - A transfer code has about 62 bits of entropy (36^12). Guessing one
//     online is infeasible at 3 or at 30 attempts per minute; neither the
//     limit nor the lockout is what protects it.
//   - The invite code is one shared secret whose strength is its length.
//     The shared ceiling bounds guessing to 30*60*24 = 43,200 attempts per
//     day, which is why INVITE_CODE should be long and random (the server
//     warns at startup in release mode when it is shorter than 12
//     characters).
//
// Per-client limits take effect once a trusted reverse proxy in front of
// Next.js forwards the client address (TRUST_FORWARDED_HEADERS=true on the
// frontend).
const (
	InviteLimitPerClient   = 5
	TransferLimitPerClient = 3
	InviteLimitShared      = 30
	TransferLimitShared    = 30
)

// RateLimiters holds one middleware per rate-limited endpoint.
//
// Invite and Transfer are keyed by client IP (c.ClientIP(), which is only
// trustworthy after gin.Engine.SetTrustedProxies), or share one bucket when
// no client address was forwarded. The rest are keyed by member id and
// must run after RequireAuth.
//
// The place limits protect two things: the member-generated list of spots
// from being filled by one account, and the upstream services, which the
// whole application shares at about one request per second (see
// services.OSMPlaceProvider) or pays for per request (Google).
type RateLimiters struct {
	Invite    gin.HandlerFunc // POST /api/auth/verify-invite: 5/min per IP, or 30/min shared
	Transfer  gin.HandlerFunc // POST /api/auth/transfer: 3/min per IP, or 30/min shared
	AI        gin.HandlerFunc // POST /api/ai/plan: 10/min per member
	Translate gin.HandlerFunc // POST /api/translate: 30/min per member
	Post      gin.HandlerFunc // POST /api/posts: 20/day per member

	PlaceSearch  gin.HandlerFunc // POST /api/places/search: 20/min per member
	PlaceNearby  gin.HandlerFunc // POST /api/places/nearby: 10/min per member
	PlaceReverse gin.HandlerFunc // POST /api/places/reverse: 20/min per member
	SpotRegister gin.HandlerFunc // POST /api/spots: 40/day per member
}

// Per-member limits of the place endpoints.
const (
	PlaceSearchLimitPerMinute  = 20
	PlaceNearbyLimitPerMinute  = 10
	PlaceReverseLimitPerMinute = 20
	SpotRegisterLimitPerDay    = 40
)

// NewRateLimiters builds the limiters on in-memory stores, which is enough
// for a single instance. Counters reset when the process restarts.
func NewRateLimiters() *RateLimiters {
	// Each limiter gets its own store. A store instance keys its counters by
	// Prefix + ":" + key only, so two limiters sharing one instance would
	// count into the same bucket; separate instances never do. The Prefix is
	// kept distinct as well so this still holds if the stores are ever
	// replaced by one shared backend such as Redis.
	newStore := func(prefix string) limiter.Store {
		return memory.NewStoreWithOptions(limiter.StoreOptions{
			Prefix:          prefix,
			CleanUpInterval: limiter.DefaultCleanUpInterval,
		})
	}
	perIP := func(prefix string, rate limiter.Rate) gin.HandlerFunc {
		return mgin.NewMiddleware(limiter.New(newStore(prefix), rate), limitReached)
	}
	shared := func(prefix string, rate limiter.Rate) gin.HandlerFunc {
		return mgin.NewMiddleware(limiter.New(newStore(prefix), rate), limitReached, sharedKey)
	}
	perMember := func(prefix string, rate limiter.Rate) gin.HandlerFunc {
		return mgin.NewMiddleware(limiter.New(newStore(prefix), rate), limitReached, memberKey)
	}

	return &RateLimiters{
		Invite: byClientAddress(
			perIP("invite", limiter.Rate{Period: time.Minute, Limit: InviteLimitPerClient}),
			shared("invite-shared", limiter.Rate{Period: time.Minute, Limit: InviteLimitShared}),
		),
		Transfer: byClientAddress(
			perIP("transfer", limiter.Rate{Period: time.Minute, Limit: TransferLimitPerClient}),
			shared("transfer-shared", limiter.Rate{Period: time.Minute, Limit: TransferLimitShared}),
		),
		AI:        perMember("ai", limiter.Rate{Period: time.Minute, Limit: 10}),
		Translate: perMember("translate", limiter.Rate{Period: time.Minute, Limit: 30}),
		Post:      perMember("post", limiter.Rate{Period: 24 * time.Hour, Limit: 20}),

		PlaceSearch:  perMember("place-search", limiter.Rate{Period: time.Minute, Limit: PlaceSearchLimitPerMinute}),
		PlaceNearby:  perMember("place-nearby", limiter.Rate{Period: time.Minute, Limit: PlaceNearbyLimitPerMinute}),
		PlaceReverse: perMember("place-reverse", limiter.Rate{Period: time.Minute, Limit: PlaceReverseLimitPerMinute}),
		SpotRegister: perMember("spot-register", limiter.Rate{Period: 24 * time.Hour, Limit: SpotRegisterLimitPerDay}),
	}
}

var memberKey = mgin.WithKeyGetter(func(c *gin.Context) string {
	if id := GetMemberID(c.Request.Context()); id != "" {
		return "member:" + id
	}
	return c.ClientIP() // runs after RequireAuth, so normally unreachable
})

// limitReached answers 429 with Retry-After: the seconds until the window
// resets, from the X-RateLimit-Reset time the limiter has already set.
var limitReached = mgin.WithLimitReachedHandler(func(c *gin.Context) {
	if reset, err := strconv.ParseInt(c.Writer.Header().Get("X-RateLimit-Reset"), 10, 64); err == nil {
		c.Header("Retry-After", strconv.FormatInt(max(reset-time.Now().Unix(), 1), 10))
	}
	abortWithProblem(c, http.StatusTooManyRequests, "rate limit exceeded")
})

// sharedKey puts every request in one bucket.
var sharedKey = mgin.WithKeyGetter(func(*gin.Context) string { return "shared" })

// HasClientAddress reports whether the request carries a real client
// address. It does not when c.ClientIP() is a loopback address: the request
// then came from the Next.js proxy on this host without a forwarded address,
// and the "client" is every guest at once.
//
// The address is parsed before the check, since Next.js reports loopback as
// 127.0.0.1 in production and as ::ffff:127.0.0.1 in development.
func HasClientAddress(c *gin.Context) bool {
	addr, err := netip.ParseAddr(c.ClientIP())
	if err != nil {
		return false
	}
	return !addr.Unmap().IsLoopback()
}

// byClientAddress runs perClient for requests with a client address and
// sharedLimit for the rest.
func byClientAddress(perClient, sharedLimit gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasClientAddress(c) {
			perClient(c)
			return
		}
		sharedLimit(c)
	}
}
