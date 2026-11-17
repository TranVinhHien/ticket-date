package repository

import (
	"context"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IVenueRepository interface {
	Create(ctx context.Context, venue *models.Venue) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Venue, error)
	List(ctx context.Context, query dto.VenueQuery) ([]*models.Venue, int64, error)
	Update(ctx context.Context, venue *models.Venue) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error
}

type VenueRepository struct {
	db *gorm.DB
}

func NewVenueRepository(db *gorm.DB) IVenueRepository {
	return &VenueRepository{db: db}
}

func (r *VenueRepository) Create(ctx context.Context, venue *models.Venue) error {
	return r.db.WithContext(ctx).Create(venue).Error
}

func (r *VenueRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Venue, error) {
	var venue models.Venue
	if err := r.db.WithContext(ctx).Preload("Organizer").Where("id = ?", id).First(&venue).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dto.ErrVenueNotFound
		}
		return nil, err
	}
	return &venue, nil
}

func (r *VenueRepository) List(ctx context.Context, query dto.VenueQuery) ([]*models.Venue, int64, error) {
	var venues []*models.Venue
	var total int64

	db := r.db.WithContext(ctx).Model(&models.Venue{})

	if query.OrganizerID != "" {
		if orgID, err := uuid.Parse(query.OrganizerID); err == nil {
			db = db.Where("organizer_id = ?", orgID)
		}
	}
	if query.City != "" {
		db = db.Where("city ILIKE ?", "%"+query.City+"%")
	}
	if query.Query != "" {
		db = db.Where("name ILIKE ? OR address ILIKE ?", "%"+query.Query+"%", "%"+query.Query+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := query.GetOffset()
	limit := query.GetLimit()
	orderClause := "created_at DESC"
	if query.Sort != "" {
		orderClause = query.Sort + " " + query.GetOrder()
	}

	if err := db.Preload("Organizer").Order(orderClause).Offset(offset).Limit(limit).Find(&venues).Error; err != nil {
		return nil, 0, err
	}

	return venues, total, nil
}

func (r *VenueRepository) Update(ctx context.Context, venue *models.Venue) error {
	return r.db.WithContext(ctx).Save(venue).Error
}

func (r *VenueRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"deleted_at": now,
	}
	if deletedBy != nil {
		updates["deleted_by"] = deletedBy
	}
	res := r.db.WithContext(ctx).Model(&models.Venue{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return dto.ErrVenueNotFound
	}
	return nil
}
