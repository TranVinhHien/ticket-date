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

type VenueHandler struct {
	service *services.VenueService
}

func NewVenueHandler(service *services.VenueService) *VenueHandler {
	return &VenueHandler{service: service}
}

func (h *VenueHandler) Create(c *gin.Context) {
	var req dto.CreateVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	venue, err := h.service.Create(c.Request.Context(), req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusCreated, "venue created successfully", venue)
}

func (h *VenueHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	venue, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "venue retrieved successfully", venue)
}

func (h *VenueHandler) List(c *gin.Context) {
	var query dto.VenueQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, err)
		return
	}

	venues, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		respondError(c, err)
		return
	}

	meta := pagination.NewPaginationMeta(total, query.GetPage(), query.GetLimit())
	respondPaginated(c, venues, meta)
}

func (h *VenueHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	var req dto.UpdateVenueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	venue, err := h.service.Update(c.Request.Context(), id, req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "venue updated successfully", venue)
}

func (h *VenueHandler) Delete(c *gin.Context) {
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

	respond(c, http.StatusOK, "venue deleted successfully", nil)
}
