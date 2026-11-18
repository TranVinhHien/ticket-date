package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/config"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

// ConnectRedis initializes and validates the Redis connection pool.
func ConnectRedis() {
	addr := fmt.Sprintf("%s:%d", config.App.Redis.Host, config.App.Redis.Port)
	if config.App.Redis.Host == "" {
		addr = "localhost:6379"
	}

	poolSize := config.App.Redis.PoolSize
	if poolSize <= 0 {
		poolSize = 20
	}

	dialTimeout := config.App.Redis.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}

	readTimeout := config.App.Redis.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 3 * time.Second
	}

	writeTimeout := config.App.Redis.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 3 * time.Second
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     config.App.Redis.Password,
		DB:           config.App.Redis.DB,
		PoolSize:     poolSize,
		MinIdleConns: config.App.Redis.MinIdleConns,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Unable to connect to Redis at %s: %v", addr, err)
		return
	}

	log.Printf("Redis connected successfully to %s (DB: %d, PoolSize: %d)", addr, config.App.Redis.DB, poolSize)
}

// GetRedisClient returns the global Redis client instance.
func GetRedisClient() *redis.Client {
	return rdb
}

// PingRedis checks the connection health of Redis.
func PingRedis(ctx context.Context) error {
	if rdb == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return rdb.Ping(ctx).Err()
}

// CloseRedis gracefully closes the Redis client connection pool.
func CloseRedis() {
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		} else {
			log.Println("Redis connection pool closed")
		}
	}
}
