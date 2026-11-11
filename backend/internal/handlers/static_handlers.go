package handlers

import (
	"net/http"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/database"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthCheck is the liveness probe.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Server is healthy",
	})
}

// ReadyCheck is the readiness probe: DB ping with a 1 s timeout.
func ReadyCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil || database.Ping(c.Request.Context(), db, time.Second) != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
