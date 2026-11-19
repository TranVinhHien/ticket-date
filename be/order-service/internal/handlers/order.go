package handlers

import (
	"errors"
	"net/http"

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
	userID := middleware.GetUserID(c)
	if userID == nil || *userID == uuid.Nil {
		respondError(c, http.StatusUnauthorized, errors.New("unauthorized: missing or invalid user id"))
		return
	}

	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	result, err := h.service.CreateOrder(c.Request.Context(), *userID, req)
	if err != nil {
		switch {
		case errors.Is(err, dto.ErrEmptyOrderItems),
			errors.Is(err, dto.ErrInvalidQuantity),
			errors.Is(err, dto.ErrInvalidUnitPrice):
			respondError(c, http.StatusBadRequest, err)
		case errors.Is(err, dto.ErrInvalidUserID):
			respondError(c, http.StatusUnauthorized, err)
		default:
			respondError(c, http.StatusInternalServerError, err)
		}
		return
	}

	respond(c, http.StatusCreated, "order created successfully", result)
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
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(c, http.StatusBadRequest, errors.New("invalid order id format"))
		return
	}

	result, err := h.service.GetOrderByID(c.Request.Context(), id)
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
