package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/cache"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	repo            repository.OrderRepository
	cache           cache.Cache
	inventoryClient pbInventory.InventoryServiceClient
	eventClient     pbEvent.EventServiceClient
}

func NewOrderService(
	repo repository.OrderRepository,
	cache cache.Cache,
	inventoryClient pbInventory.InventoryServiceClient,
	eventClient pbEvent.EventServiceClient,
) OrderService {
	return &orderService{
		repo:            repo,
		cache:           cache,
		inventoryClient: inventoryClient,
		eventClient:     eventClient,
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
	log.Printf("[OrderService.CreateOrder] Start creating order: userID=%s, rawItemsCount=%d, requestedCurrency=%q", userID, len(req.Items), req.Currency)

	if userID == uuid.Nil {
		log.Printf("[OrderService.CreateOrder] Validation failed: invalid user ID (nil)")
		return nil, dto.ErrInvalidUserID
	}

	if len(req.Items) == 0 {
		log.Printf("[OrderService.CreateOrder] Validation failed: empty order items")
		return nil, dto.ErrEmptyOrderItems
	}

	// Validate items and merge quantities by ticket_type_id
	mergedQuantities := make(map[uuid.UUID]int32)
	var uniqueTicketTypeIDs []uuid.UUID

	for _, item := range req.Items {
		if item.TicketTypeID == uuid.Nil {
			log.Printf("[OrderService.CreateOrder] Validation failed: ticket_type_id is nil")
			return nil, dto.ErrTicketTypeNotFound
		}
		if item.Quantity <= 0 {
			log.Printf("[OrderService.CreateOrder] Validation failed: invalid quantity %d for ticket %s", item.Quantity, item.TicketTypeID)
			return nil, dto.ErrInvalidQuantity
		}
		if _, exists := mergedQuantities[item.TicketTypeID]; !exists {
			uniqueTicketTypeIDs = append(uniqueTicketTypeIDs, item.TicketTypeID)
		}
		mergedQuantities[item.TicketTypeID] += item.Quantity
	}

	log.Printf("[OrderService.CreateOrder] Merged items into %d unique ticket types: %+v", len(uniqueTicketTypeIDs), mergedQuantities)

	// Fetch ticket types details from Event Service via gRPC
	ticketMap := make(map[uuid.UUID]*pbEvent.TicketTypeDetail)
	if s.eventClient != nil {
		idStrs := make([]string, len(uniqueTicketTypeIDs))
		for i, id := range uniqueTicketTypeIDs {
			idStrs[i] = id.String()
		}

		log.Printf("[OrderService.CreateOrder] Fetching ticket types from EventService via gRPC: ids=%v", idStrs)
		eventResp, err := s.eventClient.GetTicketTypes(ctx, &pbEvent.GetTicketTypesRequest{
			TicketTypeIds: idStrs,
		})
		if err != nil {
			log.Printf("[OrderService.CreateOrder] Error calling EventService.GetTicketTypes: %v", err)
			return nil, fmt.Errorf("failed to fetch ticket types from event service: %w", err)
		}
		if eventResp == nil {
			log.Printf("[OrderService.CreateOrder] EventService returned nil response")
			return nil, fmt.Errorf("event service returned nil response")
		}

		log.Printf("[OrderService.CreateOrder] Received %d ticket types from EventService", len(eventResp.TicketTypes))
		for _, tt := range eventResp.TicketTypes {
			id, err := uuid.Parse(tt.Id)
			if err == nil {
				ticketMap[id] = tt
			}
		}

		// Verify all requested ticket types exist and have non-negative prices
		for _, id := range uniqueTicketTypeIDs {
			tt, exists := ticketMap[id]
			if !exists || tt == nil {
				log.Printf("[OrderService.CreateOrder] Ticket type not found in EventService: id=%s", id)
				return nil, fmt.Errorf("%w: %s", dto.ErrTicketTypeNotFound, id.String())
			}
			if tt.Price < 0 {
				log.Printf("[OrderService.CreateOrder] Invalid negative unit price (%d) for ticket: %s", tt.Price, id)
				return nil, dto.ErrInvalidUnitPrice
			}
		}
	} else {
		log.Printf("[OrderService.CreateOrder] Error: EventService client not initialized")
		return nil, errors.New("event service client not available")
	}

	// Determine currency & calculate total amount
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		if firstTT := ticketMap[uniqueTicketTypeIDs[0]]; firstTT != nil && firstTT.Currency != "" {
			currency = firstTT.Currency
		} else {
			currency = DefaultCurrency
		}
	}

	var totalAmount int64
	for _, id := range uniqueTicketTypeIDs {
		qty := mergedQuantities[id]
		tt := ticketMap[id]
		subtotal := int64(qty) * tt.Price
		totalAmount += subtotal
	}

	log.Printf("[OrderService.CreateOrder] Total amount calculated: %d %s across %d ticket types", totalAmount, currency, len(uniqueTicketTypeIDs))

	// Deduct stock via Inventory Service gRPC
	if s.inventoryClient != nil {
		deductItems := make([]*pbInventory.DeductItem, len(uniqueTicketTypeIDs))
		for i, id := range uniqueTicketTypeIDs {
			deductItems[i] = &pbInventory.DeductItem{
				TicketTypeId: id.String(),
				Quantity:     mergedQuantities[id],
			}
		}

		log.Printf("[OrderService.CreateOrder] Deducting stock via InventoryService gRPC: itemsCount=%d", len(deductItems))
		resp, err := s.inventoryClient.DeductStock(ctx, &pbInventory.DeductStockRequest{
			Items: deductItems,
		})
		if err != nil {
			if isInsufficient, id := grpcErrors.IsInsufficientStock(err); isInsufficient {
				log.Printf("[OrderService.CreateOrder] Insufficient stock error from InventoryService: ticket_type_id=%s, msg=%s", id, grpcErrors.Message(err))
				return nil, fmt.Errorf("%w: %s (ticket_type_id: %s)", dto.ErrInsufficientStock, grpcErrors.Message(err), id)
			}
			if status.Code(err) == codes.InvalidArgument {
				log.Printf("[OrderService.CreateOrder] Invalid argument error from InventoryService: msg=%s", grpcErrors.Message(err))
				return nil, fmt.Errorf("%w: %s", dto.ErrInvalidQuantity, grpcErrors.Message(err))
			}
			log.Printf("[OrderService.CreateOrder] Failed to deduct inventory: %v", err)
			return nil, fmt.Errorf("failed to deduct inventory: %w", err)
		}
		if resp == nil || !resp.Success {
			log.Printf("[OrderService.CreateOrder] Inventory deduction unconfirmed by service")
			return nil, fmt.Errorf("%w: stock deduction unconfirmed", dto.ErrInsufficientStock)
		}
		log.Printf("[OrderService.CreateOrder] Stock deduction confirmed successfully by InventoryService")
	}

	orderID := uuid.New()
	orderCode := generateOrderCode()
	expiresAt := time.Now().Add(DefaultOrderValidity)

	var finalResp *dto.OrderResponse

	log.Printf("[OrderService.CreateOrder] Beginning DB transaction for orderID=%s, orderCode=%s", orderID, orderCode)
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
			log.Printf("[OrderService.CreateOrder] DB error creating order: %v", err)
			return fmt.Errorf("create order: %w", err)
		}

		// 2. Insert order items
		var itemResponses []dto.OrderItemResponse
		for _, id := range uniqueTicketTypeIDs {
			qty := mergedQuantities[id]
			tt := ticketMap[id]
			itemID := uuid.New()
			subtotal := int64(qty) * tt.Price

			createdItem, err := q.CreateOrderItem(ctx, db.CreateOrderItemParams{
				ID:             itemID,
				OrderID:        orderID,
				TicketTypeID:   id,
				TicketTypeName: tt.Name,
				Sku:            tt.Sku,
				Quantity:       qty,
				UnitPrice:      tt.Price,
				Subtotal:       subtotal,
				CreatedBy:      &userID,
			})
			if err != nil {
				log.Printf("[OrderService.CreateOrder] DB error creating order item for ticket %s: %v", id, err)
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
			log.Printf("[OrderService.CreateOrder] Marshal error outbox payload: %v", err)
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
			log.Printf("[OrderService.CreateOrder] DB error creating outbox event: %v", err)
			return fmt.Errorf("create outbox event: %w", err)
		}

		finalResp = &orderPayload
		return nil
	})

	if err != nil {
		log.Printf("[OrderService.CreateOrder] Transaction failed: %v", err)
		return nil, err
	}

	log.Printf("[OrderService.CreateOrder] Order created successfully: orderCode=%s, orderID=%s, totalAmount=%d %s", orderCode, orderID, totalAmount, currency)
	return finalResp, nil
}

func (s *orderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error) {
	log.Printf("[OrderService.GetOrderByID] Querying order by id=%s", id)
	queries := s.repo.Queries()

	order, err := queries.GetOrderByID(ctx, id)
	if err != nil {
		log.Printf("[OrderService.GetOrderByID] Order not found for id=%s: %v", id, err)
		return nil, dto.ErrOrderNotFound
	}

	items, err := queries.GetOrderItemsByOrderID(ctx, id)
	if err != nil {
		log.Printf("[OrderService.GetOrderByID] Error querying order items for id=%s: %v", id, err)
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

	log.Printf("[OrderService.GetOrderByID] Retrieved order %s with %d items", order.OrderCode, len(itemResponses))
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
