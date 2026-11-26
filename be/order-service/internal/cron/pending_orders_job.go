package cron

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TranVinhHien/ticket-date/be/order-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/repository"
)

const (
	OrderStatusPending       = "PENDING"
	defaultPendingInterval   = 1 * time.Minute
	defaultPendingOrderBatch = int32(100)
	pendingOrdersJobName     = "pending-orders"
)

// PendingOrdersJob fetches orders currently in PENDING status for downstream processing.
type PendingOrdersJob struct {
	repo      repository.OrderRepository
	interval  time.Duration
	batchSize int32
}

// NewPendingOrdersJob creates a new PendingOrdersJob with sensible defaults.
func NewPendingOrdersJob(repo repository.OrderRepository, interval time.Duration, batchSize int32) *PendingOrdersJob {
	if interval <= 0 {
		interval = defaultPendingInterval
	}
	if batchSize <= 0 {
		batchSize = defaultPendingOrderBatch
	}
	return &PendingOrdersJob{
		repo:      repo,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (j *PendingOrdersJob) Name() string {
	return pendingOrdersJobName
}

func (j *PendingOrdersJob) Interval() time.Duration {
	return j.interval
}

func (j *PendingOrdersJob) Run(ctx context.Context) error {
	if j.repo == nil || j.repo.Queries() == nil {
		return fmt.Errorf("order repository is not initialized")
	}

	orders, err := j.repo.Queries().ListOrdersByStatus(ctx, db.ListOrdersByStatusParams{
		Status: OrderStatusPending,
		Limit:  j.batchSize,
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to list pending orders: %w", err)
	}

	log.Printf("[PendingOrdersJob] Fetched %d pending order(s)", len(orders))
	return nil
}
