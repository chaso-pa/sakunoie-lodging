package routes

import (
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/handlers"
	"github.com/gin-gonic/gin"
)

func SetupStaticRoutes(r *gin.Engine) {
	r.GET("/health", handlers.HealthCheck)
}
