package consumers

import (
	"context"
	"errors"
	"testing"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/handlers/dto"
	"github.com/google/uuid"
)

type mockOrderService struct {
	reservedOrderID uuid.UUID
	failedOrderID   uuid.UUID
	failedMessage   string
	err             error
}

func (m *mockOrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req dto.CreateOrderRequest) (*dto.OrderResponse, error) {
	return nil, nil
}
func (m *mockOrderService) GetOrderByID(ctx context.Context, id uuid.UUID) (*dto.OrderResponse, error) {
	return nil, nil
}
func (m *mockOrderService) GetOrderByIDForUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*dto.OrderResponse, error) {
	return nil, nil
}
func (m *mockOrderService) ListMyOrders(ctx context.Context, userID uuid.UUID, page, limit int) ([]dto.OrderResponse, int64, error) {
	return nil, 0, nil
}
func (m *mockOrderService) HandleInventoryReserved(ctx context.Context, eventID string, payload events.InventoryReservedPayload) error {
	if m.err != nil {
		return m.err
	}
	m.reservedOrderID = payload.OrderID
	return nil
}
func (m *mockOrderService) HandleInventoryReservationFailed(ctx context.Context, eventID string, payload events.InventoryReservationFailedPayload) error {
	if m.err != nil {
		return m.err
	}
	m.failedOrderID = payload.OrderID
	m.failedMessage = payload.Message
	return nil
}

func TestInventoryConsumers(t *testing.T) {
	orderID := uuid.New()
	svc := &mockOrderService{}

	reservedHandler := NewInventoryReservedHandler(svc)
	reservedEnv, err := commonKafka.NewEnvelope(events.EventInventoryReserved, "inventory-service", "", events.InventoryReservedPayload{
		OrderID: orderID,
	})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	if err := reservedHandler(context.Background(), reservedEnv); err != nil {
		t.Fatalf("reservedHandler error: %v", err)
	}
	if svc.reservedOrderID != orderID {
		t.Fatalf("expected reservedOrderID=%s, got %s", orderID, svc.reservedOrderID)
	}

	failedHandler := NewInventoryReservationFailedHandler(svc)
	failedEnv, err := commonKafka.NewEnvelope(events.EventInventoryReservationFailed, "inventory-service", "", events.InventoryReservationFailedPayload{
		OrderID: orderID,
		Message: "Sản phẩm đã hết hàng",
	})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	if err := failedHandler(context.Background(), failedEnv); err != nil {
		t.Fatalf("failedHandler error: %v", err)
	}
	if svc.failedOrderID != orderID || svc.failedMessage != "Sản phẩm đã hết hàng" {
		t.Fatalf("unexpected failed call: id=%s msg=%q", svc.failedOrderID, svc.failedMessage)
	}

	svc.err = errors.New("db error")
	if err := reservedHandler(context.Background(), reservedEnv); err == nil {
		t.Fatal("expected error from reservedHandler when service fails")
	}
	if err := failedHandler(context.Background(), failedEnv); err == nil {
		t.Fatal("expected error from failedHandler when service fails")
	}
}
