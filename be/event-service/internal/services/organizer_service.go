package services

import (
	"context"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/google/uuid"
)

type OrganizerService struct {
	repo repository.IOrganizerRepository
}

func NewOrganizerService(repo repository.IOrganizerRepository) *OrganizerService {
	return &OrganizerService{repo: repo}
}

func (s *OrganizerService) Create(ctx context.Context, req dto.CreateOrganizerRequest, userCtx dto.CurrentUserContext) (*models.Organizer, error) {
	name := strings.TrimSpace(req.OrganizationName)
	if name == "" {
		return nil, dto.ErrOrganizerNameEmpty
	}

	status := req.Status
	if status == "" {
		status = models.OrganizerStatusActive
	}

	organizer := &models.Organizer{
		OrganizationName: name,
		Description:      req.Description,
		LogoURL:          req.LogoURL,
		TaxCode:          req.TaxCode,
		Address:          req.Address,
		Status:           status,
		CreatedBy:        userCtx.UserID,
		UpdatedBy:        userCtx.UserID,
	}

	if err := s.repo.Create(ctx, organizer); err != nil {
		return nil, err
	}
	return organizer, nil
}

func (s *OrganizerService) GetByID(ctx context.Context, id uuid.UUID) (*models.Organizer, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *OrganizerService) List(ctx context.Context, query dto.OrganizerQuery) ([]*models.Organizer, int64, error) {
	return s.repo.List(ctx, query)
}

func (s *OrganizerService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateOrganizerRequest, userCtx dto.CurrentUserContext) (*models.Organizer, error) {
	organizer, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.OrganizationName != nil {
		name := strings.TrimSpace(*req.OrganizationName)
		if name == "" {
			return nil, dto.ErrOrganizerNameEmpty
		}
		organizer.OrganizationName = name
	}
	if req.Description != nil {
		organizer.Description = req.Description
	}
	if req.LogoURL != nil {
		organizer.LogoURL = req.LogoURL
	}
	if req.TaxCode != nil {
		organizer.TaxCode = req.TaxCode
	}
	if req.Address != nil {
		organizer.Address = req.Address
	}
	if req.Status != nil {
		organizer.Status = *req.Status
	}

	if userCtx.UserID != nil {
		organizer.UpdatedBy = userCtx.UserID
	}

	if err := s.repo.Update(ctx, organizer); err != nil {
		return nil, err
	}
	return organizer, nil
}

func (s *OrganizerService) Delete(ctx context.Context, id uuid.UUID, userCtx dto.CurrentUserContext) error {
	return s.repo.Delete(ctx, id, userCtx.UserID)
}
