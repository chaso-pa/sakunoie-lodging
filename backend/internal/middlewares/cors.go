package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS policy for the browser calling the API origin directly (root CLAUDE.md
// §13 "Decision B revised"). Credentials are never allowed: the app sends
// Bearer tokens, not cookies.
const (
	corsAllowMethods  = "GET, POST, PATCH, OPTIONS"
	corsAllowHeaders  = "Authorization, Content-Type, Idempotency-Key, If-Match, X-Request-ID"
	corsExposeHeaders = "ETag, X-Request-ID, Retry-After, Content-Disposition"
	corsMaxAge        = "600"
)

// CORS answers cross-origin requests from an exact allowlist of origins
// (CORS_ALLOWED_ORIGINS, validated by config.Load). With an empty list it is a
// no-op and the API is same-origin only.
//
// A preflight (OPTIONS with Access-Control-Request-Method) is answered 204 here
// and aborts the chain, so it never reaches the rate limiter, admin auth or
// Huma. A preflight from an origin that is not allowed also gets a plain 204,
// but without any Access-Control-* header, so the browser blocks the request.
// Other requests continue unchanged; an allowed origin gets
// Access-Control-Allow-Origin on every response, including error envelopes,
// so the browser can read the error code.
func CORS(allowed []string) gin.HandlerFunc {
	if len(allowed) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	origins := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		origins[o] = struct{}{}
	}
	return func(c *gin.Context) {
		h := c.Writer.Header()
		// Responses differ by Origin even when it is absent or not allowed.
		h.Add("Vary", "Origin")
		origin := c.GetHeader("Origin")
		_, ok := origins[origin]
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""
		if preflight {
			h.Add("Vary", "Access-Control-Request-Method, Access-Control-Request-Headers")
			if ok {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", corsAllowMethods)
				h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
				h.Set("Access-Control-Max-Age", corsMaxAge)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		if ok {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
		}
		c.Next()
	}
}
