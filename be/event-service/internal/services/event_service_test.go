package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

type fakeEventRepo struct {
	events      map[uuid.UUID]*models.Event
	ticketTypes map[uuid.UUID]*models.TicketType
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{
		events:      make(map[uuid.UUID]*models.Event),
		ticketTypes: make(map[uuid.UUID]*models.TicketType),
	}
}

func (r *fakeEventRepo) Create(_ context.Context, e *models.Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	r.events[e.ID] = e

	for i := range e.TicketTypes {
		if e.TicketTypes[i].ID == uuid.Nil {
			e.TicketTypes[i].ID = uuid.New()
		}
		e.TicketTypes[i].EventID = e.ID
		r.ticketTypes[e.TicketTypes[i].ID] = &e.TicketTypes[i]
	}

	return nil
}

func (r *fakeEventRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Event, error) {
	e, exists := r.events[id]
	if !exists || e.DeletedAt.Valid {
		return nil, dto.ErrEventNotFound
	}
	// Attach active ticket types
	var tts []models.TicketType
	for _, tt := range r.ticketTypes {
		if tt.EventID == id && !tt.DeletedAt.Valid {
			tts = append(tts, *tt)
		}
	}
	eventCopy := *e
	eventCopy.TicketTypes = tts
	return &eventCopy, nil
}

func (r *fakeEventRepo) FindBySlug(_ context.Context, slug string) (*models.Event, error) {
	for _, e := range r.events {
		if !e.DeletedAt.Valid && e.Slug == slug {
			var tts []models.TicketType
			for _, tt := range r.ticketTypes {
				if tt.EventID == e.ID && !tt.DeletedAt.Valid {
					tts = append(tts, *tt)
				}
			}
			eventCopy := *e
			eventCopy.TicketTypes = tts
			return &eventCopy, nil
		}
	}
	return nil, dto.ErrEventNotFound
}

func (r *fakeEventRepo) FindBySKU(_ context.Context, sku string) (*models.TicketType, error) {
	for _, tt := range r.ticketTypes {
		if !tt.DeletedAt.Valid && tt.SKU == sku {
			return tt, nil
		}
	}
	return nil, dto.ErrTicketTypeNotFound
}

func (r *fakeEventRepo) List(_ context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	var list []*models.Event
	for _, e := range r.events {
		if e.DeletedAt.Valid {
			continue
		}
		if query.OrganizerID != "" && e.OrganizerID.String() != query.OrganizerID {
			continue
		}
		if query.VenueID != "" && e.VenueID.String() != query.VenueID {
			continue
		}
		if query.Query != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(query.Query)) {
			continue
		}
		if query.Status != "" && string(e.Status) != query.Status {
			continue
		}

		var tts []models.TicketType
		for _, tt := range r.ticketTypes {
			if tt.EventID == e.ID && !tt.DeletedAt.Valid {
				tts = append(tts, *tt)
			}
		}
		eventCopy := *e
		eventCopy.TicketTypes = tts
		list = append(list, &eventCopy)
	}
	return list, int64(len(list)), nil
}

func (r *fakeEventRepo) Update(_ context.Context, e *models.Event) error {
	r.events[e.ID] = e

	if e.TicketTypes != nil {
		activeIDs := make(map[uuid.UUID]bool)
		for _, tt := range e.TicketTypes {
			if tt.ID == uuid.Nil {
				tt.ID = uuid.New()
			}
			activeIDs[tt.ID] = true
			copyTT := tt
			r.ticketTypes[tt.ID] = &copyTT
		}

		// Soft-delete any ticket types for this event not in activeIDs
		for id, tt := range r.ticketTypes {
			if tt.EventID == e.ID && !activeIDs[id] && !tt.DeletedAt.Valid {
				tt.DeletedAt.Time = time.Now()
				tt.DeletedAt.Valid = true
			}
		}
	}

	return nil
}

func (r *fakeEventRepo) Delete(_ context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	e, exists := r.events[id]
	if !exists || e.DeletedAt.Valid {
		return dto.ErrEventNotFound
	}
	now := time.Now()
	e.DeletedAt.Time = now
	e.DeletedAt.Valid = true
	e.DeletedBy = deletedBy

	for _, tt := range r.ticketTypes {
		if tt.EventID == id {
			tt.DeletedAt.Time = now
			tt.DeletedAt.Valid = true
			tt.DeletedBy = deletedBy
		}
	}

	return nil
}

func TestEventService_CRUD_With_TicketTypes(t *testing.T) {
	eventRepo := newFakeEventRepo()
	orgRepo := newFakeOrganizerRepo()
	venueRepo := newFakeVenueRepo()
	svc := NewEventService(eventRepo, orgRepo, venueRepo)
	ctx := context.Background()

	adminID := uuid.New()
	userCtx := dto.CurrentUserContext{UserID: &adminID}

	// Create prerequisites
	org := &models.Organizer{ID: uuid.New(), OrganizationName: "Fest Org", Status: models.OrganizerStatusActive}
	_ = orgRepo.Create(ctx, org)
	venue := &models.Venue{ID: uuid.New(), OrganizerID: org.ID, Name: "Fest Park", Address: "Park Ave", Capacity: 5000, Status: models.VenueStatusActive}
	_ = venueRepo.Create(ctx, venue)

	startAt := time.Now().Add(24 * time.Hour)
	endAt := startAt.Add(4 * time.Hour)
	saleStartAt := time.Now()
	saleEndAt := startAt

	// 1. Validation error: Ticket type has negative price
	_, err := svc.Create(ctx, dto.CreateEventRequest{
		OrganizerID: org.ID,
		VenueID:     venue.ID,
		Name:        "Rock Fest",
		StartAt:     startAt,
		EndAt:       endAt,
		TicketTypes: []dto.CreateTicketTypeInput{
			{
				SKU:         "SKU-VIP",
				Name:        "VIP Standing",
				Price:       -500,
				SaleStartAt: saleStartAt,
				SaleEndAt:   saleEndAt,
			},
		},
	}, userCtx)
	if err != dto.ErrInvalidPrice {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}

	// 2. Validation error: Ticket type sale dates invalid
	_, err = svc.Create(ctx, dto.CreateEventRequest{
		OrganizerID: org.ID,
		VenueID:     venue.ID,
		Name:        "Rock Fest",
		StartAt:     startAt,
		EndAt:       endAt,
		TicketTypes: []dto.CreateTicketTypeInput{
			{
				SKU:         "SKU-VIP",
				Name:        "VIP Standing",
				Price:       100000,
				SaleStartAt: saleEndAt,
				SaleEndAt:   saleStartAt,
			},
		},
	}, userCtx)
	if err != dto.ErrInvalidSalePeriod {
		t.Fatalf("expected ErrInvalidSalePeriod, got %v", err)
	}

	// 3. Validation error: Duplicate SKU within payload
	_, err = svc.Create(ctx, dto.CreateEventRequest{
		OrganizerID: org.ID,
		VenueID:     venue.ID,
		Name:        "Rock Fest",
		StartAt:     startAt,
		EndAt:       endAt,
		TicketTypes: []dto.CreateTicketTypeInput{
			{
				SKU:         "SKU-VIP",
				Name:        "VIP 1",
				Price:       100000,
				SaleStartAt: saleStartAt,
				SaleEndAt:   saleEndAt,
			},
			{
				SKU:         "SKU-VIP",
				Name:        "VIP 2",
				Price:       200000,
				SaleStartAt: saleStartAt,
				SaleEndAt:   saleEndAt,
			},
		},
	}, userCtx)
	if err != dto.ErrSKUAlreadyExists {
		t.Fatalf("expected ErrSKUAlreadyExists, got %v", err)
	}

	// 4. Create Success with 2 Ticket Types
	created, err := svc.Create(ctx, dto.CreateEventRequest{
		OrganizerID: org.ID,
		VenueID:     venue.ID,
		Name:        "Summer Rock Fest 2026",
		StartAt:     startAt,
		EndAt:       endAt,
		Status:      models.EventStatusPublished,
		TicketTypes: []dto.CreateTicketTypeInput{
			{
				SKU:         "RF26-VIP",
				Name:        "VIP Standing",
				Price:       1500000,
				Currency:    "VND",
				MaxPerOrder: 4,
				SaleStartAt: saleStartAt,
				SaleEndAt:   saleEndAt,
				Status:      models.TicketTypeStatusActive,
			},
			{
				SKU:         "RF26-GA",
				Name:        "General Admission",
				Price:       500000,
				Currency:    "VND",
				MaxPerOrder: 6,
				SaleStartAt: saleStartAt,
				SaleEndAt:   saleEndAt,
				Status:      models.TicketTypeStatusActive,
			},
		},
	}, userCtx)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}
	if created.Slug != "summer-rock-fest-2026" {
		t.Fatalf("unexpected slug: %s", created.Slug)
	}
	if len(created.TicketTypes) != 2 {
		t.Fatalf("expected 2 ticket types, got %d", len(created.TicketTypes))
	}

	// 5. GetByID and GetBySlug verifying ticket types
	fetchedByID, err := svc.GetByID(ctx, created.ID)
	if err != nil || len(fetchedByID.TicketTypes) != 2 {
		t.Fatalf("GetByID failed: %v, ticket types len: %d", err, len(fetchedByID.TicketTypes))
	}
	fetchedBySlug, err := svc.GetBySlug(ctx, "summer-rock-fest-2026")
	if err != nil || len(fetchedBySlug.TicketTypes) != 2 {
		t.Fatalf("GetBySlug failed: %v", err)
	}

	// 6. Update Event: modify VIP price, delete GA ticket type, add VVIP tier
	var vipID, gaID uuid.UUID
	for _, tt := range created.TicketTypes {
		if tt.SKU == "RF26-VIP" {
			vipID = tt.ID
		} else if tt.SKU == "RF26-GA" {
			gaID = tt.ID
		}
	}
	newVIPPrice := int64(1800000)
	newVVIPSKU := "RF26-VVIP"
	newVVIPName := "VVIP Backstage"
	newVVIPPrice := int64(3000000)

	updated, err := svc.Update(ctx, created.ID, dto.UpdateEventRequest{
		TicketTypes: []dto.UpdateTicketTypeInput{
			{
				ID:    &vipID,
				Price: &newVIPPrice,
			},
			{
				ID:        &gaID,
				IsDeleted: true,
			},
			{
				SKU:   &newVVIPSKU,
				Name:  &newVVIPName,
				Price: &newVVIPPrice,
			},
		},
	}, userCtx)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Re-verify fetched event ticket types: GA should be gone, VIP price updated, VVIP added (total 2 active)
	if len(updated.TicketTypes) != 2 {
		t.Fatalf("expected 2 active ticket types, got %d", len(updated.TicketTypes))
	}
	var hasVIP, hasVVIP bool
	for _, tt := range updated.TicketTypes {
		if tt.SKU == "RF26-VIP" && tt.Price == 1800000 {
			hasVIP = true
		}
		if tt.SKU == "RF26-VVIP" && tt.Price == 3000000 {
			hasVVIP = true
		}
	}
	if !hasVIP || !hasVVIP {
		t.Fatalf("ticket types update mismatch: %+v", updated.TicketTypes)
	}

	// 7. Delete Event: should cascade delete event and ticket types
	if err := svc.Delete(ctx, created.ID, userCtx); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := svc.GetByID(ctx, created.ID); err != dto.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
}
