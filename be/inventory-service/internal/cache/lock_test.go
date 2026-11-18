package cache_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/cache"
)

func TestGetIntWithLock_CacheStampedePrevention(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	cacheKey := "test:stock:available:123"
	lockKey := cache.ProLockKeyItem("123")

	var dbQueryCount int32
	fetchFn := func(ctx context.Context) (int64, error) {
		atomic.AddInt32(&dbQueryCount, 1)
		time.Sleep(30 * time.Millisecond) // simulate DB query latency
		return 88, nil
	}

	concurrency := 10
	var wg sync.WaitGroup
	results := make([]int64, concurrency)
	errors := make([]error, concurrency)

	// Simulate 10 concurrent requests when cache is empty (Cache Breakdown scenario)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val, err := cache.GetIntWithLock(
				ctx,
				c,
				cacheKey,
				lockKey,
				5*time.Minute,
				fetchFn,
				cache.LockOptions{
					LockTTL:      2 * time.Second,
					MaxRetries:   10,
					RetryBackoff: 15 * time.Millisecond,
				},
			)
			results[idx] = val
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// 1. All concurrent requests must succeed with the correct value
	for i := 0; i < concurrency; i++ {
		if errors[i] != nil {
			t.Fatalf("goroutine %d failed with err: %v", i, errors[i])
		}
		if results[i] != 88 {
			t.Fatalf("goroutine %d got unexpected value: %d", i, results[i])
		}
	}

	// 2. Database MUST be queried only ONCE!
	calls := atomic.LoadInt32(&dbQueryCount)
	if calls != 1 {
		t.Fatalf("expected DB fetch to be called exactly 1 time, got %d times", calls)
	}

	// 3. Subsequent calls should hit cache without locking or querying DB
	val, err := cache.GetIntWithLock(ctx, c, cacheKey, lockKey, 5*time.Minute, fetchFn)
	if err != nil || val != 88 {
		t.Fatalf("expected 88 from cache, got %d, err=%v", val, err)
	}
	if atomic.LoadInt32(&dbQueryCount) != 1 {
		t.Fatalf("subsequent call should hit cache, db count should remain 1")
	}
}

type eventDetails struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`
	Price     int    `json:"price"`
}

func TestRememberWithLock_GenericType(t *testing.T) {
	c := cache.NewMemoryCache(10*time.Minute, 1*time.Minute)
	ctx := context.Background()

	cacheKey := "test:event:details:456"
	lockKey := cache.ProLockKeyItem("event_456")

	var dbQueryCount int32
	fetchFn := func(ctx context.Context) (eventDetails, error) {
		atomic.AddInt32(&dbQueryCount, 1)
		time.Sleep(20 * time.Millisecond)
		return eventDetails{EventID: "456", EventName: "Coldplay Live", Price: 1500000}, nil
	}

	concurrency := 5
	var wg sync.WaitGroup
	results := make([]eventDetails, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data, err := cache.RememberWithLock(
				ctx,
				c,
				cacheKey,
				lockKey,
				5*time.Minute,
				fetchFn,
			)
			if err != nil {
				t.Errorf("error: %v", err)
			}
			results[idx] = data
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		if results[i].EventID != "456" || results[i].Price != 1500000 {
			t.Fatalf("goroutine %d got invalid data: %+v", i, results[i])
		}
	}

	if atomic.LoadInt32(&dbQueryCount) != 1 {
		t.Fatalf("expected DB fetch for generic object to be called exactly 1 time, got %d", dbQueryCount)
	}
}
