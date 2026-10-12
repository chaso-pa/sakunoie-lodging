package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDHeader carries the request id on the response.
const RequestIDHeader = "X-Request-Id"

// RequestID gives every request a fresh id, puts it on the request context
// (logging.RequestID) and returns it in the X-Request-Id response header.
// An id sent by the client is ignored, so log lines cannot be forged into
// someone else's request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), id))
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// RequestLogger writes one structured line per request.
//
// It logs the route template ("/api/posts/:id"), never the URL: query
// strings carry search text and coordinates. Headers, cookies and bodies
// are not logged either. Requests that match no route are logged without
// their path, since a client can put anything there.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		// RequestID and RequireAuth replace c.Request, so this context has
		// both the request id and, when authenticated, the member id.
		ctx := c.Request.Context()
		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		}
		logger.LogAttrs(ctx, level, "request",
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.Int("bytes", max(c.Writer.Size(), 0)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// Recovery turns a panic into a 500 problem response. Unlike gin.Recovery
// it does not dump the request, whose Cookie header holds the session token.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorContext(c.Request.Context(), "panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				if !c.Writer.Written() {
					abortWithProblem(c, http.StatusInternalServerError, "internal server error")
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}
