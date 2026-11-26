package cron

import (
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/order-service/internal/repository"
)

// NewConfiguredManager builds a cron Manager and registers all order-service cron jobs.
// Add new Job registrations here as new background jobs are introduced.
func NewConfiguredManager(
	cfg config.CronConfig,
	repo repository.OrderRepository,
) (*Manager, error) {
	manager := NewManager()

	if cfg.PendingOrders.Enabled {
		if err := manager.Register(
			NewPendingOrdersJob(
				repo,
				cfg.PendingOrders.Interval,
				cfg.PendingOrders.Batch,
			),
		); err != nil {
			return nil, err
		}
	}

	return manager, nil
}
