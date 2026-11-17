package repository

import (
	"context"

	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepository interface {
	Queries() *db.Queries
	GetPool() *pgxpool.Pool
	ExecTx(ctx context.Context, fn func(q *db.Queries) error) error
}

type inventoryRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewInventoryRepository(pool *pgxpool.Pool) InventoryRepository {
	return &inventoryRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (r *inventoryRepository) Queries() *db.Queries {
	return r.queries
}

func (r *inventoryRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

func (r *inventoryRepository) ExecTx(ctx context.Context, fn func(q *db.Queries) error) error {
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
