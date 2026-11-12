package middlewares

import (
	"log/slog"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// RequestLogger logs one line per request with the route TEMPLATE
// (c.FullPath()), never the raw path, query, body, headers or client IP.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "request",
			slog.String("request_id", utils.RequestID(c.Request.Context())),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("actor_type", actorType(route)),
		)
	}
}

func actorType(route string) string {
	switch {
	case strings.HasPrefix(route, "/api/v1/guest-registrations/manage"):
		return "guest_token"
	case strings.HasPrefix(route, "/api/v1/admin"):
		return "admin"
	default:
		return "anonymous"
	}
}
