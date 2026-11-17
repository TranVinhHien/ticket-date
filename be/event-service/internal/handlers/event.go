package handlers

import (
	"net/http"

	"github.com/TranVinhHien/ticket-date/be/common/pagination"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/middleware"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventHandler struct {
	service *services.EventService
}

func NewEventHandler(service *services.EventService) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) Create(c *gin.Context) {
	var req dto.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	event, err := h.service.Create(c.Request.Context(), req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusCreated, "event created successfully", event)
}

func (h *EventHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	event, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "event retrieved successfully", event)
}

func (h *EventHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		respondError(c, dto.ErrNotFound)
		return
	}

	event, err := h.service.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "event retrieved successfully", event)
}

func (h *EventHandler) List(c *gin.Context) {
	var query dto.EventQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, err)
		return
	}

	events, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		respondError(c, err)
		return
	}

	meta := pagination.NewPaginationMeta(total, query.GetPage(), query.GetLimit())
	respondPaginated(c, events, meta)
}

func (h *EventHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	var req dto.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	event, err := h.service.Update(c.Request.Context(), id, req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "event updated successfully", event)
}

func (h *EventHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	if err := h.service.Delete(c.Request.Context(), id, userCtx); err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "event deleted successfully", nil)
}
