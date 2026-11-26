package consumers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/services"
	"github.com/google/uuid"
)

type mockInventoryService struct {
	calledOrderID    uuid.UUID
	calledEventID    string
	calledItems      []events.OrderCreatedItem
	err              error
	failedOrderIDs   []uuid.UUID
	failedMessages   []string
	recordFailedErr  error
}

func (m *mockInventoryService) CreateInventoryBatch(ctx context.Context, userID *uuid.UUID, req dto.CreateInventoryBatchRequest) ([]dto.InventoryResponse, error) {
	return nil, nil
}
func (m *mockInventoryService) UpdateInventoryStockBatch(ctx context.Context, userID *uuid.UUID, req dto.UpdateInventoryBatchRequest) ([]dto.StockResponse, error) {
	return nil, nil
}
func (m *mockInventoryService) GetCurrentStock(ctx context.Context, ticketTypeID uuid.UUID) (*dto.StockResponse, error) {
	return nil, nil
}
func (m *mockInventoryService) DeductStock(ctx context.Context, req dto.DeductStockRequest) error {
	return nil
}
func (m *mockInventoryService) RestoreStock(ctx context.Context, req dto.RestoreStockRequest) error {
	return nil
}
func (m *mockInventoryService) HandleOrderCreated(ctx context.Context, eventID string, payload events.OrderCreatedPayload) error {
	if m.err != nil {
		return m.err
	}
	m.calledEventID = eventID
	m.calledOrderID = payload.ID
	m.calledItems = payload.Items
	return nil
}
func (m *mockInventoryService) RecordOrderReservationFailed(ctx context.Context, orderID uuid.UUID, message string) error {
	if m.recordFailedErr != nil {
		return m.recordFailedErr
	}
	m.failedOrderIDs = append(m.failedOrderIDs, orderID)
	m.failedMessages = append(m.failedMessages, message)
	return nil
}

func TestOrderCreatedHandler(t *testing.T) {
	orderID := uuid.New()
	ticketTypeID := uuid.New()
	svc := &mockInventoryService{}
	handler := NewOrderCreatedHandler(svc)

	env, err := commonKafka.NewEnvelope(events.TopicOrderEvents, "order-service", "", events.OrderCreatedPayload{
		ID:        orderID,
		OrderCode: "ORD-TEST-001",
		Items: []events.OrderCreatedItem{
			{TicketTypeID: ticketTypeID, Quantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}

	if err := handler(context.Background(), env); err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if svc.calledOrderID != orderID || svc.calledEventID != env.EventID || len(svc.calledItems) != 1 {
		t.Fatalf("unexpected service call: orderID=%s eventID=%s items=%d", svc.calledOrderID, svc.calledEventID, len(svc.calledItems))
	}

	// Insufficient stock -> stops immediately (returns nil) and records out-of-stock message
	svc.err = fmt.Errorf("%w: held=0 expected=1", services.ErrInsufficientInventory)
	if err := handler(context.Background(), env); err != nil {
		t.Fatalf("expected nil on insufficient inventory to stop retries, got: %v", err)
	}
	if len(svc.failedOrderIDs) != 1 || svc.failedOrderIDs[0] != orderID || svc.failedMessages[0] != outOfStockMessage {
		t.Fatalf("expected out-of-stock failure recorded, got ids=%v msgs=%v", svc.failedOrderIDs, svc.failedMessages)
	}

	// Generic DB/server error -> returns error and records "Lỗi server" on final attempt
	svc.err = errors.New("db error")
	if err := handler(context.Background(), env); err == nil {
		t.Fatal("expected error when service fails with server error")
	}
	if len(svc.failedOrderIDs) != 2 || svc.failedMessages[1] != serverErrorMessage {
		t.Fatalf("expected server error failure recorded, got ids=%v msgs=%v", svc.failedOrderIDs, svc.failedMessages)
	}
}
