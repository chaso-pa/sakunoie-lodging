package routes

import (
	"github.com/chaso-pa/gin-template/internal/handlers"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func SetupStaticRoutes(r *gin.Engine) {
	r.GET("/health", handlers.HealthCheck)
	r.GET("/hello", handlers.HelloWorld)
}

func SetupHumaRoutes(api huma.API) {
	huma.Get(api, "/greeting/{name}", handlers.HumaHelloWorld)
}
