package routes

import (
	"github.com/adapolytech/stock-manager/internal/handler"
	"github.com/adapolytech/stock-manager/internal/handler/middleware"
	"github.com/gin-gonic/gin"
)

func SetupProductRoutes(router *gin.Engine, productHandler *handler.ProductHandler, accessSecret string) {
	v1 := router.Group("/api/v1")
	products := v1.Group("/products")
	products.Use(middleware.Auth(accessSecret), middleware.Tenant())

	products.GET("", productHandler.List)
	products.POST("", productHandler.Create)
	products.GET("/:id", productHandler.GetByID)
	products.PUT("/:id", productHandler.Update)
	products.DELETE("/:id", productHandler.Delete)
	products.POST("/:id/variants", productHandler.AddVariant)
	products.PUT("/:id/variants/:variantId", productHandler.UpdateVariant)
	products.DELETE("/:id/variants/:variantId", productHandler.DeleteVariant)
}
