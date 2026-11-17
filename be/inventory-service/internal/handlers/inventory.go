package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/middleware"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InventoryHandler struct {
	service services.InventoryService
}

func NewInventoryHandler(service services.InventoryService) *InventoryHandler {
	return &InventoryHandler{service: service}
}

// CreateBatch creates initial inventory records in batch.
// @Summary Create batch inventory
// @Description Batch creates inventory records for ticket types. Fails if any already exists.
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body dto.CreateInventoryBatchRequest true "Batch Create Request"
// @Success 201 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/inventories [post]
func (h *InventoryHandler) CreateBatch(c *gin.Context) {
	var req dto.CreateInventoryBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	userID := middleware.GetUserID(c)
	result, err := h.service.CreateInventoryBatch(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, dto.ErrInventoryAlreadyExists):
			respondError(c, http.StatusConflict, err)
		case errors.Is(err, dto.ErrEmptyBatch), errors.Is(err, dto.ErrDuplicateTicketTypeID):
			respondError(c, http.StatusBadRequest, err)
		default:
			respondError(c, http.StatusInternalServerError, err)
		}
		return
	}

	respond(c, http.StatusCreated, "inventory created successfully", result)
}

// UpdateStockBatch updates inventory stock in batch.
// @Summary Batch update inventory stock
// @Description Updates stock for existing inventory records by Admin.
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body dto.UpdateInventoryBatchRequest true "Batch Update Request"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/inventories/stock [put]
func (h *InventoryHandler) UpdateStockBatch(c *gin.Context) {
	var req dto.UpdateInventoryBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	userID := middleware.GetUserID(c)
	result, err := h.service.UpdateInventoryStockBatch(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, dto.ErrInventoryNotFound):
			respondError(c, http.StatusNotFound, err)
		case errors.Is(err, dto.ErrInvalidQuantity), errors.Is(err, dto.ErrEmptyBatch), errors.Is(err, dto.ErrDuplicateTicketTypeID):
			respondError(c, http.StatusBadRequest, err)
		default:
			respondError(c, http.StatusInternalServerError, err)
		}
		return
	}

	respond(c, http.StatusOK, "inventory stock updated successfully", result)
}

// GetStock returns current stock for a list of ticket_type_ids.
// @Summary Get current stock
// @Description Returns current stock for a list of ticket type IDs.
// @Tags Inventory
// @Accept json
// @Produce json
// @Param ticket_type_ids query string false "Comma separated ticket type UUIDs"
// @Param request body dto.GetStockRequest false "Optional JSON body with ticket_type_ids"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/inventories/stock [get]
func (h *InventoryHandler) GetStock(c *gin.Context) {
	var ids []uuid.UUID

	// Check if body is present (POST or GET with JSON body)
	if c.Request.ContentLength > 0 && strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		var req dto.GetStockRequest
		if err := c.ShouldBindJSON(&req); err == nil && len(req.TicketTypeIDs) > 0 {
			ids = req.TicketTypeIDs
		}
	}

	// Also parse from query parameters if not found in body
	if len(ids) == 0 {
		rawQuery := c.QueryArray("ticket_type_ids")
		if len(rawQuery) == 0 {
			rawParam := c.Query("ticket_type_ids")
			if rawParam != "" {
				rawQuery = strings.Split(rawParam, ",")
			}
		}

		for _, item := range rawQuery {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			parsed, err := uuid.Parse(trimmed)
			if err != nil {
				respondError(c, http.StatusBadRequest, errors.New("invalid ticket_type_id format: "+trimmed))
				return
			}
			ids = append(ids, parsed)
		}
	}

	result, err := h.service.GetCurrentStock(c.Request.Context(), ids)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	respond(c, http.StatusOK, "current stock retrieved successfully", result)
}
