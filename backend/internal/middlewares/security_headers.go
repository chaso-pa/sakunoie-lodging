package middlewares

import "github.com/gin-gonic/gin"

// SecurityHeaders sets headers that apply to every API response. HSTS is set
// by the TLS-terminating edge, not here (README "Client IP and the edge proxy").
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
