package middlewares

import "github.com/gin-gonic/gin"

// ForRoute applies handler only to the route with the given method and full
// Gin path (including the group prefix, e.g. "/api/posts/:id"); other
// requests pass through.
//
// Huma registers every operation of an API on one Gin group, so per-route
// Gin middleware is attached to the group through this filter. It must be
// added with group.Use before the routes are registered.
func ForRoute(method, fullPath string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == method && c.FullPath() == fullPath {
			handler(c)
			return
		}
		c.Next()
	}
}
