package middlewares

import "github.com/gin-gonic/gin"

// SecurityHeaders sets headers that apply to every API response. HSTS is set
// by the TLS-terminating edge (Cloudflare), not here (README "Cloudflare
// Tunnel deployment").
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
