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

type fakeOrganizerRepo struct {
	items map[uuid.UUID]*models.Organizer
}

func newFakeOrganizerRepo() *fakeOrganizerRepo {
	return &fakeOrganizerRepo{items: make(map[uuid.UUID]*models.Organizer)}
}

func (r *fakeOrganizerRepo) Create(_ context.Context, o *models.Organizer) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	r.items[o.ID] = o
	return nil
}

func (r *fakeOrganizerRepo) FindByID(_ context.Context, id uuid.UUID) (*models.Organizer, error) {
	o, exists := r.items[id]
	if !exists || o.DeletedAt.Valid {
		return nil, dto.ErrOrganizerNotFound
	}
	return o, nil
}

func (r *fakeOrganizerRepo) List(_ context.Context, query dto.OrganizerQuery) ([]*models.Organizer, int64, error) {
	var list []*models.Organizer
	for _, o := range r.items {
		if o.DeletedAt.Valid {
			continue
		}
		if query.Query != "" && !strings.Contains(strings.ToLower(o.OrganizationName), strings.ToLower(query.Query)) {
			continue
		}
		if query.Status != "" && string(o.Status) != query.Status {
			continue
		}
		list = append(list, o)
	}
	return list, int64(len(list)), nil
}

func (r *fakeOrganizerRepo) Update(_ context.Context, o *models.Organizer) error {
	r.items[o.ID] = o
	return nil
}

func (r *fakeOrganizerRepo) Delete(_ context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	o, exists := r.items[id]
	if !exists || o.DeletedAt.Valid {
		return dto.ErrOrganizerNotFound
	}
	o.DeletedAt.Time = time.Now()
	o.DeletedAt.Valid = true
	o.DeletedBy = deletedBy
	return nil
}

func TestOrganizerService_CRUD(t *testing.T) {
	repo := newFakeOrganizerRepo()
	svc := NewOrganizerService(repo)
	ctx := context.Background()

	adminID := uuid.New()
	userCtx := dto.CurrentUserContext{UserID: &adminID}

	// 1. Validation error: empty name
	_, err := svc.Create(ctx, dto.CreateOrganizerRequest{OrganizationName: ""}, userCtx)
	if err != dto.ErrOrganizerNameEmpty {
		t.Fatalf("expected ErrOrganizerNameEmpty, got %v", err)
	}

	// 2. Create success
	desc := "Leading music promoter"
	logo := "https://example.com/logo.png"
	tax := "0109998887"
	addr := "123 Main St"
	created, err := svc.Create(ctx, dto.CreateOrganizerRequest{
		OrganizationName: "LiveConcert Inc",
		Description:      &desc,
		LogoURL:          &logo,
		TaxCode:          &tax,
		Address:          &addr,
		Status:           models.OrganizerStatusActive,
	}, userCtx)
	if err != nil {
		t.Fatalf("failed to create organizer: %v", err)
	}
	if created.OrganizationName != "LiveConcert Inc" || created.CreatedBy == nil || *created.CreatedBy != adminID {
		t.Fatalf("invalid created organizer: %+v", created)
	}

	// 3. GetByID
	fetched, err := svc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get organizer: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("got ID %v, expected %v", fetched.ID, created.ID)
	}

	// 4. List
	list, total, err := svc.List(ctx, dto.OrganizerQuery{BaseQuery: dto.BaseQuery{Query: "Live"}})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 organizer, got total=%d, len=%d", total, len(list))
	}

	// 5. Update
	newDesc := "Updated Description"
	updated, err := svc.Update(ctx, created.ID, dto.UpdateOrganizerRequest{
		Description: &newDesc,
	}, userCtx)
	if err != nil {
		t.Fatalf("failed to update organizer: %v", err)
	}
	if *updated.Description != newDesc {
		t.Fatalf("expected description %s, got %s", newDesc, *updated.Description)
	}

	// 6. Delete
	err = svc.Delete(ctx, created.ID, userCtx)
	if err != nil {
		t.Fatalf("failed to delete organizer: %v", err)
	}

	// 7. Verify not found after delete
	_, err = svc.GetByID(ctx, created.ID)
	if err != dto.ErrOrganizerNotFound {
		t.Fatalf("expected ErrOrganizerNotFound after deletion, got %v", err)
	}
}
