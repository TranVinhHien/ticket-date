package router

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/middleware"
)

// SetupRouter initializes and configures the gin router for order service.
func SetupRouter(
	orderHandler *handlers.OrderHandler,
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
		orders := api.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("/:id", orderHandler.GetOrderByID)
		}
	}

	return r
}
