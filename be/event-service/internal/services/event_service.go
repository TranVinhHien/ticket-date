package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/google/uuid"
)

type EventService struct {
	eventRepo     repository.IEventRepository
	organizerRepo repository.IOrganizerRepository
	venueRepo     repository.IVenueRepository
}

func NewEventService(
	eventRepo repository.IEventRepository,
	organizerRepo repository.IOrganizerRepository,
	venueRepo repository.IVenueRepository,
) *EventService {
	return &EventService{
		eventRepo:     eventRepo,
		organizerRepo: organizerRepo,
		venueRepo:     venueRepo,
	}
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
	}
	res := strings.Trim(b.String(), "-")
	if res == "" {
		res = fmt.Sprintf("event-%d", time.Now().Unix())
	}
	return res
}

func (s *EventService) Create(ctx context.Context, req dto.CreateEventRequest, userCtx dto.CurrentUserContext) (*models.Event, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, dto.ErrEventNameEmpty
	}

	if !req.EndAt.After(req.StartAt) {
		return nil, dto.ErrInvalidPeriod
	}

	if _, err := s.organizerRepo.FindByID(ctx, req.OrganizerID); err != nil {
		return nil, dto.ErrOrganizerNotFound
	}

	if _, err := s.venueRepo.FindByID(ctx, req.VenueID); err != nil {
		return nil, dto.ErrVenueNotFound
	}

	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		slug = Slugify(name)
	} else {
		slug = Slugify(slug)
	}

	// Check slug uniqueness
	existing, err := s.eventRepo.FindBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, dto.ErrSlugAlreadyExists
	}

	status := req.Status
	if status == "" {
		status = models.EventStatusDraft
	}

	// Validate and transform ticket types
	seenSKUs := make(map[string]bool)
	var ticketTypes []models.TicketType

	for _, tt := range req.TicketTypes {
		ttName := strings.TrimSpace(tt.Name)
		if ttName == "" {
			return nil, dto.ErrTicketTypeNameEmpty
		}

		sku := strings.TrimSpace(tt.SKU)
		if sku == "" {
			return nil, dto.ErrTicketTypeNameEmpty
		}

		if seenSKUs[sku] {
			return nil, dto.ErrSKUAlreadyExists
		}
		seenSKUs[sku] = true

		if tt.Price < 0 {
			return nil, dto.ErrInvalidPrice
		}

		maxPerOrder := tt.MaxPerOrder
		if maxPerOrder <= 0 {
			maxPerOrder = 4
		}

		if !tt.SaleEndAt.After(tt.SaleStartAt) {
			return nil, dto.ErrInvalidSalePeriod
		}

		// Check SKU uniqueness in DB
		existingTT, err := s.eventRepo.FindBySKU(ctx, sku)
		if err == nil && existingTT != nil {
			return nil, dto.ErrSKUAlreadyExists
		}

		currency := strings.TrimSpace(tt.Currency)
		if currency == "" {
			currency = "VND"
		}

		ttStatus := tt.Status
		if ttStatus == "" {
			ttStatus = models.TicketTypeStatusDraft
		}

		attrs := tt.Attributes
		if attrs == nil {
			attrs = models.JSONMap{}
		}

		ticketTypes = append(ticketTypes, models.TicketType{
			SKU:         sku,
			Name:        ttName,
			Description: tt.Description,
			Price:       tt.Price,
			Currency:    currency,
			MaxPerOrder: maxPerOrder,
			SaleStartAt: tt.SaleStartAt,
			SaleEndAt:   tt.SaleEndAt,
			Attributes:  attrs,
			Status:      ttStatus,
		})
	}

	event := &models.Event{
		OrganizerID: req.OrganizerID,
		VenueID:     req.VenueID,
		Name:        name,
		Slug:        slug,
		Description: req.Description,
		PosterURL:   req.PosterURL,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		Status:      status,
		CreatedBy:   userCtx.UserID,
		UpdatedBy:   userCtx.UserID,
		TicketTypes: ticketTypes,
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	return s.eventRepo.FindByID(ctx, event.ID)
}

func (s *EventService) GetByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	return s.eventRepo.FindByID(ctx, id)
}

func (s *EventService) GetBySlug(ctx context.Context, slug string) (*models.Event, error) {
	return s.eventRepo.FindBySlug(ctx, slug)
}

func (s *EventService) List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	return s.eventRepo.List(ctx, query)
}

func (s *EventService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateEventRequest, userCtx dto.CurrentUserContext) (*models.Event, error) {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.OrganizerID != nil {
		if _, err := s.organizerRepo.FindByID(ctx, *req.OrganizerID); err != nil {
			return nil, dto.ErrOrganizerNotFound
		}
		event.OrganizerID = *req.OrganizerID
	}

	if req.VenueID != nil {
		if _, err := s.venueRepo.FindByID(ctx, *req.VenueID); err != nil {
			return nil, dto.ErrVenueNotFound
		}
		event.VenueID = *req.VenueID
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, dto.ErrEventNameEmpty
		}
		event.Name = name
	}

	if req.Slug != nil {
		newSlug := Slugify(*req.Slug)
		if newSlug != event.Slug {
			existing, err := s.eventRepo.FindBySlug(ctx, newSlug)
			if err == nil && existing != nil && existing.ID != event.ID {
				return nil, dto.ErrSlugAlreadyExists
			}
			event.Slug = newSlug
		}
	}

	startAt := event.StartAt
	if req.StartAt != nil {
		startAt = *req.StartAt
	}
	endAt := event.EndAt
	if req.EndAt != nil {
		endAt = *req.EndAt
	}
	if !endAt.After(startAt) {
		return nil, dto.ErrInvalidPeriod
	}
	event.StartAt = startAt
	event.EndAt = endAt

	if req.Description != nil {
		event.Description = req.Description
	}
	if req.PosterURL != nil {
		event.PosterURL = req.PosterURL
	}
	if req.Status != nil {
		event.Status = *req.Status
	}

	if userCtx.UserID != nil {
		event.UpdatedBy = userCtx.UserID
	}

	if req.TicketTypes != nil {
		existingMap := make(map[uuid.UUID]*models.TicketType)
		for i := range event.TicketTypes {
			existingMap[event.TicketTypes[i].ID] = &event.TicketTypes[i]
		}

		touchedIDs := make(map[uuid.UUID]bool)
		seenSKUs := make(map[string]bool)

		// Identify touched IDs
		for _, tt := range req.TicketTypes {
			if tt.ID != nil && *tt.ID != uuid.Nil {
				touchedIDs[*tt.ID] = true
			}
		}

		// Keep untouched existing ticket types and register their SKUs
		var updatedTicketTypes []models.TicketType
		for _, existing := range event.TicketTypes {
			if !touchedIDs[existing.ID] {
				updatedTicketTypes = append(updatedTicketTypes, existing)
				seenSKUs[existing.SKU] = true
			}
		}

		// 1. Process modified existing ticket types
		for _, tt := range req.TicketTypes {
			if tt.ID != nil && *tt.ID != uuid.Nil {
				existing, ok := existingMap[*tt.ID]
				if !ok {
					return nil, dto.ErrTicketTypeNotFound
				}

				if tt.IsDeleted {
					continue
				}

				if tt.Name != nil {
					name := strings.TrimSpace(*tt.Name)
					if name == "" {
						return nil, dto.ErrTicketTypeNameEmpty
					}
					existing.Name = name
				}

				if tt.SKU != nil {
					sku := strings.TrimSpace(*tt.SKU)
					if sku == "" {
						return nil, dto.ErrTicketTypeNameEmpty
					}
					if seenSKUs[sku] {
						return nil, dto.ErrSKUAlreadyExists
					}
					if sku != existing.SKU {
						existingTT, err := s.eventRepo.FindBySKU(ctx, sku)
						if err == nil && existingTT != nil && existingTT.ID != existing.ID {
							return nil, dto.ErrSKUAlreadyExists
						}
					}
					existing.SKU = sku
					seenSKUs[sku] = true
				} else {
					if seenSKUs[existing.SKU] {
						return nil, dto.ErrSKUAlreadyExists
					}
					seenSKUs[existing.SKU] = true
				}

				if tt.Description != nil {
					existing.Description = tt.Description
				}

				if tt.Price != nil {
					if *tt.Price < 0 {
						return nil, dto.ErrInvalidPrice
					}
					existing.Price = *tt.Price
				}

				if tt.Currency != nil && strings.TrimSpace(*tt.Currency) != "" {
					existing.Currency = strings.TrimSpace(*tt.Currency)
				}

				if tt.MaxPerOrder != nil {
					if *tt.MaxPerOrder <= 0 {
						return nil, dto.ErrInvalidMaxPerOrder
					}
					existing.MaxPerOrder = *tt.MaxPerOrder
				}

				if tt.SaleStartAt != nil {
					existing.SaleStartAt = *tt.SaleStartAt
				}
				if tt.SaleEndAt != nil {
					existing.SaleEndAt = *tt.SaleEndAt
				}
				if !existing.SaleEndAt.After(existing.SaleStartAt) {
					return nil, dto.ErrInvalidSalePeriod
				}

				if tt.Attributes != nil {
					existing.Attributes = *tt.Attributes
				}

				if tt.Status != nil && *tt.Status != "" {
					existing.Status = *tt.Status
				}

				if userCtx.UserID != nil {
					existing.UpdatedBy = userCtx.UserID
				}

				updatedTicketTypes = append(updatedTicketTypes, *existing)
			}
		}

		// 2. Process newly added ticket types
		for _, tt := range req.TicketTypes {
			if (tt.ID == nil || *tt.ID == uuid.Nil) && !tt.IsDeleted {
				name := ""
				if tt.Name != nil {
					name = strings.TrimSpace(*tt.Name)
				}
				if name == "" {
					return nil, dto.ErrTicketTypeNameEmpty
				}

				sku := ""
				if tt.SKU != nil {
					sku = strings.TrimSpace(*tt.SKU)
				}
				if sku == "" {
					return nil, dto.ErrTicketTypeNameEmpty
				}

				if seenSKUs[sku] {
					return nil, dto.ErrSKUAlreadyExists
				}
				seenSKUs[sku] = true

				existingTT, err := s.eventRepo.FindBySKU(ctx, sku)
				if err == nil && existingTT != nil {
					return nil, dto.ErrSKUAlreadyExists
				}

				price := int64(0)
				if tt.Price != nil {
					if *tt.Price < 0 {
						return nil, dto.ErrInvalidPrice
					}
					price = *tt.Price
				}

				currency := "VND"
				if tt.Currency != nil && strings.TrimSpace(*tt.Currency) != "" {
					currency = strings.TrimSpace(*tt.Currency)
				}

				maxPerOrder := 4
				if tt.MaxPerOrder != nil {
					if *tt.MaxPerOrder <= 0 {
						return nil, dto.ErrInvalidMaxPerOrder
					}
					maxPerOrder = *tt.MaxPerOrder
				}

				saleStartAt := event.StartAt
				if tt.SaleStartAt != nil {
					saleStartAt = *tt.SaleStartAt
				}
				saleEndAt := event.EndAt
				if tt.SaleEndAt != nil {
					saleEndAt = *tt.SaleEndAt
				}
				if !saleEndAt.After(saleStartAt) {
					return nil, dto.ErrInvalidSalePeriod
				}

				attrs := models.JSONMap{}
				if tt.Attributes != nil {
					attrs = *tt.Attributes
				}

				status := models.TicketTypeStatusDraft
				if tt.Status != nil && *tt.Status != "" {
					status = *tt.Status
				}

				newTT := models.TicketType{
					ID:          uuid.New(),
					EventID:     event.ID,
					SKU:         sku,
					Name:        name,
					Description: tt.Description,
					Price:       price,
					Currency:    currency,
					MaxPerOrder: maxPerOrder,
					SaleStartAt: saleStartAt,
					SaleEndAt:   saleEndAt,
					Attributes:  attrs,
					Status:      status,
					CreatedBy:   userCtx.UserID,
					UpdatedBy:   userCtx.UserID,
				}
				updatedTicketTypes = append(updatedTicketTypes, newTT)
			}
		}

		event.TicketTypes = updatedTicketTypes
	} else {
		event.TicketTypes = nil
	}

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	return s.eventRepo.FindByID(ctx, event.ID)
}

func (s *EventService) Delete(ctx context.Context, id uuid.UUID, userCtx dto.CurrentUserContext) error {
	return s.eventRepo.Delete(ctx, id, userCtx.UserID)
}
