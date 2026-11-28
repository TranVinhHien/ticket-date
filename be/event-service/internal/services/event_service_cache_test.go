package services

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/cache"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

type countingEventRepo struct {
	*fakeEventRepo
	findByIDCalls   int32
	findBySlugCalls int32
	findTTCalls     int32
	listCalls       int32
	queryDelay      time.Duration
	mu              sync.Mutex
	lastTTIDs       []uuid.UUID
}

func newCountingEventRepo() *countingEventRepo {
	return &countingEventRepo{
		fakeEventRepo: newFakeEventRepo(),
	}
}

func (r *countingEventRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	atomic.AddInt32(&r.findByIDCalls, 1)
	if r.queryDelay > 0 {
		time.Sleep(r.queryDelay)
	}
	return r.fakeEventRepo.FindByID(ctx, id)
}

func (r *countingEventRepo) FindBySlug(ctx context.Context, slug string) (*models.Event, error) {
	atomic.AddInt32(&r.findBySlugCalls, 1)
	if r.queryDelay > 0 {
		time.Sleep(r.queryDelay)
	}
	return r.fakeEventRepo.FindBySlug(ctx, slug)
}

func (r *countingEventRepo) FindTicketTypesByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.TicketType, error) {
	atomic.AddInt32(&r.findTTCalls, 1)
	r.mu.Lock()
	r.lastTTIDs = append([]uuid.UUID(nil), ids...)
	r.mu.Unlock()
	if r.queryDelay > 0 {
		time.Sleep(r.queryDelay)
	}
	return r.fakeEventRepo.FindTicketTypesByIDs(ctx, ids)
}

func (r *countingEventRepo) List(ctx context.Context, query dto.EventQuery) ([]*models.Event, int64, error) {
	atomic.AddInt32(&r.listCalls, 1)
	if r.queryDelay > 0 {
		time.Sleep(r.queryDelay)
	}
	return r.fakeEventRepo.List(ctx, query)
}

func seedSampleEvent(repo *countingEventRepo) (*models.Event, uuid.UUID) {
	eventID := uuid.New()
	ttID := uuid.New()
	now := time.Now()
	ev := &models.Event{
		ID:          eventID,
		OrganizerID: uuid.New(),
		VenueID:     uuid.New(),
		Name:        "Concert 2026",
		Slug:        "concert-2026",
		StartAt:     now.Add(24 * time.Hour),
		EndAt:       now.Add(28 * time.Hour),
		Status:      models.EventStatusPublished,
		TicketTypes: []models.TicketType{
			{
				ID:          ttID,
				EventID:     eventID,
				SKU:         "SKU-CONCERT-VIP",
				Name:        "VIP",
				Price:       500000,
				Currency:    "VND",
				MaxPerOrder: 4,
				SaleStartAt: now,
				SaleEndAt:   now.Add(24 * time.Hour),
				Status:      models.TicketTypeStatusActive,
			},
		},
	}
	_ = repo.Create(context.Background(), ev)
	return ev, ttID
}

func TestEventService_GetByID_StampedePreventionAndCacheHit(t *testing.T) {
	repo := newCountingEventRepo()
	repo.queryDelay = 30 * time.Millisecond
	ev, _ := seedSampleEvent(repo)

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	const concurrency = 10
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	results := make([]*models.Event, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = svc.GetByID(ctx, ev.ID)
		}(i)
	}
	wg.Wait()

	for i := 0; i < concurrency; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d failed: %v", i, errs[i])
		}
		if results[i] == nil || results[i].ID != ev.ID {
			t.Fatalf("goroutine %d got unexpected event: %+v", i, results[i])
		}
	}

	if calls := atomic.LoadInt32(&repo.findByIDCalls); calls != 1 {
		t.Fatalf("expected FindByID to be called exactly 1 time, got %d", calls)
	}

	// Subsequent call must hit cache
	got, err := svc.GetByID(ctx, ev.ID)
	if err != nil || got.Slug != ev.Slug {
		t.Fatalf("expected cached event, got %+v, err=%v", got, err)
	}
	if calls := atomic.LoadInt32(&repo.findByIDCalls); calls != 1 {
		t.Fatalf("expected FindByID count to remain 1, got %d", calls)
	}
}

func TestEventService_GetBySlug_StampedePreventionAndNegativeCache(t *testing.T) {
	repo := newCountingEventRepo()
	repo.queryDelay = 25 * time.Millisecond
	ev, _ := seedSampleEvent(repo)

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.GetBySlug(ctx, ev.Slug)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d failed: %v", i, err)
		}
	}
	if calls := atomic.LoadInt32(&repo.findBySlugCalls); calls != 1 {
		t.Fatalf("expected FindBySlug called 1 time, got %d", calls)
	}

	// Verify slug cache stores EventID and warms event:id:<id> so GetByID hits cache (0 DB calls)
	var slugMapping cachedEventSlug
	if err := memCache.Get(ctx, eventBySlugCacheKey(ev.Slug), &slugMapping); err != nil || slugMapping.EventID != ev.ID {
		t.Fatalf("expected slug cache to store EventID %s, got %+v, err=%v", ev.ID, slugMapping, err)
	}
	gotByID, err := svc.GetByID(ctx, ev.ID)
	if err != nil || gotByID.ID != ev.ID {
		t.Fatalf("expected GetByID to hit warmed cache, got %+v, err=%v", gotByID, err)
	}
	if calls := atomic.LoadInt32(&repo.findByIDCalls); calls != 0 {
		t.Fatalf("expected 0 FindByID DB calls after GetBySlug warmed ID cache, got %d", calls)
	}
	// Subsequent GetBySlug resolves ID from slug cache -> reads event from ID cache (0 extra DB calls)
	gotBySlug, err := svc.GetBySlug(ctx, ev.Slug)
	if err != nil || gotBySlug.ID != ev.ID {
		t.Fatalf("expected GetBySlug cache hit via ID cache, got %+v, err=%v", gotBySlug, err)
	}
	if calls := atomic.LoadInt32(&repo.findBySlugCalls); calls != 1 || atomic.LoadInt32(&repo.findByIDCalls) != 0 {
		t.Fatalf("expected no extra DB calls, got slug=%d id=%d", repo.findBySlugCalls, repo.findByIDCalls)
	}

	// Negative caching for non-existent slug
	repo.queryDelay = 0
	_, err = svc.GetBySlug(ctx, "non-existent-slug")
	if !errors.Is(err, dto.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound, got %v", err)
	}
	_, err = svc.GetBySlug(ctx, "non-existent-slug")
	if !errors.Is(err, dto.ErrEventNotFound) {
		t.Fatalf("expected ErrEventNotFound from negative cache, got %v", err)
	}
	if calls := atomic.LoadInt32(&repo.findBySlugCalls); calls != 2 {
		t.Fatalf("expected FindBySlug count to be 2 (1 for hit, 1 for first miss), got %d", calls)
	}
}

func TestEventService_GetTicketTypesByIDs_PartialCacheAndNegativeCache(t *testing.T) {
	repo := newCountingEventRepo()
	ev, tt1ID := seedSampleEvent(repo)
	tt2ID := uuid.New()
	repo.ticketTypes[tt2ID] = &models.TicketType{
		ID:       tt2ID,
		EventID:  ev.ID,
		SKU:      "SKU-CONCERT-STD",
		Name:     "Standard",
		Price:    200000,
		Currency: "VND",
	}
	missingID := uuid.New()

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	// 1. Fetch tt1ID -> caches tt1ID
	res1, err := svc.GetTicketTypesByIDs(ctx, []uuid.UUID{tt1ID})
	if err != nil || len(res1) != 1 || res1[0].ID != tt1ID {
		t.Fatalf("expected 1 ticket type %s, got %+v, err=%v", tt1ID, res1, err)
	}
	if atomic.LoadInt32(&repo.findTTCalls) != 1 {
		t.Fatalf("expected 1 DB call, got %d", repo.findTTCalls)
	}

	// 2. Fetch [tt1ID, tt2ID, missingID] -> only [tt2ID, missingID] should hit DB
	res2, err := svc.GetTicketTypesByIDs(ctx, []uuid.UUID{tt1ID, tt2ID, missingID})
	if err != nil || len(res2) != 2 {
		t.Fatalf("expected 2 ticket types, got %d, err=%v", len(res2), err)
	}
	if atomic.LoadInt32(&repo.findTTCalls) != 2 {
		t.Fatalf("expected 2 DB calls total, got %d", repo.findTTCalls)
	}
	repo.mu.Lock()
	queriedCount := len(repo.lastTTIDs)
	repo.mu.Unlock()
	if queriedCount != 2 {
		t.Fatalf("expected DB to be queried only for 2 missing IDs, got %d", queriedCount)
	}

	// 3. Repeat fetch [tt1ID, tt2ID, missingID] -> all 3 (including negative missingID) served from cache
	res3, err := svc.GetTicketTypesByIDs(ctx, []uuid.UUID{tt1ID, tt2ID, missingID})
	if err != nil || len(res3) != 2 {
		t.Fatalf("expected 2 ticket types from cache, got %d, err=%v", len(res3), err)
	}
	if atomic.LoadInt32(&repo.findTTCalls) != 2 {
		t.Fatalf("expected DB call count to remain 2, got %d", repo.findTTCalls)
	}
}

func TestEventService_CacheInvalidationOnUpdateAndDelete(t *testing.T) {
	repo := newCountingEventRepo()
	orgRepo := newFakeOrganizerRepo()
	venueRepo := newFakeVenueRepo()
	ev, ttID := seedSampleEvent(repo)

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, orgRepo, venueRepo, memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	// Warm caches for ID, Slug, and TicketType
	if _, err := svc.GetByID(ctx, ev.ID); err != nil {
		t.Fatalf("warmup GetByID failed: %v", err)
	}
	if _, err := svc.GetBySlug(ctx, ev.Slug); err != nil {
		t.Fatalf("warmup GetBySlug failed: %v", err)
	}
	if _, err := svc.GetTicketTypesByIDs(ctx, []uuid.UUID{ttID}); err != nil {
		t.Fatalf("warmup GetTicketTypesByIDs failed: %v", err)
	}

	// Update slug -> must invalidate old slug, id, and ticket type caches
	newSlug := "concert-2026-updated"
	_, err := svc.Update(ctx, ev.ID, dto.UpdateEventRequest{Slug: &newSlug}, dto.CurrentUserContext{})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if exists, _ := memCache.Exists(ctx, eventByIDCacheKey(ev.ID)); exists {
		t.Fatalf("expected event:id cache to be invalidated after Update")
	}
	if exists, _ := memCache.Exists(ctx, eventBySlugCacheKey(ev.Slug)); exists {
		t.Fatalf("expected old event:slug cache to be invalidated after Update")
	}
	if exists, _ := memCache.Exists(ctx, ticketTypeByIDCacheKey(ttID)); exists {
		t.Fatalf("expected ticket_type cache to be invalidated after Update")
	}

	// Re-warm with new slug, then Delete
	if _, err := svc.GetBySlug(ctx, newSlug); err != nil {
		t.Fatalf("GetBySlug after update failed: %v", err)
	}
	if err := svc.Delete(ctx, ev.ID, dto.CurrentUserContext{}); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if exists, _ := memCache.Exists(ctx, eventBySlugCacheKey(newSlug)); exists {
		t.Fatalf("expected event:slug cache to be invalidated after Delete")
	}
}

func TestEventService_RetryExhaustionReturnsPleaseTryAgain(t *testing.T) {
	repo := newCountingEventRepo()
	ev, ttID := seedSampleEvent(repo)
	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	// Pre-acquire locks so service hits max retries
	_, _ = memCache.AcquireLock(ctx, eventLockKey("id", ev.ID), 5*time.Second)
	_, _ = memCache.AcquireLock(ctx, eventLockKey("slug", ev.Slug), 5*time.Second)
	_, _ = memCache.AcquireLock(ctx, eventLockKey("ticket_type", ttID), 5*time.Second)

	if _, err := svc.GetByID(ctx, ev.ID); !errors.Is(err, cache.ErrPleaseTryAgain) {
		t.Fatalf("expected ErrPleaseTryAgain for GetByID, got %v", err)
	}
	if _, err := svc.GetBySlug(ctx, ev.Slug); !errors.Is(err, cache.ErrPleaseTryAgain) {
		t.Fatalf("expected ErrPleaseTryAgain for GetBySlug, got %v", err)
	}
	if _, err := svc.GetTicketTypesByIDs(ctx, []uuid.UUID{ttID}); !errors.Is(err, cache.ErrPleaseTryAgain) {
		t.Fatalf("expected ErrPleaseTryAgain for GetTicketTypesByIDs, got %v", err)
	}
	if atomic.LoadInt32(&repo.findByIDCalls) != 0 || atomic.LoadInt32(&repo.findBySlugCalls) != 0 || atomic.LoadInt32(&repo.findTTCalls) != 0 {
		t.Fatalf("expected 0 DB calls when lock retries are exhausted")
	}
}

func TestEventService_List_CacheByParamsStampedeAndInvalidation(t *testing.T) {
	repo := newCountingEventRepo()
	repo.queryDelay = 25 * time.Millisecond
	ev, _ := seedSampleEvent(repo)

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 5*time.Minute, 1*time.Minute)
	ctx := context.Background()

	q1 := dto.EventQuery{
		BaseQuery: dto.BaseQuery{Page: 1, Limit: 10, Order: "asc"},
		Status:    string(models.EventStatusPublished),
	}

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	totals := make([]int64, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			events, total, err := svc.List(ctx, q1)
			errs[idx] = err
			totals[idx] = total
			if err == nil && len(events) != 1 {
				t.Errorf("goroutine %d expected 1 event, got %d", idx, len(events))
			}
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d failed: %v", i, err)
		}
		if totals[i] != 1 {
			t.Fatalf("goroutine %d expected total 1, got %d", i, totals[i])
		}
	}
	if calls := atomic.LoadInt32(&repo.listCalls); calls != 1 {
		t.Fatalf("expected List DB call once under stampede, got %d", calls)
	}

	// Same params (even with whitespace/default casing) must hit cache
	repo.queryDelay = 0
	q1Same := dto.EventQuery{
		BaseQuery: dto.BaseQuery{Page: 1, Limit: 10, Order: "ASC"},
		Status:    " " + string(models.EventStatusPublished) + " ",
	}
	if _, total, err := svc.List(ctx, q1Same); err != nil || total != 1 {
		t.Fatalf("expected cached list hit, got total=%d, err=%v", total, err)
	}
	if calls := atomic.LoadInt32(&repo.listCalls); calls != 1 {
		t.Fatalf("expected List DB count to remain 1, got %d", calls)
	}

	// Different params must use a separate cache key and query DB
	q2 := dto.EventQuery{
		BaseQuery: dto.BaseQuery{Page: 2, Limit: 10, Order: "asc"},
		Status:    string(models.EventStatusPublished),
	}
	if _, total, err := svc.List(ctx, q2); err != nil || total != 1 {
		t.Fatalf("expected q2 list query, got total=%d, err=%v", total, err)
	}
	if calls := atomic.LoadInt32(&repo.listCalls); calls != 2 {
		t.Fatalf("expected List DB count 2 for different query params, got %d", calls)
	}

	// Updating event invalidates List cache via version bump
	newName := "Concert 2026 Updated"
	if _, err := svc.Update(ctx, ev.ID, dto.UpdateEventRequest{Name: &newName}, dto.CurrentUserContext{}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	eventsAfter, _, err := svc.List(ctx, q1)
	if err != nil || len(eventsAfter) != 1 || eventsAfter[0].Name != newName {
		t.Fatalf("expected refreshed list after Update, got %+v, err=%v", eventsAfter, err)
	}
	if calls := atomic.LoadInt32(&repo.listCalls); calls != 3 {
		t.Fatalf("expected List DB count 3 after invalidation, got %d", calls)
	}
}

func TestEventService_List_TTLExpiration(t *testing.T) {
	repo := newCountingEventRepo()
	seedSampleEvent(repo)

	memCache := cache.NewMemoryCache(5*time.Minute, 1*time.Minute)
	svc := NewEventService(repo, newFakeOrganizerRepo(), newFakeVenueRepo(), memCache, 40*time.Millisecond, 20*time.Millisecond)
	ctx := context.Background()

	q := dto.EventQuery{BaseQuery: dto.BaseQuery{Page: 1, Limit: 10}}
	if _, _, err := svc.List(ctx, q); err != nil {
		t.Fatalf("first List failed: %v", err)
	}
	if atomic.LoadInt32(&repo.listCalls) != 1 {
		t.Fatalf("expected 1 DB call, got %d", repo.listCalls)
	}

	time.Sleep(60 * time.Millisecond)

	if _, _, err := svc.List(ctx, q); err != nil {
		t.Fatalf("second List after TTL failed: %v", err)
	}
	if atomic.LoadInt32(&repo.listCalls) != 2 {
		t.Fatalf("expected 2 DB calls after TTL expiration, got %d", repo.listCalls)
	}
}
