package routes

import (
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupStaticRoutes registers non-API Gin routes (probes).
func SetupStaticRoutes(r *gin.Engine, db *gorm.DB) {
	r.GET("/health", handlers.HealthCheck)
	r.GET("/ready", handlers.ReadyCheck(db))
}
