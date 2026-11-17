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

type OrganizerHandler struct {
	service *services.OrganizerService
}

func NewOrganizerHandler(service *services.OrganizerService) *OrganizerHandler {
	return &OrganizerHandler{service: service}
}

func (h *OrganizerHandler) Create(c *gin.Context) {
	var req dto.CreateOrganizerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	organizer, err := h.service.Create(c.Request.Context(), req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusCreated, "organizer created successfully", organizer)
}

func (h *OrganizerHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	organizer, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "organizer retrieved successfully", organizer)
}

func (h *OrganizerHandler) List(c *gin.Context) {
	var query dto.OrganizerQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, err)
		return
	}

	organizers, total, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		respondError(c, err)
		return
	}

	meta := pagination.NewPaginationMeta(total, query.GetPage(), query.GetLimit())
	respondPaginated(c, organizers, meta)
}

func (h *OrganizerHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, dto.ErrNotFound)
		return
	}

	var req dto.UpdateOrganizerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, err)
		return
	}

	userCtx := middleware.GetCurrentUser(c)
	organizer, err := h.service.Update(c.Request.Context(), id, req, userCtx)
	if err != nil {
		respondError(c, err)
		return
	}

	respond(c, http.StatusOK, "organizer updated successfully", organizer)
}

func (h *OrganizerHandler) Delete(c *gin.Context) {
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

	respond(c, http.StatusOK, "organizer deleted successfully", nil)
}
