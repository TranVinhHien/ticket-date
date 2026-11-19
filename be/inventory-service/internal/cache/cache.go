package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss is returned when the requested key does not exist in the cache.
var ErrCacheMiss = errors.New("cache: key not found")

// Cache defines the general interface for cache operations.
// Implementations can be Redis, In-Memory, or multi-level caches.
type Cache interface {
	// Get retrieves the value for key and unmarshals it into dest.
	// Returns ErrCacheMiss if key does not exist or has expired.
	Get(ctx context.Context, key string, dest any) error

	// Set sets the key-value pair with a time-to-live (TTL).
	// If ttl <= 0, the key will not expire.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete removes one or more keys from the cache.
	Delete(ctx context.Context, keys ...string) error

	// Exists checks whether a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Remember implements the Cache-Aside pattern:
	// If key exists, it populates dest. If miss, it invokes fn(), caches the result with ttl,
	// and unmarshals into dest.
	Remember(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error

	// Numeric Counter operations (for atomic +, - operations and available stock checks)
	GetInt(ctx context.Context, key string) (int64, error)
	SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error
	IncrBy(ctx context.Context, key string, delta int64) (int64, error)
	DecrBy(ctx context.Context, key string, delta int64) (int64, error)

	// Distributed Lock operations (for Cache Stampede / Breakdown prevention)
	AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key string) error

	// Atomic stock operations
	SetIntNX(ctx context.Context, key string, value int64, ttl time.Duration) (bool, error)
	DeductStockAtomic(ctx context.Context, items []DeductStockItem) (*DeductStockResult, error)
}

// DeductStockItem represents an individual stock item to deduct.
type DeductStockItem struct {
	Key      string
	Quantity int64
}

// DeductStatus indicates the outcome of atomic deduction.
type DeductStatus int

const (
	DeductStatusOK DeductStatus = iota
	DeductStatusMissing
	DeductStatusInsufficient
)

// DeductStockResult contains the result of an atomic deduction execution.
type DeductStockResult struct {
	Status      DeductStatus
	FailedKey   string
	MissingKeys []string
}
