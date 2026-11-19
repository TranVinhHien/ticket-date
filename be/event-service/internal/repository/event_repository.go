package repository

import (
	"context"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IEventRepository interface {
	Create(ctx context.Context, event *models.Event) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Event, error)
	FindBySlug(ctx context.Context, slug string) (*models.Event, error)
	FindBySKU(ctx context.Context, sku string) (*models.TicketType, error)
	FindTicketTypesByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error)
	List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error)
	Update(ctx context.Context, event *models.Event) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error
}

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) IEventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Set audit fields on ticket types before create
		for i := range event.TicketTypes {
			if event.CreatedBy != nil {
				event.TicketTypes[i].CreatedBy = event.CreatedBy
				event.TicketTypes[i].UpdatedBy = event.CreatedBy
			}
		}

		// GORM automatically inserts the event and its associated TicketTypes in a single operation
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).
		Preload("Organizer").
		Preload("Venue").
		Preload("TicketTypes", "deleted_at IS NULL").
		Where("id = ?", id).
		First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dto.ErrEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

func (r *EventRepository) FindBySlug(ctx context.Context, slug string) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).
		Preload("Organizer").
		Preload("Venue").
		Preload("TicketTypes", "deleted_at IS NULL").
		Where("slug = ?", slug).
		First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dto.ErrEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

func (r *EventRepository) FindBySKU(ctx context.Context, sku string) (*models.TicketType, error) {
	var ticketType models.TicketType
	if err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&ticketType).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dto.ErrTicketTypeNotFound
		}
		return nil, err
	}
	return &ticketType, nil
}

func (r *EventRepository) FindTicketTypesByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error) {
	var ticketTypes []*models.TicketType
	if len(ids) == 0 {
		return ticketTypes, nil
	}
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&ticketTypes).Error; err != nil {
		return nil, err
	}
	return ticketTypes, nil
}

func (r *EventRepository) List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	var events []*models.Event
	var total int64

	db := r.db.WithContext(ctx).Model(&models.Event{})

	if query.OrganizerID != "" {
		if orgID, err := uuid.Parse(query.OrganizerID); err == nil {
			db = db.Where("organizer_id = ?", orgID)
		}
	}
	if query.VenueID != "" {
		if venueID, err := uuid.Parse(query.VenueID); err == nil {
			db = db.Where("venue_id = ?", venueID)
		}
	}
	if query.Query != "" {
		db = db.Where("name ILIKE ? OR description ILIKE ?", "%"+query.Query+"%", "%"+query.Query+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := query.GetOffset()
	limit := query.GetLimit()
	orderClause := "start_at ASC"
	if query.Sort != "" {
		orderClause = query.Sort + " " + query.GetOrder()
	}

	if err := db.
		Preload("Organizer").
		Preload("Venue").
		Preload("TicketTypes", "deleted_at IS NULL").
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *EventRepository) Update(ctx context.Context, event *models.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Update scalar event fields (omit associations so GORM doesn't perform individual operations)
		if err := tx.Omit("TicketTypes", "Organizer", "Venue").Save(event).Error; err != nil {
			return err
		}

		if event.TicketTypes == nil {
			return nil
		}

		now := time.Now().UTC()

		// 2. Separate active ticket types and collect their IDs
		var activeTicketTypes []models.TicketType
		var activeIDs []uuid.UUID

		for i := range event.TicketTypes {
			tt := &event.TicketTypes[i]
			if tt.DeletedAt.Valid {
				continue
			}
			if tt.ID == uuid.Nil {
				tt.ID = uuid.New()
			}
			tt.EventID = event.ID
			if event.UpdatedBy != nil {
				tt.UpdatedBy = event.UpdatedBy
				if tt.CreatedBy == nil {
					tt.CreatedBy = event.UpdatedBy
				}
			}
			activeTicketTypes = append(activeTicketTypes, *tt)
			activeIDs = append(activeIDs, tt.ID)
		}

		// 3. Batch soft-delete any ticket types belonging to this event that are NOT in activeIDs (single query)
		delQuery := tx.Model(&models.TicketType{}).
			Where("event_id = ? AND deleted_at IS NULL", event.ID)
		if len(activeIDs) > 0 {
			delQuery = delQuery.Where("id NOT IN ?", activeIDs)
		}
		delUpdates := map[string]interface{}{
			"deleted_at": now,
		}
		if event.UpdatedBy != nil {
			delUpdates["deleted_by"] = event.UpdatedBy
		}
		if err := delQuery.Updates(delUpdates).Error; err != nil {
			return err
		}

		// 4. Batch upsert (INSERT ... ON CONFLICT DO UPDATE) all active ticket types in a single query
		if len(activeTicketTypes) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"sku",
					"name",
					"description",
					"price",
					"currency",
					"max_per_order",
					"sale_start_at",
					"sale_end_at",
					"attributes",
					"status",
					"updated_by",
					"updated_at",
				}),
			}).Create(&activeTicketTypes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *EventRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		updates := map[string]interface{}{
			"deleted_at": now,
		}
		if deletedBy != nil {
			updates["deleted_by"] = deletedBy
		}

		res := tx.Model(&models.Event{}).Where("id = ?", id).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return dto.ErrEventNotFound
		}

		// Soft-delete all associated ticket types in batch
		if err := tx.Model(&models.TicketType{}).
			Where("event_id = ? AND deleted_at IS NULL", id).
			Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})
}
