package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type memoryItem struct {
	value     []byte
	expiresAt time.Time
}

func (i memoryItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(i.expiresAt)
}

type memoryCache struct {
	mu         sync.RWMutex
	items      map[string]memoryItem
	defaultTTL time.Duration
	stopChan   chan struct{}
}

// NewMemoryCache creates an In-Memory implementation of Cache.
// A background goroutine periodically purges expired keys.
func NewMemoryCache(defaultTTL time.Duration, cleanupInterval time.Duration) Cache {
	if cleanupInterval <= 0 {
		cleanupInterval = 1 * time.Minute
	}

	c := &memoryCache{
		items:      make(map[string]memoryItem),
		defaultTTL: defaultTTL,
		stopChan:   make(chan struct{}),
	}

	go c.startJanitor(cleanupInterval)
	return c
}

func (c *memoryCache) startJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopChan:
			ticker.Stop()
			return
		}
	}
}

func (c *memoryCache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, item := range c.items {
		if !item.expiresAt.IsZero() && now.After(item.expiresAt) {
			delete(c.items, k)
		}
	}
}

// Close stops the background eviction cleaner.
func (c *memoryCache) Close() {
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}
}

func (c *memoryCache) Get(ctx context.Context, key string, dest any) error {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists || item.isExpired() {
		return ErrCacheMiss
	}

	if err := json.Unmarshal(item.value, dest); err != nil {
		return fmt.Errorf("memory unmarshal %s: %w", key, err)
	}

	return nil
}

func (c *memoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("memory marshal %s: %w", key, err)
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 && c.defaultTTL > 0 {
		effectiveTTL = c.defaultTTL
	}

	var exp time.Time
	if effectiveTTL > 0 {
		exp = time.Now().Add(effectiveTTL)
	}

	c.mu.Lock()
	c.items[key] = memoryItem{
		value:     data,
		expiresAt: exp,
	}
	c.mu.Unlock()

	return nil
}

func (c *memoryCache) Delete(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.items, k)
	}
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists || item.isExpired() {
		return false, nil
	}

	return true, nil
}

func (c *memoryCache) Remember(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	val, err := fn()
	if err != nil {
		return err
	}

	_ = c.Set(ctx, key, val, ttl)

	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

func (c *memoryCache) GetInt(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists || item.isExpired() {
		return 0, ErrCacheMiss
	}

	val, err := strconv.ParseInt(string(item.value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("memory parse int %s: %w", key, err)
	}

	return val, nil
}

func (c *memoryCache) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	effectiveTTL := ttl
	if effectiveTTL <= 0 && c.defaultTTL > 0 {
		effectiveTTL = c.defaultTTL
	}

	var exp time.Time
	if effectiveTTL > 0 {
		exp = time.Now().Add(effectiveTTL)
	}

	c.mu.Lock()
	c.items[key] = memoryItem{
		value:     []byte(strconv.FormatInt(value, 10)),
		expiresAt: exp,
	}
	c.mu.Unlock()

	return nil
}

func (c *memoryCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	var current int64
	var exp time.Time

	if exists && !item.isExpired() {
		parsed, err := strconv.ParseInt(string(item.value), 10, 64)
		if err == nil {
			current = parsed
		}
		exp = item.expiresAt
	} else if c.defaultTTL > 0 {
		exp = time.Now().Add(c.defaultTTL)
	}

	newVal := current + delta
	c.items[key] = memoryItem{
		value:     []byte(strconv.FormatInt(newVal, 10)),
		expiresAt: exp,
	}

	return newVal, nil
}

func (c *memoryCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return c.IncrBy(ctx, key, -delta)
}

func (c *memoryCache) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if exists && !item.isExpired() {
		return false, nil // already locked
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 {
		effectiveTTL = 5 * time.Second
	}

	c.items[key] = memoryItem{
		value:     []byte("false"),
		expiresAt: time.Now().Add(effectiveTTL),
	}
	return true, nil
}

func (c *memoryCache) ReleaseLock(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}

func (c *memoryCache) SetIntNX(ctx context.Context, key string, value int64, ttl time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if exists && !item.isExpired() {
		return false, nil
	}

	effectiveTTL := ttl
	var exp time.Time
	if effectiveTTL > 0 {
		exp = time.Now().Add(effectiveTTL)
	}

	c.items[key] = memoryItem{
		value:     []byte(strconv.FormatInt(value, 10)),
		expiresAt: exp,
	}

	return true, nil
}

func (c *memoryCache) DeductStockAtomic(ctx context.Context, items []DeductStockItem) (*DeductStockResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(items) == 0 {
		return &DeductStockResult{Status: DeductStatusOK}, nil
	}

	// 1. Check if all keys exist and are not expired
	var missing []string
	for _, item := range items {
		cached, exists := c.items[item.Key]
		if !exists || cached.isExpired() {
			missing = append(missing, item.Key)
		}
	}
	if len(missing) > 0 {
		return &DeductStockResult{
			Status:      DeductStatusMissing,
			MissingKeys: missing,
		}, nil
	}

	// 2. Check stock sufficiency for all keys
	for _, item := range items {
		cached := c.items[item.Key]
		currentStock, err := strconv.ParseInt(string(cached.value), 10, 64)
		if err != nil || currentStock < item.Quantity {
			return &DeductStockResult{
				Status:    DeductStatusInsufficient,
				FailedKey: item.Key,
			}, nil
		}
	}

	// 3. Deduct stock for all keys
	for _, item := range items {
		cached := c.items[item.Key]
		currentStock, _ := strconv.ParseInt(string(cached.value), 10, 64)
		newStock := currentStock - item.Quantity
		c.items[item.Key] = memoryItem{
			value:     []byte(strconv.FormatInt(newStock, 10)),
			expiresAt: cached.expiresAt,
		}
	}

	return &DeductStockResult{Status: DeductStatusOK}, nil
}
