package router

import (
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	organizerHandler *handlers.OrganizerHandler,
	venueHandler *handlers.VenueHandler,
	eventHandler *handlers.EventHandler,
) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, dto.Response{
			Code:    "200",
			Message: "ok",
			Meta:    nil,
			Data:    nil,
		})
	})

	api := r.Group("/api/v1")
	api.Use(middleware.UserContextMiddleware())
	{
		// Organizers
		organizers := api.Group("/organizers")
		{
			organizers.POST("", organizerHandler.Create)
			organizers.GET("", organizerHandler.List)
			organizers.GET("/:id", organizerHandler.GetByID)
			organizers.PUT("/:id", organizerHandler.Update)
			organizers.DELETE("/:id", organizerHandler.Delete)
		}

		// Venues
		venues := api.Group("/venues")
		{
			venues.POST("", venueHandler.Create)
			venues.GET("", venueHandler.List)
			venues.GET("/:id", venueHandler.GetByID)
			venues.PUT("/:id", venueHandler.Update)
			venues.DELETE("/:id", venueHandler.Delete)
		}

		// Events (Ticket types are managed directly within Events)
		events := api.Group("/events")
		{
			events.POST("", eventHandler.Create)
			events.GET("", eventHandler.List)
			events.GET("/slug/:slug", eventHandler.GetBySlug)
			events.GET("/:id", eventHandler.GetByID)
			events.PUT("/:id", eventHandler.Update)
			events.DELETE("/:id", eventHandler.Delete)
		}
	}

	return r
}
