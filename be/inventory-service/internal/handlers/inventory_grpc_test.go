package handlers_test

import (
	"context"
	"errors"
	"testing"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbInventory "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/inventory"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/handlers/dto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

type mockInventoryService struct {
	deductStockFn  func(ctx context.Context, req dto.DeductStockRequest) error
	restoreStockFn func(ctx context.Context, req dto.RestoreStockRequest) error
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
	if m.deductStockFn != nil {
		return m.deductStockFn(ctx, req)
	}
	return nil
}

func (m *mockInventoryService) RestoreStock(ctx context.Context, req dto.RestoreStockRequest) error {
	if m.restoreStockFn != nil {
		return m.restoreStockFn(ctx, req)
	}
	return nil
}

func (m *mockInventoryService) HandleOrderCreated(ctx context.Context, eventID string, payload events.OrderCreatedPayload) error {
	return nil
}

func (m *mockInventoryService) RecordOrderReservationFailed(ctx context.Context, orderID uuid.UUID, message string) error {
	return nil
}

func TestInventoryGRPCHandler_DeductStock_Success(t *testing.T) {
	ticketID := uuid.New()
	mockSvc := &mockInventoryService{
		deductStockFn: func(ctx context.Context, req dto.DeductStockRequest) error {
			if len(req.Items) != 1 || req.Items[0].TicketTypeID != ticketID || req.Items[0].Quantity != 2 {
				return errors.New("unexpected payload")
			}
			return nil
		},
	}

	h := handlers.NewInventoryGRPCHandler(mockSvc)
	resp, err := h.DeductStock(context.Background(), &pbInventory.DeductStockRequest{
		Items: []*pbInventory.DeductItem{
			{TicketTypeId: ticketID.String(), Quantity: 2},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatalf("expected success true, got %v", resp)
	}
}

func TestInventoryGRPCHandler_DeductStock_ValidationErrors(t *testing.T) {
	h := handlers.NewInventoryGRPCHandler(&mockInventoryService{})
	ctx := context.Background()

	// 1. Nil request
	_, err := h.DeductStock(ctx, nil)
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil req, got %v", err)
	}

	// 2. Empty items
	_, err = h.DeductStock(ctx, &pbInventory.DeductStockRequest{Items: []*pbInventory.DeductItem{}})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on empty items, got %v", err)
	}

	// 3. Nil item
	_, err = h.DeductStock(ctx, &pbInventory.DeductStockRequest{Items: []*pbInventory.DeductItem{nil}})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil item, got %v", err)
	}

	// 4. Invalid UUID
	_, err = h.DeductStock(ctx, &pbInventory.DeductStockRequest{
		Items: []*pbInventory.DeductItem{
			{TicketTypeId: "invalid-uuid", Quantity: 1},
		},
	})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on bad UUID, got %v", err)
	}

	// 5. Quantity <= 0
	_, err = h.DeductStock(ctx, &pbInventory.DeductStockRequest{
		Items: []*pbInventory.DeductItem{
			{TicketTypeId: uuid.New().String(), Quantity: 0},
		},
	})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on 0 quantity, got %v", err)
	}
}

func TestInventoryGRPCHandler_RestoreStock_Success(t *testing.T) {
	ticketID := uuid.New()
	mockSvc := &mockInventoryService{
		restoreStockFn: func(ctx context.Context, req dto.RestoreStockRequest) error {
			if len(req.Items) != 1 || req.Items[0].TicketTypeID != ticketID || req.Items[0].Quantity != 3 {
				return errors.New("unexpected payload")
			}
			return nil
		},
	}

	h := handlers.NewInventoryGRPCHandler(mockSvc)
	resp, err := h.RestoreStock(context.Background(), &pbInventory.RestoreStockRequest{
		Items: []*pbInventory.RestoreItem{
			{TicketTypeId: ticketID.String(), Quantity: 3},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || !resp.Success {
		t.Fatalf("expected success true, got %v", resp)
	}
}

func TestInventoryGRPCHandler_RestoreStock_ValidationErrors(t *testing.T) {
	h := handlers.NewInventoryGRPCHandler(&mockInventoryService{})
	ctx := context.Background()

	if _, err := h.RestoreStock(ctx, nil); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil req, got %v", err)
	}
	if _, err := h.RestoreStock(ctx, &pbInventory.RestoreStockRequest{}); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on empty items, got %v", err)
	}
	if _, err := h.RestoreStock(ctx, &pbInventory.RestoreStockRequest{
		Items: []*pbInventory.RestoreItem{{TicketTypeId: uuid.New().String(), Quantity: -1}},
	}); grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on negative quantity, got %v", err)
	}
}
