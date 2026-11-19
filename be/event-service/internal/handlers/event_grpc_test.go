package handlers_test

import (
	"context"
	"testing"

	grpcErrors "github.com/TranVinhHien/ticket-date/be/common/grpc/errors"
	pbEvent "github.com/TranVinhHien/ticket-date/be/common/grpc/pb/event"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/services"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

type mockEventRepo struct {
	findTicketTypesByIDsFn func(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error)
}

func (m *mockEventRepo) Create(ctx context.Context, event *models.Event) error { return nil }
func (m *mockEventRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	return nil, nil
}
func (m *mockEventRepo) FindBySlug(ctx context.Context, slug string) (*models.Event, error) {
	return nil, nil
}
func (m *mockEventRepo) FindBySKU(ctx context.Context, sku string) (*models.TicketType, error) {
	return nil, nil
}
func (m *mockEventRepo) FindTicketTypesByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error) {
	if m.findTicketTypesByIDsFn != nil {
		return m.findTicketTypesByIDsFn(ctx, ids)
	}
	return nil, nil
}
func (m *mockEventRepo) List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	return nil, 0, nil
}
func (m *mockEventRepo) Update(ctx context.Context, event *models.Event) error                     { return nil }
func (m *mockEventRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error { return nil }

func TestEventGRPCHandler_GetTicketTypes_Success(t *testing.T) {
	ttID := uuid.New()
	eventID := uuid.New()
	desc := "VIP access"

	repo := &mockEventRepo{
		findTicketTypesByIDsFn: func(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error) {
			return []*models.TicketType{
				{
					ID:          ttID,
					EventID:     eventID,
					SKU:         "VIP-01",
					Name:        "VIP Ticket",
					Description: &desc,
					Price:       500000,
					Currency:    "VND",
					MaxPerOrder: 4,
					Status:      models.TicketTypeStatusActive,
				},
			}, nil
		},
	}

	eventSvc := services.NewEventService(repo, nil, nil)
	h := handlers.NewEventGRPCHandler(eventSvc)

	resp, err := h.GetTicketTypes(context.Background(), &pbEvent.GetTicketTypesRequest{
		TicketTypeIds: []string{ttID.String()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.TicketTypes) != 1 {
		t.Fatalf("expected 1 ticket type, got %d", len(resp.TicketTypes))
	}
	tt := resp.TicketTypes[0]
	if tt.Id != ttID.String() || tt.Name != "VIP Ticket" || tt.Price != 500000 || tt.Sku != "VIP-01" {
		t.Errorf("ticket type detail mismatch: %+v", tt)
	}
}

func TestEventGRPCHandler_GetTicketTypes_ValidationErrors(t *testing.T) {
	eventSvc := services.NewEventService(&mockEventRepo{}, nil, nil)
	h := handlers.NewEventGRPCHandler(eventSvc)
	ctx := context.Background()

	// 1. Nil request
	_, err := h.GetTicketTypes(ctx, nil)
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on nil request, got %v", err)
	}

	// 2. Empty ticket_type_ids
	_, err = h.GetTicketTypes(ctx, &pbEvent.GetTicketTypesRequest{TicketTypeIds: []string{}})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on empty IDs, got %v", err)
	}

	// 3. Invalid UUID string
	_, err = h.GetTicketTypes(ctx, &pbEvent.GetTicketTypesRequest{TicketTypeIds: []string{"invalid-uuid"}})
	if grpcErrors.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument on invalid UUID, got %v", err)
	}
}
