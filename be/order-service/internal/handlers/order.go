package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/common/pagination"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/middleware"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service services.OrderService
}

func NewOrderHandler(service services.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// CreateOrder creates a new order and outbox event in a single transaction.
// @Summary Create order
// @Description Creates a new order along with order items and outbox event atomically.
// @Tags Order
// @Accept json
// @Produce json
// @Param request body dto.CreateOrderRequest true "Create Order Request"
// @Success 201 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	log.Printf("[OrderHandler.CreateOrder] Received POST /api/v1/orders request from %s", c.ClientIP())
	userID := middleware.GetUserID(c)
	if userID == nil || *userID == uuid.Nil {
		log.Printf("[OrderHandler.CreateOrder] Unauthorized: missing or invalid user ID")
		respondError(c, http.StatusUnauthorized, errors.New("unauthorized: missing or invalid user id"))
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[OrderHandler.CreateOrder] Bad request JSON bind error: %v", err)
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		req.IdempotencyKey = strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	}

	result, err := h.service.CreateOrder(c.Request.Context(), *userID, req)
	if err != nil {
		log.Printf("[OrderHandler.CreateOrder] Service returned error: %v", err)
		switch {
		case errors.Is(err, dto.ErrEmptyOrderItems),
			errors.Is(err, dto.ErrInvalidQuantity),
			errors.Is(err, dto.ErrInvalidUnitPrice),
			errors.Is(err, dto.ErrInvalidIdempotencyKey):
			respondError(c, http.StatusBadRequest, err)
		case errors.Is(err, dto.ErrInvalidUserID):
			respondError(c, http.StatusUnauthorized, err)
		case errors.Is(err, dto.ErrTicketTypeNotFound):
			respondError(c, http.StatusNotFound, err)
		case errors.Is(err, dto.ErrInsufficientStock),
			errors.Is(err, dto.ErrOrderInProgress):
			respondError(c, http.StatusConflict, err)
		default:
			respondError(c, http.StatusInternalServerError, err)
		}
		return
	}

	log.Printf("[OrderHandler.CreateOrder] Responding 201 Created for order %s", result.OrderCode)
	respond(c, http.StatusCreated, "order created successfully", result)
}

// ListOrders retrieves paginated list of orders for the authenticated user.
// @Summary List user orders
// @Description Retrieves orders belonging to the authenticated user.
// @Tags Order
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Page limit (default 10, max 100)"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil || *userID == uuid.Nil {
		respondError(c, http.StatusUnauthorized, errors.New("unauthorized: missing or invalid user id"))
		return
	}

	var query dto.BaseQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	orders, total, err := h.service.ListMyOrders(c.Request.Context(), *userID, query.GetPage(), query.GetLimit())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	meta := pagination.NewPaginationMeta(total, query.GetPage(), query.GetLimit())
	respondPaginated(c, orders, meta)
}

// GetOrderByID gets order details by ID.
// @Summary Get order by ID
// @Description Retrieves an order by its unique ID.
// @Tags Order
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == nil || *userID == uuid.Nil {
		respondError(c, http.StatusUnauthorized, errors.New("unauthorized: missing or invalid user id"))
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(c, http.StatusBadRequest, errors.New("invalid order id format"))
		return
	}

	result, err := h.service.GetOrderByIDForUser(c.Request.Context(), id, *userID)
	if err != nil {
		if errors.Is(err, dto.ErrOrderNotFound) {
			respondError(c, http.StatusNotFound, err)
			return
		}
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	respond(c, http.StatusOK, "order retrieved successfully", result)
}
