package middlewares

import "github.com/gin-gonic/gin"

// HSTSValue is sent in release mode: two years, subdomains included.
const HSTSValue = "max-age=63072000; includeSubDomains"

// SecurityHeaders sets the response headers every reply carries, whatever
// its status: it runs first on the engine, so replies written by later
// middleware (401, 429), by Huma (422) and for unknown routes (404) have
// them too.
//
//   - Cache-Control: no-store. Responses are per member (feeds, visits,
//     profile); neither the browser nor a shared proxy may keep a copy.
//   - X-Content-Type-Options: nosniff. JSON is never to be guessed as HTML.
//   - X-Frame-Options: DENY. Nothing here is meant to be framed.
//   - Strict-Transport-Security, in release mode only: local development
//     runs over plain HTTP.
func SecurityHeaders(release bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		if release {
			h.Set("Strict-Transport-Security", HSTSValue)
		}
		c.Next()
	}
}
