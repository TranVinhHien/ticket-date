package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func Connect() {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.App.CockroachDB.User,
		config.App.CockroachDB.Password,
		config.App.CockroachDB.Host,
		config.App.CockroachDB.Port,
		config.App.CockroachDB.DBName,
		config.App.CockroachDB.SSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	maxOpen := config.App.CockroachDB.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 25
	}
	maxIdle := config.App.CockroachDB.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5
	}
	lifetime := config.App.CockroachDB.ConnMaxLifetime
	if lifetime == 0 {
		lifetime = time.Hour
	}

	poolConfig.MaxConns = int32(maxOpen)
	poolConfig.MinConns = int32(maxIdle)
	poolConfig.MaxConnLifetime = lifetime
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	log.Println("Database connected successfully (pgxpool)")
}

func GetPool() *pgxpool.Pool {
	return pool
}

func Close() {
	if pool != nil {
		pool.Close()
		log.Println("Database connection pool closed")
	}
}
