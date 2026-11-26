package cron

import (
	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/config"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
)

// NewConfiguredManager builds a cron Manager and registers all inventory-service cron jobs.
func NewConfiguredManager(
	cfg config.CronConfig,
	repo repository.InventoryRepository,
	producer commonKafka.Producer,
) (*Manager, error) {
	manager := NewManager()

	if cfg.OutboxRelay.Enabled && producer != nil {
		if err := manager.Register(
			NewOutboxRelayJob(
				repo,
				producer,
				cfg.OutboxRelay.Interval,
				cfg.OutboxRelay.Batch,
			),
		); err != nil {
			return nil, err
		}
	}

	return manager, nil
}
