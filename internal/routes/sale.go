package routes

import (
	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/handler/middleware"
	"github.com/gin-gonic/gin"
)

func SetupSaleRoutes(router *gin.Engine, saleHandler *handler.SaleHandler, accessSecret string) {
	sales := router.Group("/api/v1/sales")
	sales.Use(middleware.Auth(accessSecret), middleware.Tenant())

	sales.GET("", saleHandler.List)
	sales.GET("/dashboard", saleHandler.Dashboard)
	sales.GET("/reports", saleHandler.Dashboard)
	sales.POST("", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager, domain.UserRoleSeller), saleHandler.Create)
	sales.GET("/:id", saleHandler.Get)
	sales.PUT("/:id/status", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), saleHandler.Transition)
}
