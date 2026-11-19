package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockOrderService struct {
	createOrderFn  func(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error)
	getOrderByIDFn func(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error)
}

func (m *mockOrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	if m.createOrderFn != nil {
		return m.createOrderFn(ctx, userID, req)
	}
	return &dto.OrderResponse{ID: uuid.New()}, nil
}

func (m *mockOrderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error) {
	if m.getOrderByIDFn != nil {
		return m.getOrderByIDFn(ctx, id)
	}
	return &dto.OrderResponse{ID: id}, nil
}

func setupTestRouter(svc *mockOrderService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewOrderHandler(svc)

	r.POST("/orders", func(c *gin.Context) {
		// simulate auth middleware if X-User-ID present
		if uidStr := c.GetHeader("X-User-ID"); uidStr != "" {
			if uid, err := uuid.Parse(uidStr); err == nil {
				c.Set("user_id", uid)
			}
		}
		handler.CreateOrder(c)
	})
	r.GET("/orders/:id", handler.GetOrderByID)
	return r
}

func TestOrderHandler_CreateOrder(t *testing.T) {
	userID := uuid.New()
	ticketID := uuid.New()

	t.Run("unauthorized when no user_id in context", func(t *testing.T) {
		svc := &mockOrderService{}
		r := setupTestRouter(svc)

		body, _ := json.Marshal(dto.CreateOrderRequest{
			Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketID, Quantity: 1}},
		})
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("404 when ticket type not found", func(t *testing.T) {
		svc := &mockOrderService{
			createOrderFn: func(ctx context.Context, u uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
				return nil, dto.ErrTicketTypeNotFound
			},
		}
		r := setupTestRouter(svc)

		body, _ := json.Marshal(dto.CreateOrderRequest{
			Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketID, Quantity: 1}},
		})
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("409 when insufficient stock", func(t *testing.T) {
		svc := &mockOrderService{
			createOrderFn: func(ctx context.Context, u uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
				return nil, dto.ErrInsufficientStock
			},
		}
		r := setupTestRouter(svc)

		body, _ := json.Marshal(dto.CreateOrderRequest{
			Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketID, Quantity: 1}},
		})
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", w.Code)
		}
	})

	t.Run("201 created successfully", func(t *testing.T) {
		orderID := uuid.New()
		svc := &mockOrderService{
			createOrderFn: func(ctx context.Context, u uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
				return &dto.OrderResponse{ID: orderID, TotalAmount: 500000, Status: "PENDING"}, nil
			},
		}
		r := setupTestRouter(svc)

		body, _ := json.Marshal(dto.CreateOrderRequest{
			Items: []dto.CreateOrderItemRequest{{TicketTypeID: ticketID, Quantity: 2}},
		})
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", userID.String())
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", w.Code)
		}
	})
}
