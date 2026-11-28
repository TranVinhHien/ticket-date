package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/cache"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/repository"
	"github.com/google/uuid"
)

type EventService struct {
	eventRepo        repository.IEventRepository
	organizerRepo    repository.IOrganizerRepository
	venueRepo        repository.IVenueRepository
	cache            cache.Cache
	cacheTTL         time.Duration
	negativeCacheTTL time.Duration
}

func NewEventService(
	eventRepo repository.IEventRepository,
	organizerRepo repository.IOrganizerRepository,
	venueRepo repository.IVenueRepository,
	cacheEngine cache.Cache,
	cacheTTL time.Duration,
	negativeCacheTTL time.Duration,
) *EventService {
	if cacheTTL <= 0 {
		cacheTTL = defaultEventCacheTTL
	}
	if negativeCacheTTL <= 0 {
		negativeCacheTTL = defaultEventNegativeCacheTTL
	}
	return &EventService{
		eventRepo:        eventRepo,
		organizerRepo:    organizerRepo,
		venueRepo:        venueRepo,
		cache:            cacheEngine,
		cacheTTL:         cacheTTL,
		negativeCacheTTL: negativeCacheTTL,
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

	slugSource := name
	if strings.TrimSpace(req.Slug) != "" {
		slugSource = req.Slug
	}
	slug := Slugify(slugSource)
	if existing, err := s.eventRepo.FindBySlug(ctx, slug); err == nil && existing != nil {
		return nil, dto.ErrSlugAlreadyExists
	}

	status := req.Status
	if status == "" {
		status = models.EventStatusDraft
	}

	ticketTypes, err := s.buildCreateTicketTypes(ctx, req.TicketTypes)
	if err != nil {
		return nil, err
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

	ttIDs := make([]uuid.UUID, 0, len(event.TicketTypes))
	for _, tt := range event.TicketTypes {
		ttIDs = append(ttIDs, tt.ID)
	}
	s.invalidateEventCache(ctx, event.ID, []string{event.Slug}, ttIDs)

	return s.eventRepo.FindByID(ctx, event.ID)
}

func (s *EventService) GetByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	res, err := rememberWithNegativeLock(
		ctx, s.cache, eventByIDCacheKey(id), eventLockKey("id", id),
		s.cacheTTL, s.negativeCacheTTL,
		func(ctx context.Context) (cachedEvent, bool, error) {
			ev, err := s.eventRepo.FindByID(ctx, id)
			if err != nil {
				return cachedEvent{NotFound: true}, isEventNotFoundErr(err), err
			}
			return cachedEvent{Event: ev}, false, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if res.NotFound {
		return nil, dto.ErrEventNotFound
	}
	return res.Event, nil
}

func (s *EventService) GetBySlug(ctx context.Context, slug string) (*models.Event, error) {
	if s.cache == nil {
		return s.eventRepo.FindBySlug(ctx, slug)
	}

	var fetchedFromDB *models.Event
	res, err := rememberWithNegativeLock(
		ctx, s.cache, eventBySlugCacheKey(slug), eventLockKey("slug", slug),
		s.cacheTTL, s.negativeCacheTTL,
		func(ctx context.Context) (cachedEventSlug, bool, error) {
			ev, err := s.eventRepo.FindBySlug(ctx, slug)
			if err != nil {
				return cachedEventSlug{NotFound: true}, isEventNotFoundErr(err), err
			}
			fetchedFromDB = ev
			_ = s.cache.Set(ctx, eventByIDCacheKey(ev.ID), cachedEvent{Event: ev}, s.cacheTTL)
			return cachedEventSlug{EventID: ev.ID}, false, nil
		},
	)
	if err != nil {
		return nil, err
	}
	if res.NotFound {
		return nil, dto.ErrEventNotFound
	}
	if fetchedFromDB != nil {
		return fetchedFromDB, nil
	}
	return s.GetByID(ctx, res.EventID)
}

func (s *EventService) List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	if s.cache == nil {
		return s.eventRepo.List(ctx, query)
	}

	version, err := s.cache.GetInt(ctx, eventListVersionKey)
	if err != nil {
		version = 0
	}

	queryHash := eventListQueryHash(query)
	res, err := cache.RememberWithLock(
		ctx,
		s.cache,
		eventListCacheKey(version, queryHash),
		eventLockKey("list", fmt.Sprintf("v%d:%s", version, queryHash)),
		s.cacheTTL,
		func(ctx context.Context) (cachedEventList, error) {
			events, total, err := s.eventRepo.List(ctx, query)
			if err != nil {
				return cachedEventList{}, err
			}
			return cachedEventList{Events: events, Total: total}, nil
		},
		cache.LockOptions{
			LockTTL:      eventLockTTL,
			MaxRetries:   eventLockMaxRetries,
			RetryBackoff: eventLockRetryBackoff,
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return res.Events, res.Total, nil
}

func (s *EventService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateEventRequest, userCtx dto.CurrentUserContext) (*models.Event, error) {
	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldSlug := event.Slug
	ttIDs := make([]uuid.UUID, 0, len(event.TicketTypes))
	for _, tt := range event.TicketTypes {
		ttIDs = append(ttIDs, tt.ID)
	}

	if err := s.applyEventFieldUpdates(ctx, event, req, userCtx); err != nil {
		return nil, err
	}

	if req.TicketTypes != nil {
		updatedTTs, err := s.syncEventTicketTypes(ctx, event, req.TicketTypes, userCtx)
		if err != nil {
			return nil, err
		}
		event.TicketTypes = updatedTTs
	} else {
		event.TicketTypes = nil
	}

	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, err
	}

	updated, err := s.eventRepo.FindByID(ctx, event.ID)
	if err != nil {
		s.invalidateEventCache(ctx, id, []string{oldSlug, event.Slug}, ttIDs)
		return nil, err
	}

	for _, tt := range updated.TicketTypes {
		ttIDs = append(ttIDs, tt.ID)
	}
	s.invalidateEventCache(ctx, id, []string{oldSlug, updated.Slug}, ttIDs)
	return updated, nil
}

func (s *EventService) Delete(ctx context.Context, id uuid.UUID, userCtx dto.CurrentUserContext) error {
	var slugs []string
	var ttIDs []uuid.UUID
	if existing, err := s.eventRepo.FindByID(ctx, id); err == nil && existing != nil {
		slugs = append(slugs, existing.Slug)
		for _, tt := range existing.TicketTypes {
			ttIDs = append(ttIDs, tt.ID)
		}
	}

	if err := s.eventRepo.Delete(ctx, id, userCtx.UserID); err != nil {
		return err
	}

	s.invalidateEventCache(ctx, id, slugs, ttIDs)
	return nil
}

func (s *EventService) GetTicketTypesByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error) {
	sortedIDs := normalizeUUIDs(ids)
	if len(sortedIDs) == 0 {
		return []*models.TicketType{}, nil
	}
	if s.cache == nil {
		return s.eventRepo.FindTicketTypesByIDs(ctx, sortedIDs)
	}

	resultMap := make(map[uuid.UUID]*models.TicketType, len(sortedIDs))
	notFoundMap := make(map[uuid.UUID]bool, len(sortedIDs))

	for attempt := 0; attempt <= eventLockMaxRetries; attempt++ {
		missingIDs := s.readTicketTypesFromCache(ctx, sortedIDs, resultMap, notFoundMap)
		if len(missingIDs) == 0 {
			return collectOrderedTicketTypes(sortedIDs, resultMap), nil
		}

		lockedKeys, acquiredAll, err := s.acquireTicketTypeLocks(ctx, missingIDs)
		if err != nil {
			return nil, err
		}
		if !acquiredAll {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(eventLockRetryBackoff):
			}
			continue
		}

		err = s.loadMissingTicketTypes(ctx, missingIDs, resultMap, notFoundMap)
		s.releaseLocks(ctx, lockedKeys)
		if err != nil {
			return nil, err
		}
		return collectOrderedTicketTypes(sortedIDs, resultMap), nil
	}

	return nil, cache.ErrPleaseTryAgain
}

func (s *EventService) buildCreateTicketTypes(ctx context.Context, inputs []dto.CreateTicketTypeInput) ([]models.TicketType, error) {
	seenSKUs := make(map[string]bool, len(inputs))
	ticketTypes := make([]models.TicketType, 0, len(inputs))

	for _, tt := range inputs {
		ttName := strings.TrimSpace(tt.Name)
		sku := strings.TrimSpace(tt.SKU)
		if ttName == "" || sku == "" {
			return nil, dto.ErrTicketTypeNameEmpty
		}
		if seenSKUs[sku] {
			return nil, dto.ErrSKUAlreadyExists
		}
		seenSKUs[sku] = true

		if tt.Price < 0 {
			return nil, dto.ErrInvalidPrice
		}
		if !tt.SaleEndAt.After(tt.SaleStartAt) {
			return nil, dto.ErrInvalidSalePeriod
		}
		if existingTT, err := s.eventRepo.FindBySKU(ctx, sku); err == nil && existingTT != nil {
			return nil, dto.ErrSKUAlreadyExists
		}

		maxPerOrder := tt.MaxPerOrder
		if maxPerOrder <= 0 {
			maxPerOrder = 4
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
	return ticketTypes, nil
}

func (s *EventService) applyEventFieldUpdates(
	ctx context.Context,
	event *models.Event,
	req dto.UpdateEventRequest,
	userCtx dto.CurrentUserContext,
) error {
	if req.OrganizerID != nil {
		if _, err := s.organizerRepo.FindByID(ctx, *req.OrganizerID); err != nil {
			return dto.ErrOrganizerNotFound
		}
		event.OrganizerID = *req.OrganizerID
	}
	if req.VenueID != nil {
		if _, err := s.venueRepo.FindByID(ctx, *req.VenueID); err != nil {
			return dto.ErrVenueNotFound
		}
		event.VenueID = *req.VenueID
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return dto.ErrEventNameEmpty
		}
		event.Name = name
	}
	if req.Slug != nil {
		newSlug := Slugify(*req.Slug)
		if newSlug != event.Slug {
			if existing, err := s.eventRepo.FindBySlug(ctx, newSlug); err == nil && existing != nil && existing.ID != event.ID {
				return dto.ErrSlugAlreadyExists
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
		return dto.ErrInvalidPeriod
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
	return nil
}

func (s *EventService) syncEventTicketTypes(
	ctx context.Context,
	event *models.Event,
	inputs []dto.UpdateTicketTypeInput,
	userCtx dto.CurrentUserContext,
) ([]models.TicketType, error) {
	existingMap := make(map[uuid.UUID]*models.TicketType, len(event.TicketTypes))
	touchedIDs := make(map[uuid.UUID]bool, len(inputs))
	for i := range event.TicketTypes {
		existingMap[event.TicketTypes[i].ID] = &event.TicketTypes[i]
	}
	for _, tt := range inputs {
		if tt.ID != nil && *tt.ID != uuid.Nil {
			touchedIDs[*tt.ID] = true
		}
	}

	seenSKUs := make(map[string]bool)
	var result []models.TicketType
	for _, existing := range event.TicketTypes {
		if !touchedIDs[existing.ID] {
			result = append(result, existing)
			seenSKUs[existing.SKU] = true
		}
	}

	for _, tt := range inputs {
		if tt.ID != nil && *tt.ID != uuid.Nil {
			existing, ok := existingMap[*tt.ID]
			if !ok {
				return nil, dto.ErrTicketTypeNotFound
			}
			if tt.IsDeleted {
				continue
			}
			if err := s.updateExistingTicketType(ctx, existing, tt, seenSKUs, userCtx); err != nil {
				return nil, err
			}
			result = append(result, *existing)
		}
	}
	for _, tt := range inputs {
		if (tt.ID == nil || *tt.ID == uuid.Nil) && !tt.IsDeleted {
			newTT, err := s.createNewTicketType(ctx, event, tt, seenSKUs, userCtx)
			if err != nil {
				return nil, err
			}
			result = append(result, newTT)
		}
	}
	return result, nil
}

func (s *EventService) updateExistingTicketType(
	ctx context.Context,
	existing *models.TicketType,
	tt dto.UpdateTicketTypeInput,
	seenSKUs map[string]bool,
	userCtx dto.CurrentUserContext,
) error {
	if tt.Name != nil {
		name := strings.TrimSpace(*tt.Name)
		if name == "" {
			return dto.ErrTicketTypeNameEmpty
		}
		existing.Name = name
	}
	if tt.SKU != nil {
		sku := strings.TrimSpace(*tt.SKU)
		if sku == "" {
			return dto.ErrTicketTypeNameEmpty
		}
		if seenSKUs[sku] {
			return dto.ErrSKUAlreadyExists
		}
		if sku != existing.SKU {
			if existingTT, err := s.eventRepo.FindBySKU(ctx, sku); err == nil && existingTT != nil && existingTT.ID != existing.ID {
				return dto.ErrSKUAlreadyExists
			}
		}
		existing.SKU = sku
	} else if seenSKUs[existing.SKU] {
		return dto.ErrSKUAlreadyExists
	}
	seenSKUs[existing.SKU] = true

	if tt.Description != nil {
		existing.Description = tt.Description
	}
	if tt.Price != nil {
		if *tt.Price < 0 {
			return dto.ErrInvalidPrice
		}
		existing.Price = *tt.Price
	}
	if tt.Currency != nil && strings.TrimSpace(*tt.Currency) != "" {
		existing.Currency = strings.TrimSpace(*tt.Currency)
	}
	if tt.MaxPerOrder != nil {
		if *tt.MaxPerOrder <= 0 {
			return dto.ErrInvalidMaxPerOrder
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
		return dto.ErrInvalidSalePeriod
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
	return nil
}

func (s *EventService) createNewTicketType(
	ctx context.Context,
	event *models.Event,
	tt dto.UpdateTicketTypeInput,
	seenSKUs map[string]bool,
	userCtx dto.CurrentUserContext,
) (models.TicketType, error) {
	name, sku := "", ""
	if tt.Name != nil {
		name = strings.TrimSpace(*tt.Name)
	}
	if tt.SKU != nil {
		sku = strings.TrimSpace(*tt.SKU)
	}
	if name == "" || sku == "" {
		return models.TicketType{}, dto.ErrTicketTypeNameEmpty
	}
	if seenSKUs[sku] {
		return models.TicketType{}, dto.ErrSKUAlreadyExists
	}
	seenSKUs[sku] = true

	if existingTT, err := s.eventRepo.FindBySKU(ctx, sku); err == nil && existingTT != nil {
		return models.TicketType{}, dto.ErrSKUAlreadyExists
	}

	var price int64
	if tt.Price != nil {
		if *tt.Price < 0 {
			return models.TicketType{}, dto.ErrInvalidPrice
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
			return models.TicketType{}, dto.ErrInvalidMaxPerOrder
		}
		maxPerOrder = *tt.MaxPerOrder
	}

	saleStartAt, saleEndAt := event.StartAt, event.EndAt
	if tt.SaleStartAt != nil {
		saleStartAt = *tt.SaleStartAt
	}
	if tt.SaleEndAt != nil {
		saleEndAt = *tt.SaleEndAt
	}
	if !saleEndAt.After(saleStartAt) {
		return models.TicketType{}, dto.ErrInvalidSalePeriod
	}

	attrs := models.JSONMap{}
	if tt.Attributes != nil {
		attrs = *tt.Attributes
	}
	status := models.TicketTypeStatusDraft
	if tt.Status != nil && *tt.Status != "" {
		status = *tt.Status
	}

	return models.TicketType{
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
	}, nil
}
