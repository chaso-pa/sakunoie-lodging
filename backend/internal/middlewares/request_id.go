package middlewares

import (
	"regexp"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-Id"

var requestIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// RequestID accepts a well-formed inbound X-Request-Id or generates one, and
// puts it in the request context and the response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !requestIDRe.MatchString(id) {
			id = uuid.NewString()
		}
		c.Request = c.Request.WithContext(utils.WithRequestID(c.Request.Context(), id))
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}
