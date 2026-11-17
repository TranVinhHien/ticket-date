package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/middleware"
)

// SetupRouter initializes and configures the gin router for inventory service.
func SetupRouter(
	inventoryHandler *handlers.InventoryHandler,
) *gin.Engine {
	r := gin.Default()

	// Gateway Auth Middleware extracts X-User-ID header passed by API Gateway
	r.Use(middleware.GatewayAuthMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, dto.Response{
			Code:    "200",
			Message: "ok",
			Meta:    nil,
			Data:    nil,
		})
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	api := r.Group("/api/v1")
	{
		inventories := api.Group("/inventories")
		{
			// 1. API Create Inventory (batch)
			inventories.POST("", inventoryHandler.CreateBatch)

			// 2. API Update Inventory Stock (batch)
			inventories.PUT("/stock", inventoryHandler.UpdateStockBatch)

			// 3. API Get Current Stock (by ticket_type_ids)
			inventories.GET("/stock", inventoryHandler.GetStock)
		}
	}

	return r
}
