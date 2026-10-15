package routes

import (
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(router *gin.Engine, authHandler *handler.AuthHandler) {
	v1 := router.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
}
