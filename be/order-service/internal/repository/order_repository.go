package repository

import (
	"context"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	Queries() *db.Queries
	GetPool() *pgxpool.Pool
	ExecTx(ctx context.Context, fn func(q *db.Queries) error) error
}

type orderRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &orderRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (r *orderRepository) Queries() *db.Queries {
	return r.queries
}

func (r *orderRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

func (r *orderRepository) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
