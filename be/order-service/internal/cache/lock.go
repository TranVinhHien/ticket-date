package cache

import (
	"context"
	"fmt"
	"time"
)

// ProLockKeyItemPrefix is the standard lock prefix requested for ticket item mutex locking.
const ProLockKeyItemPrefix = "PRO_LOCK_KEY_ITEM:"

// ProLockKeyItem returns the formatted lock key for a given item identifier (e.g. ticketTypeID).
func ProLockKeyItem(id any) string {
	return fmt.Sprintf("%s%v", ProLockKeyItemPrefix, id)
}

// LockOptions configures the behavior of distributed lock and retry.
type LockOptions struct {
	// LockTTL specifies how long the mutex lock is held before auto-expiring (prevents deadlocks).
	LockTTL time.Duration
	// MaxRetries specifies maximum number of retries before falling back to DB.
	MaxRetries int
	// RetryBackoff specifies how long to wait before retrying from cache.
	RetryBackoff time.Duration
}

func resolveLockOptions(opts ...LockOptions) LockOptions {
	cfg := LockOptions{
		LockTTL:      5 * time.Second,
		MaxRetries:   10,
		RetryBackoff: 50 * time.Millisecond,
	}

	if len(opts) > 0 {
		user := opts[0]
		if user.LockTTL > 0 {
			cfg.LockTTL = user.LockTTL
		}
		if user.MaxRetries > 0 {
			cfg.MaxRetries = user.MaxRetries
		}
		if user.RetryBackoff > 0 {
			cfg.RetryBackoff = user.RetryBackoff
		}
	}

	return cfg
}

// GetIntWithLock implements the Cache-Aside pattern with a Distributed Mutex Lock to prevent Cache Breakdown (Stampede).
// Workflow:
// 1. Checks cache for integer value. If hit, returns immediately.
// 2. If miss, attempts to acquire lockKey (PRO_LOCK_KEY_ITEM + ticketId).
// 3. Winner executes fetchFn (queries DB), saves to cache, releases lock, and returns.
// 4. Loser threads wait for RetryBackoff and retry FROM THE VERY BEGINNING (step 1: checking cache again).
func GetIntWithLock(
	ctx context.Context,
	c Cache,
	cacheKey string,
	lockKey string,
	cacheTTL time.Duration,
	fetchFn func(ctx context.Context) (int64, error),
	opts ...LockOptions,
) (int64, error) {
	if c == nil {
		return fetchFn(ctx)
	}

	opt := resolveLockOptions(opts...)

	for attempt := 0; attempt <= opt.MaxRetries; attempt++ {
		// 1. Check cache first
		val, err := c.GetInt(ctx, cacheKey)
		if err == nil {
			return val, nil // Cache Hit!
		}

		// 2. Cache Miss: Attempt to acquire mutex lock
		acquired, err := c.AcquireLock(ctx, lockKey, opt.LockTTL)
		if err != nil {
			// If lock fails return try again
			return -1, fmt.Errorf("please try again")
		}

		if acquired {
			// Lock acquired! Ensure cleanup on error
			defer c.ReleaseLock(ctx, lockKey)

			// Double-check cache in case another worker just finished
			if val, err := c.GetInt(ctx, cacheKey); err == nil {
				return val, nil
			}

			// Query database
			dbVal, err := fetchFn(ctx)
			if err != nil {
				return 0, err
			}

			// Save to cache
			_ = c.SetInt(ctx, cacheKey, dbVal, cacheTTL)

			// Release lock immediately
			_ = c.ReleaseLock(ctx, lockKey)

			return dbVal, nil
		}

		// 3. Lock not acquired: another thread is fetching DB and updating cache.
		// Wait and retry from the VERY BEGINNING (Step 1: check cache again!)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(opt.RetryBackoff):
			// Continue loop -> retries from checking cache
		}
	}

	// Max retries exhausted: Fallback to DB
	return -1, fmt.Errorf("please try again")

}

// RememberWithLock is a generic pattern for any data type (JSON struct, string, list, etc.).
// It prevents Cache Stampede across any service entity by using distributed lock & cache retry.
func RememberWithLock[T any](
	ctx context.Context,
	c Cache,
	cacheKey string,
	lockKey string,
	cacheTTL time.Duration,
	fetchFn func(ctx context.Context) (T, error),
	opts ...LockOptions,
) (T, error) {
	var zero T
	if c == nil {
		return fetchFn(ctx)
	}

	opt := resolveLockOptions(opts...)

	for attempt := 0; attempt <= opt.MaxRetries; attempt++ {
		// 1. Check cache first
		var cachedVal T
		if err := c.Get(ctx, cacheKey, &cachedVal); err == nil {
			return cachedVal, nil // Cache Hit!
		}

		// 2. Cache Miss: Attempt to acquire mutex lock
		acquired, err := c.AcquireLock(ctx, lockKey, opt.LockTTL)
		if err != nil {
			return fetchFn(ctx)
		}

		if acquired {
			defer c.ReleaseLock(ctx, lockKey)

			// Double check cache
			if err := c.Get(ctx, cacheKey, &cachedVal); err == nil {
				return cachedVal, nil
			}

			// Query fallback
			data, err := fetchFn(ctx)
			if err != nil {
				return zero, err
			}

			// Save to cache
			_ = c.Set(ctx, cacheKey, data, cacheTTL)

			// Release lock
			_ = c.ReleaseLock(ctx, lockKey)

			return data, nil
		}

		// 3. Retry from the beginning (checking cache)
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(opt.RetryBackoff):
			// loop
		}
	}

	return fetchFn(ctx)
}
