package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/cache"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/repository"
	"github.com/google/uuid"
)

const (
	DefaultCurrency      = "VND"
	DefaultOrderValidity = 15 * time.Minute
	EventOrderCreated    = "OrderCreated"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error)
}

type orderService struct {
	repo  repository.OrderRepository
	cache cache.Cache
}

func NewOrderService(repo repository.OrderRepository, cache cache.Cache) OrderService {
	return &orderService{
		repo:  repo,
		cache: cache,
	}
}

func generateOrderCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102150405"), uuid.New().String()[:6])
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102150405"), strings.ToUpper(hex.EncodeToString(bytes)))
}

func (s *orderService) CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	if userID == uuid.Nil {
		return nil, dto.ErrInvalidUserID
	}

	if len(req.Items) == 0 {
		return nil, dto.ErrEmptyOrderItems
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = DefaultCurrency
	}

	var totalAmount int64
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, dto.ErrInvalidQuantity
		}
		if item.UnitPrice < 0 {
			return nil, dto.ErrInvalidUnitPrice
		}
		subtotal := int64(item.Quantity) * item.UnitPrice
		totalAmount += subtotal
	}

	orderID := uuid.New()
	orderCode := generateOrderCode()
	expiresAt := time.Now().Add(DefaultOrderValidity)

	var finalResp *dto.OrderResponse

	err := s.repo.ExecTx(ctx, func(q *db.Queries) error {
		// 1. Insert order
		createdOrder, err := q.CreateOrder(ctx, db.CreateOrderParams{
			ID:          orderID,
			UserID:      userID,
			OrderCode:   orderCode,
			TotalAmount: totalAmount,
			Currency:    currency,
			Status:      "PENDING",
			ExpiresAt:   &expiresAt,
			PaidAt:      nil,
			CreatedBy:   &userID,
		})
		if err != nil {
			return fmt.Errorf("create order: %w", err)
		}

		// 2. Insert order items
		var itemResponses []dto.OrderItemResponse
		for _, item := range req.Items {
			itemID := uuid.New()
			subtotal := int64(item.Quantity) * item.UnitPrice

			createdItem, err := q.CreateOrderItem(ctx, db.CreateOrderItemParams{
				ID:             itemID,
				OrderID:        orderID,
				TicketTypeID:   item.TicketTypeID,
				TicketTypeName: item.TicketTypeName,
				Sku:            item.Sku,
				Quantity:       item.Quantity,
				UnitPrice:      item.UnitPrice,
				Subtotal:       subtotal,
				CreatedBy:      &userID,
			})
			if err != nil {
				return fmt.Errorf("create order item: %w", err)
			}

			itemResponses = append(itemResponses, dto.OrderItemResponse{
				ID:             createdItem.ID,
				OrderID:        createdItem.OrderID,
				TicketTypeID:   createdItem.TicketTypeID,
				TicketTypeName: createdItem.TicketTypeName,
				Sku:            createdItem.Sku,
				Quantity:       createdItem.Quantity,
				UnitPrice:      createdItem.UnitPrice,
				Subtotal:       createdItem.Subtotal,
				CreatedAt:      createdItem.CreatedAt,
			})
		}

		// 3. Prepare payload for outbox
		orderPayload := dto.OrderResponse{
			ID:          createdOrder.ID,
			UserID:      createdOrder.UserID,
			OrderCode:   createdOrder.OrderCode,
			TotalAmount: createdOrder.TotalAmount,
			Currency:    createdOrder.Currency,
			Status:      createdOrder.Status,
			ExpiresAt:   createdOrder.ExpiresAt,
			PaidAt:      createdOrder.PaidAt,
			Items:       itemResponses,
			CreatedAt:   createdOrder.CreatedAt,
			UpdatedAt:   createdOrder.UpdatedAt,
		}

		payloadBytes, err := json.Marshal(orderPayload)
		if err != nil {
			return fmt.Errorf("marshal order outbox payload: %w", err)
		}

		// 4. Insert into outbox_event
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			AggregateID: orderID.String(),
			EventType:   EventOrderCreated,
			Payload:     string(payloadBytes),
			Status:      0,
		})
		if err != nil {
			return fmt.Errorf("create outbox event: %w", err)
		}

		finalResp = &orderPayload
		return nil
	})

	if err != nil {
		return nil, err
	}

	return finalResp, nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error) {
	queries := s.repo.Queries()

	order, err := queries.GetOrderByID(ctx, id)
	if err != nil {
		return nil, dto.ErrOrderNotFound
	}

	items, err := queries.GetOrderItemsByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}

	var itemResponses []dto.OrderItemResponse
	for _, item := range items {
		itemResponses = append(itemResponses, dto.OrderItemResponse{
			ID:             item.ID,
			OrderID:        item.OrderID,
			TicketTypeID:   item.TicketTypeID,
			TicketTypeName: item.TicketTypeName,
			Sku:            item.Sku,
			Quantity:       item.Quantity,
			UnitPrice:      item.UnitPrice,
			Subtotal:       item.Subtotal,
			CreatedAt:      item.CreatedAt,
		})
	}

	return &dto.OrderResponse{
		ID:          order.ID,
		UserID:      order.UserID,
		OrderCode:   order.OrderCode,
		TotalAmount: order.TotalAmount,
		Currency:    order.Currency,
		Status:      order.Status,
		ExpiresAt:   order.ExpiresAt,
		PaidAt:      order.PaidAt,
		Items:       itemResponses,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}, nil
}
