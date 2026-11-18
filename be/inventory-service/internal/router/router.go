package router

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database"
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

	// Health check with DB and Redis probe
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "ok"
		if pool := database.GetPool(); pool == nil || pool.Ping(ctx) != nil {
			dbStatus = "down"
		}

		redisStatus := "ok"
		if err := database.PingRedis(ctx); err != nil {
			redisStatus = "down"
		}

		statusCode := 200
		statusMsg := "ok"
		if dbStatus != "ok" {
			statusCode = 503
			statusMsg = "degraded"
		}

		c.JSON(statusCode, dto.Response{
			Code:    fmt.Sprintf("%d", statusCode),
			Message: statusMsg,
			Data: gin.H{
				"database": dbStatus,
				"redis":    redisStatus,
			},
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

			// 3. API Get Current Stock (by ticket_type_id query param or path param)
			inventories.GET("/stock", inventoryHandler.GetStock)
			inventories.GET("/stock/:ticket_type_id", inventoryHandler.GetStock)
		}
	}

	return r
}
