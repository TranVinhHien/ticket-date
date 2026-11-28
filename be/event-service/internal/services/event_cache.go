package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/cache"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/handlers/dto"
	"github.com/TranVinhHien/ticket-date/be/event-service/internal/models"
	"github.com/google/uuid"
)

const (
	defaultEventCacheTTL         = 5 * time.Minute
	defaultEventNegativeCacheTTL = 1 * time.Minute
	eventLockTTL                 = 5 * time.Second
	eventLockMaxRetries          = 10
	eventLockRetryBackoff        = 50 * time.Millisecond
	eventListVersionKey          = "event:list:version"
)

type cachedEvent struct {
	NotFound bool          `json:"not_found"`
	Event    *models.Event `json:"event,omitempty"`
}

type cachedEventSlug struct {
	NotFound bool      `json:"not_found"`
	EventID  uuid.UUID `json:"event_id,omitempty"`
}

type cachedEventList struct {
	Events []*models.Event `json:"events"`
	Total  int64           `json:"total"`
}

type cachedTicketType struct {
	NotFound   bool               `json:"not_found"`
	TicketType *models.TicketType `json:"ticket_type,omitempty"`
}

func eventByIDCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("event:id:%s", id)
}

func eventBySlugCacheKey(slug string) string {
	return fmt.Sprintf("event:slug:%s", slug)
}

func ticketTypeByIDCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("event:ticket_type:%s", id)
}

func eventLockKey(prefix string, id any) string {
	return fmt.Sprintf("EVENT_LOCK_KEY:%s:%v", prefix, id)
}

func eventListQueryHash(query dto.EventQuery) string {
	canonical := fmt.Sprintf(
		"page=%d|limit=%d|sort=%s|order=%s|query=%s|org=%s|venue=%s|status=%s",
		query.GetPage(),
		query.GetLimit(),
		strings.TrimSpace(query.Sort),
		query.GetOrder(),
		strings.TrimSpace(query.Query),
		strings.TrimSpace(query.OrganizerID),
		strings.TrimSpace(query.VenueID),
		strings.TrimSpace(query.Status),
	)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func eventListCacheKey(version int64, queryHash string) string {
	return fmt.Sprintf("event:list:v%d:%s", version, queryHash)
}

// rememberWithNegativeLock implements Cache-Aside + Distributed Mutex Lock with negative caching.
// Only one goroutine queries the DB on cache miss; others wait and retry from cache.
func rememberWithNegativeLock[T any](
	ctx context.Context,
	c cache.Cache,
	cacheKey string,
	lockKey string,
	cacheTTL time.Duration,
	negativeTTL time.Duration,
	fetchFn func(ctx context.Context) (val T, isNotFound bool, err error),
) (T, error) {
	var zero T
	if c == nil {
		val, _, err := fetchFn(ctx)
		return val, err
	}

	for attempt := 0; attempt <= eventLockMaxRetries; attempt++ {
		var cached T
		if err := c.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}

		acquired, err := c.AcquireLock(ctx, lockKey, eventLockTTL)
		if err != nil {
			return zero, cache.ErrPleaseTryAgain
		}

		if acquired {
			defer func() { _ = c.ReleaseLock(ctx, lockKey) }()

			var doubleChecked T
			if err := c.Get(ctx, cacheKey, &doubleChecked); err == nil {
				return doubleChecked, nil
			}

			val, isNotFound, err := fetchFn(ctx)
			if err != nil {
				if isNotFound {
					_ = c.Set(ctx, cacheKey, val, negativeTTL)
				}
				return zero, err
			}

			_ = c.Set(ctx, cacheKey, val, cacheTTL)
			return val, nil
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(eventLockRetryBackoff):
		}
	}

	return zero, cache.ErrPleaseTryAgain
}

func (s *EventService) invalidateEventCache(ctx context.Context, eventID uuid.UUID, slugs []string, ticketTypeIDs []uuid.UUID) {
	if s.cache == nil {
		return
	}

	keySet := make(map[string]struct{})
	if eventID != uuid.Nil {
		keySet[eventByIDCacheKey(eventID)] = struct{}{}
	}
	for _, slug := range slugs {
		if slug != "" {
			keySet[eventBySlugCacheKey(slug)] = struct{}{}
		}
	}
	for _, ttID := range ticketTypeIDs {
		if ttID != uuid.Nil {
			keySet[ticketTypeByIDCacheKey(ttID)] = struct{}{}
		}
	}

	if len(keySet) > 0 {
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		_ = s.cache.Delete(ctx, keys...)
	}

	// Bump list version so cached List queries for any filter/page are invalidated immediately.
	_, _ = s.cache.IncrBy(ctx, eventListVersionKey, 1)
}

func normalizeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	sorted := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			sorted = append(sorted, id)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})
	return sorted
}

func (s *EventService) readTicketTypesFromCache(
	ctx context.Context,
	ids []uuid.UUID,
	resultMap map[uuid.UUID]*models.TicketType,
	notFoundMap map[uuid.UUID]bool,
) []uuid.UUID {
	pendingIDs := make([]uuid.UUID, 0, len(ids))
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := resultMap[id]; ok || notFoundMap[id] {
			continue
		}
		pendingIDs = append(pendingIDs, id)
		keys = append(keys, ticketTypeByIDCacheKey(id))
	}
	if len(pendingIDs) == 0 {
		return nil
	}

	rawMap, err := s.cache.MGet(ctx, keys)
	if err != nil {
		return pendingIDs
	}

	missing := make([]uuid.UUID, 0)
	for i, id := range pendingIDs {
		raw, ok := rawMap[keys[i]]
		if !ok {
			missing = append(missing, id)
			continue
		}
		var cached cachedTicketType
		if err := json.Unmarshal(raw, &cached); err != nil {
			missing = append(missing, id)
			continue
		}
		if cached.NotFound {
			notFoundMap[id] = true
		} else if cached.TicketType != nil {
			resultMap[id] = cached.TicketType
		}
	}
	return missing
}

func (s *EventService) acquireTicketTypeLocks(ctx context.Context, ids []uuid.UUID) ([]string, bool, error) {
	lockedKeys := make([]string, 0, len(ids))
	for _, id := range ids {
		lk := eventLockKey("ticket_type", id)
		acquired, err := s.cache.AcquireLock(ctx, lk, eventLockTTL)
		if err != nil {
			s.releaseLocks(ctx, lockedKeys)
			return nil, false, cache.ErrPleaseTryAgain
		}
		if !acquired {
			s.releaseLocks(ctx, lockedKeys)
			return nil, false, nil
		}
		lockedKeys = append(lockedKeys, lk)
	}
	return lockedKeys, true, nil
}

func (s *EventService) loadMissingTicketTypes(
	ctx context.Context,
	missingIDs []uuid.UUID,
	resultMap map[uuid.UUID]*models.TicketType,
	notFoundMap map[uuid.UUID]bool,
) error {
	stillMissing := s.readTicketTypesFromCache(ctx, missingIDs, resultMap, notFoundMap)
	if len(stillMissing) == 0 {
		return nil
	}

	log.Printf("[EventService.GetTicketTypesByIDs] Querying %d ticket types from DB: %v", len(stillMissing), stillMissing)
	tts, err := s.eventRepo.FindTicketTypesByIDs(ctx, stillMissing)
	if err != nil {
		log.Printf("[EventService.GetTicketTypesByIDs] Error querying ticket types: %v", err)
		return err
	}

	foundMap := make(map[uuid.UUID]*models.TicketType, len(tts))
	for _, tt := range tts {
		foundMap[tt.ID] = tt
	}
	for _, id := range stillMissing {
		if tt, ok := foundMap[id]; ok {
			_ = s.cache.Set(ctx, ticketTypeByIDCacheKey(id), cachedTicketType{TicketType: tt}, s.cacheTTL)
			resultMap[id] = tt
		} else {
			_ = s.cache.Set(ctx, ticketTypeByIDCacheKey(id), cachedTicketType{NotFound: true}, s.negativeCacheTTL)
			notFoundMap[id] = true
		}
	}
	log.Printf("[EventService.GetTicketTypesByIDs] Successfully found %d ticket types in DB", len(tts))
	return nil
}

func (s *EventService) releaseLocks(ctx context.Context, lockKeys []string) {
	if len(lockKeys) > 0 {
		_ = s.cache.Delete(ctx, lockKeys...)
	}
}

func collectOrderedTicketTypes(sortedIDs []uuid.UUID, resultMap map[uuid.UUID]*models.TicketType) []*models.TicketType {
	res := make([]*models.TicketType, 0, len(resultMap))
	for _, id := range sortedIDs {
		if tt, ok := resultMap[id]; ok {
			res = append(res, tt)
		}
	}
	return res
}

func isEventNotFoundErr(err error) bool {
	return errors.Is(err, dto.ErrEventNotFound)
}
