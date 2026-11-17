package services

import (
	"context"
	"strings"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/google/uuid"
)

type VenueService struct {
	venueRepo     repository.IVenueRepository
	organizerRepo repository.IOrganizerRepository
}

func NewVenueService(venueRepo repository.IVenueRepository, organizerRepo repository.IOrganizerRepository) *VenueService {
	return &VenueService{
		venueRepo:     venueRepo,
		organizerRepo: organizerRepo,
	}
}

func (s *VenueService) Create(ctx context.Context, req dto.CreateVenueRequest, userCtx dto.CurrentUserContext) (*models.Venue, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, dto.ErrVenueNameEmpty
	}
	address := strings.TrimSpace(req.Address)
	if address == "" {
		return nil, dto.ErrVenueAddressEmpty
	}
	if req.Capacity <= 0 {
		return nil, dto.ErrInvalidCapacity
	}

	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		return nil, dto.ErrInvalidCoordinates
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		return nil, dto.ErrInvalidCoordinates
	}

	if _, err := s.organizerRepo.FindByID(ctx, req.OrganizerID); err != nil {
		return nil, dto.ErrOrganizerNotFound
	}

	country := req.Country
	if strings.TrimSpace(country) == "" {
		country = "Vietnam"
	}
	status := req.Status
	if status == "" {
		status = models.VenueStatusActive
	}

	venue := &models.Venue{
		OrganizerID: req.OrganizerID,
		Name:        name,
		Address:     address,
		City:        req.City,
		Country:     country,
		MapURL:      req.MapURL,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Capacity:    req.Capacity,
		Description: req.Description,
		Status:      status,
		CreatedBy:   userCtx.UserID,
		UpdatedBy:   userCtx.UserID,
	}

	if err := s.venueRepo.Create(ctx, venue); err != nil {
		return nil, err
	}
	return venue, nil
}

func (s *VenueService) GetByID(ctx context.Context, id uuid.UUID) (*models.Venue, error) {
	return s.venueRepo.FindByID(ctx, id)
}

func (s *VenueService) List(ctx context.Context, query dto.VenueQuery) ([]*models.Venue, int64, error) {
	return s.venueRepo.List(ctx, query)
}

func (s *VenueService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateVenueRequest, userCtx dto.CurrentUserContext) (*models.Venue, error) {
	venue, err := s.venueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.OrganizerID != nil {
		if _, err := s.organizerRepo.FindByID(ctx, *req.OrganizerID); err != nil {
			return nil, dto.ErrOrganizerNotFound
		}
		venue.OrganizerID = *req.OrganizerID
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, dto.ErrVenueNameEmpty
		}
		venue.Name = name
	}
	if req.Address != nil {
		address := strings.TrimSpace(*req.Address)
		if address == "" {
			return nil, dto.ErrVenueAddressEmpty
		}
		venue.Address = address
	}
	if req.City != nil {
		venue.City = req.City
	}
	if req.Country != nil && strings.TrimSpace(*req.Country) != "" {
		venue.Country = *req.Country
	}
	if req.MapURL != nil {
		venue.MapURL = req.MapURL
	}
	if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 {
			return nil, dto.ErrInvalidCoordinates
		}
		venue.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		if *req.Longitude < -180 || *req.Longitude > 180 {
			return nil, dto.ErrInvalidCoordinates
		}
		venue.Longitude = req.Longitude
	}
	if req.Capacity != nil {
		if *req.Capacity <= 0 {
			return nil, dto.ErrInvalidCapacity
		}
		venue.Capacity = *req.Capacity
	}
	if req.Description != nil {
		venue.Description = req.Description
	}
	if req.Status != nil {
		venue.Status = *req.Status
	}

	if userCtx.UserID != nil {
		venue.UpdatedBy = userCtx.UserID
	}

	if err := s.venueRepo.Update(ctx, venue); err != nil {
		return nil, err
	}
	return venue, nil
}

func (s *VenueService) Delete(ctx context.Context, id uuid.UUID, userCtx dto.CurrentUserContext) error {
	return s.venueRepo.Delete(ctx, id, userCtx.UserID)
}
