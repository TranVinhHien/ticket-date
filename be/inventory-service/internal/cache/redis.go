package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewRedisCache creates a new Cache implementation backed by Redis.
func NewRedisCache(client *redis.Client, defaultTTL time.Duration) Cache {
	return &redisCache{
		client:     client,
		defaultTTL: defaultTTL,
	}
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) error {
	if c.client == nil {
		return ErrCacheMiss
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("redis get %s: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("redis unmarshal %s: %w", key, err)
	}

	return nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("redis marshal %s: %w", key, err)
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 && c.defaultTTL > 0 {
		effectiveTTL = c.defaultTTL
	}

	if err := c.client.Set(ctx, key, data, effectiveTTL).Err(); err != nil {
		return fmt.Errorf("redis set %s: %w", key, err)
	}

	return nil
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	if c.client == nil || len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis del keys: %w", err)
	}

	return nil
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.client == nil {
		return false, nil
	}

	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists %s: %w", key, err)
	}

	return n > 0, nil
}

func (c *redisCache) Remember(ctx context.Context, key string, dest any, ttl time.Duration, fn func() (any, error)) error {
	err := c.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	// If it's anything other than miss, we log or continue to fallback
	val, err := fn()
	if err != nil {
		return err
	}

	// Cache the freshly computed/queried value
	_ = c.Set(ctx, key, val, ttl)

	// Populate dest
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

func (c *redisCache) GetInt(ctx context.Context, key string) (int64, error) {
	if c.client == nil {
		return 0, ErrCacheMiss
	}

	val, err := c.client.Get(ctx, key).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrCacheMiss
		}
		return 0, fmt.Errorf("redis get int %s: %w", key, err)
	}

	return val, nil
}

func (c *redisCache) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if c.client == nil {
		return nil
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 && c.defaultTTL > 0 {
		effectiveTTL = c.defaultTTL
	}

	if err := c.client.Set(ctx, key, value, effectiveTTL).Err(); err != nil {
		return fmt.Errorf("redis set int %s: %w", key, err)
	}

	return nil
}

func (c *redisCache) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if c.client == nil {
		return 0, nil
	}

	val, err := c.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("redis incrby %s by %d: %w", key, delta, err)
	}

	return val, nil
}

func (c *redisCache) DecrBy(ctx context.Context, key string, delta int64) (int64, error) {
	if c.client == nil {
		return 0, nil
	}

	val, err := c.client.DecrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("redis decrby %s by %d: %w", key, delta, err)
	}

	return val, nil
}

func (c *redisCache) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if c.client == nil {
		return true, nil
	}

	effectiveTTL := ttl
	if effectiveTTL <= 0 {
		effectiveTTL = 5 * time.Second
	}

	// SetNX sets key to "false" (locked) if it does not already exist
	ok, err := c.client.SetNX(ctx, key, "false", effectiveTTL).Result()
	if err != nil {
		return false, fmt.Errorf("redis acquire lock %s: %w", key, err)
	}

	return ok, nil
}

func (c *redisCache) ReleaseLock(ctx context.Context, key string) error {
	if c.client == nil {
		return nil
	}

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis release lock %s: %w", key, err)
	}

	return nil
}

func (c *redisCache) SetIntNX(ctx context.Context, key string, value int64, ttl time.Duration) (bool, error) {
	if c.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	ok, err := c.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx int %s: %w", key, err)
	}

	return ok, nil
}

var deductStockLua = redis.NewScript(`
local missing = {}
for i = 1, #KEYS do
    local exists = redis.call('EXISTS', KEYS[i])
    if exists == 0 then
        table.insert(missing, KEYS[i])
    end
end

if #missing > 0 then
    local res = { 1 }
    for i = 1, #missing do
        table.insert(res, missing[i])
    end
    return res
end

for i = 1, #KEYS do
    local val = redis.call('GET', KEYS[i])
    local current_stock = tonumber(val)
    local req_qty = tonumber(ARGV[i])
    if current_stock == nil or current_stock < req_qty then
        return { 2, KEYS[i] }
    end
end

for i = 1, #KEYS do
    local req_qty = tonumber(ARGV[i])
    redis.call('DECRBY', KEYS[i], req_qty)
end

return { 0 }
`)

func (c *redisCache) DeductStockAtomic(ctx context.Context, items []DeductStockItem) (*DeductStockResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if len(items) == 0 {
		return &DeductStockResult{Status: DeductStatusOK}, nil
	}

	keys := make([]string, len(items))
	args := make([]interface{}, len(items))
	for i, item := range items {
		keys[i] = item.Key
		args[i] = item.Quantity
	}

	rawRes, err := deductStockLua.Run(ctx, c.client, keys, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis lua deduct stock: %w", err)
	}

	slice, ok := rawRes.([]interface{})
	if !ok || len(slice) == 0 {
		return nil, fmt.Errorf("unexpected lua result format: %v", rawRes)
	}

	statusVal, ok := slice[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected status code in lua result: %v", slice[0])
	}

	switch statusVal {
	case 0:
		return &DeductStockResult{Status: DeductStatusOK}, nil
	case 1:
		missing := make([]string, 0, len(slice)-1)
		for _, v := range slice[1:] {
			if s, ok := v.(string); ok {
				missing = append(missing, s)
			}
		}
		return &DeductStockResult{
			Status:      DeductStatusMissing,
			MissingKeys: missing,
		}, nil
	case 2:
		failedKey := ""
		if len(slice) > 1 {
			if s, ok := slice[1].(string); ok {
				failedKey = s
			}
		}
		return &DeductStockResult{
			Status:    DeductStatusInsufficient,
			FailedKey: failedKey,
		}, nil
	default:
		return nil, fmt.Errorf("unknown lua status code: %d", statusVal)
	}
}

var restoreStockLua = redis.NewScript(`
for i = 1, #KEYS do
    local req_qty = tonumber(ARGV[i])
    if redis.call('EXISTS', KEYS[i]) == 1 then
        redis.call('INCRBY', KEYS[i], req_qty)
    end
end
return 0
`)

func (c *redisCache) RestoreStockAtomic(ctx context.Context, items []DeductStockItem) error {
	if c.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if len(items) == 0 {
		return nil
	}

	keys := make([]string, len(items))
	args := make([]interface{}, len(items))
	for i, item := range items {
		keys[i] = item.Key
		args[i] = item.Quantity
	}

	if err := restoreStockLua.Run(ctx, c.client, keys, args...).Err(); err != nil {
		return fmt.Errorf("redis lua restore stock: %w", err)
	}

	return nil
}
