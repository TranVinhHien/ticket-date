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

type fakeVenueRepo struct {
	items map[uuid.UUID]*models.Venue
}

func newFakeVenueRepo() *fakeVenueRepo {
	return &fakeVenueRepo{items: make(map[uuid.UUID]*models.Venue)}
}

func (r *fakeVenueRepo) Create(_ context.Context, v *models.Venue) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	r.items[v.ID] = v
	return nil
}

func (r *fakeVenueRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Venue, error) {
	v, exists := r.items[id]
	if !exists || v.DeletedAt.Valid {
		return nil, dto.ErrVenueNotFound
	}
	return v, nil
}

func (r *fakeVenueRepo) List(_ context.Context, query dto.VenueQuery) ([]*models.Venue, int64, error) {
	var list []*models.Venue
	for _, v := range r.items {
		if v.DeletedAt.Valid {
			continue
		}
		if query.OrganizerID != "" && v.OrganizerID.String() != query.OrganizerID {
			continue
		}
		if query.City != "" && (v.City == nil || !strings.Contains(strings.ToLower(*v.City), strings.ToLower(query.City))) {
			continue
		}
		if query.Query != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(query.Query)) {
			continue
		}
		list = append(list, v)
	}
	return list, int64(len(list)), nil
}

func (r *fakeVenueRepo) Update(_ context.Context, v *models.Venue) error {
	r.items[v.ID] = v
	return nil
}

func (r *fakeVenueRepo) Delete(_ context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	v, exists := r.items[id]
	if !exists || v.DeletedAt.Valid {
		return dto.ErrVenueNotFound
	}
	v.DeletedAt.Time = time.Now()
	v.DeletedAt.Valid = true
	v.DeletedBy = deletedBy
	return nil
}

func TestVenueService_CRUD(t *testing.T) {
	venueRepo := newFakeVenueRepo()
	orgRepo := newFakeOrganizerRepo()
	svc := NewVenueService(venueRepo, orgRepo)
	ctx := context.Background()

	adminID := uuid.New()
	userCtx := dto.CurrentUserContext{UserID: &adminID}

	// Create an organizer first
	org := &models.Organizer{
		ID:               uuid.New(),
		OrganizationName: "LiveNation",
		Status:           models.OrganizerStatusActive,
	}
	_ = orgRepo.Create(ctx, org)

	// 1. Validation error: non-existent organizer
	_, err := svc.Create(ctx, dto.CreateVenueRequest{
		OrganizerID: uuid.New(),
		Name:        "Arena",
		Address:     "123 St",
		Capacity:    1000,
	}, userCtx)
	if err != dto.ErrOrganizerNotFound {
		t.Fatalf("expected ErrOrganizerNotFound, got %v", err)
	}

	// 2. Validation error: capacity <= 0
	_, err = svc.Create(ctx, dto.CreateVenueRequest{
		OrganizerID: org.ID,
		Name:        "Arena",
		Address:     "123 St",
		Capacity:    0,
	}, userCtx)
	if err != dto.ErrInvalidCapacity {
		t.Fatalf("expected ErrInvalidCapacity, got %v", err)
	}

	// 3. Validation error: invalid coordinates
	badLat := 95.0
	_, err = svc.Create(ctx, dto.CreateVenueRequest{
		OrganizerID: org.ID,
		Name:        "Arena",
		Address:     "123 St",
		Capacity:    5000,
		Latitude:    &badLat,
	}, userCtx)
	if err != dto.ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}

	// 4. Create success
	city := "Ho Chi Minh City"
	lat := 10.7769
	lng := 106.7009
	created, err := svc.Create(ctx, dto.CreateVenueRequest{
		OrganizerID: org.ID,
		Name:        "Saigon Dome",
		Address:     "Nguyen Thi Minh Khai, District 1",
		City:        &city,
		Capacity:    8000,
		Latitude:    &lat,
		Longitude:   &lng,
		Status:      models.VenueStatusActive,
	}, userCtx)
	if err != nil {
		t.Fatalf("failed to create venue: %v", err)
	}
	if created.Name != "Saigon Dome" || created.Capacity != 8000 {
		t.Fatalf("invalid venue: %+v", created)
	}

	// 5. GetByID
	fetched, err := svc.GetByID(ctx, created.ID)
	if err != nil || fetched.ID != created.ID {
		t.Fatalf("GetByID failed: %v", err)
	}

	// 6. List with city filter
	list, total, err := svc.List(ctx, dto.VenueQuery{City: "Ho Chi Minh"})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List failed: total=%d, len=%d, err=%v", total, len(list), err)
	}

	// 7. Update
	newCap := 9500
	updated, err := svc.Update(ctx, created.ID, dto.UpdateVenueRequest{Capacity: &newCap}, userCtx)
	if err != nil || updated.Capacity != 9500 {
		t.Fatalf("Update failed: %v", err)
	}

	// 8. Delete
	if err := svc.Delete(ctx, created.ID, userCtx); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 9. Verify 404
	if _, err := svc.GetByID(ctx, created.ID); err != dto.ErrVenueNotFound {
		t.Fatalf("expected ErrVenueNotFound, got %v", err)
	}
}
