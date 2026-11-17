package repository

import (
	"context"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IOrganizerRepository interface {
	Create(ctx context.Context, organizer *models.Organizer) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Organizer, error)
	List(ctx context.Context, query dto.OrganizerQuery) ([]*models.Organizer, int64, error)
	Update(ctx context.Context, organizer *models.Organizer) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error
}

type OrganizerRepository struct {
	db *gorm.DB
}

func NewOrganizerRepository(db *gorm.DB) IOrganizerRepository {
	return &OrganizerRepository{db: db}
}

func (r *OrganizerRepository) Create(ctx context.Context, organizer *models.Organizer) error {
	return r.db.WithContext(ctx).Create(organizer).Error
}

func (r *OrganizerRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Organizer, error) {
	var organizer models.Organizer
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&organizer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dto.ErrOrganizerNotFound
		}
		return nil, err
	}
	return &organizer, nil
}

func (r *OrganizerRepository) List(ctx context.Context, query dto.OrganizerQuery) ([]*models.Organizer, int64, error) {
	var organizers []*models.Organizer
	var total int64

	db := r.db.WithContext(ctx).Model(&models.Organizer{})

	if query.Query != "" {
		db = db.Where("organization_name ILIKE ?", "%"+query.Query+"%")
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

	if err := db.Order(orderClause).Offset(offset).Limit(limit).Find(&organizers).Error; err != nil {
		return nil, 0, err
	}

	return organizers, total, nil
}

func (r *OrganizerRepository) Update(ctx context.Context, organizer *models.Organizer) error {
	return r.db.WithContext(ctx).Save(organizer).Error
}

func (r *OrganizerRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy *uuid.UUID) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"deleted_at": now,
	}
	if deletedBy != nil {
		updates["deleted_by"] = deletedBy
	}
	res := r.db.WithContext(ctx).Model(&models.Organizer{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return dto.ErrOrganizerNotFound
	}
	return nil
}
