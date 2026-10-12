package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxUploadBytes is the request body cap for the API (10MB).
const MaxUploadBytes = 10 << 20

// MaxBodySize caps the request body. Huma's MaxBodyBytes does not apply to
// multipart requests, so uploads are limited here; reading past the limit
// makes multipart parsing fail and Huma answers 422.
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
