package middlewares

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// Recovery replaces gin.Recovery, which dumps the request (path with query)
// to the log. Only the panic value's type is logged.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, rec any) {
		logger.Error("panic recovered",
			"request_id", utils.RequestID(c.Request.Context()),
			"route", c.FullPath(),
			"panic_type", fmt.Sprintf("%T", rec),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			models.EnvelopeFor(c.Request.Context(), http.StatusInternalServerError, "", "internal error"))
	})
}

// NoRoute answers unknown paths with the error envelope.
func NoRoute(c *gin.Context) {
	c.JSON(http.StatusNotFound, models.EnvelopeFor(c.Request.Context(), http.StatusNotFound, "", "not found"))
}
