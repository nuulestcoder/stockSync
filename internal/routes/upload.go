package routes

import (
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/handler/middleware"
	"github.com/gin-gonic/gin"
)

func SetupUploadRoutes(router *gin.Engine, uploadHandler *handler.UploadHandler, accessSecret string) {
	uploads := router.Group("/api/v1/uploads")
	uploads.Use(middleware.Auth(accessSecret), middleware.Tenant())
	uploads.POST("/single", uploadHandler.Single)
	uploads.POST("/multiple", uploadHandler.Multiple)
}
