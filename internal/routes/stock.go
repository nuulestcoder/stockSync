package routes

import (
	"github.com/adapolytech/stock-manager/internal/domain"
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/handler/middleware"
	"github.com/gin-gonic/gin"
)

func SetupStockRoutes(router *gin.Engine, stockHandler *handler.StockHandler, accessSecret string) {
	stock := router.Group("/api/v1/stock")
	stock.Use(middleware.Auth(accessSecret), middleware.Tenant())

	stock.GET("", stockHandler.List)
	stock.GET("/alerts", stockHandler.Alerts)
	stock.GET("/dashboard", stockHandler.Dashboard)
	stock.GET("/requests", stockHandler.ListRequests)
	stock.POST("/requests/inbound", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), stockHandler.CreateInbound)
	stock.POST("/requests/outbound", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager, domain.UserRoleSeller), stockHandler.CreateOutbound)
	stock.POST("/requests/:id/approve", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), stockHandler.Approve)
	stock.POST("/requests/:id/reject", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), stockHandler.Reject)
	stock.POST("/requests/:id/cancel", stockHandler.Cancel)
	stock.POST("/requests/:id/complete", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), stockHandler.Complete)
	stock.GET("/:variantId/movements", stockHandler.Movements)
	stock.GET("/:variantId", stockHandler.Get)
	stock.POST("/:variantId/move", middleware.RequireRoles(domain.UserRoleAdmin, domain.UserRoleManager), stockHandler.Move)
}
