package cron

import (
	"context"
	"time"
)

// Job defines a scheduled background task managed by the cron Manager.
// Implement this interface to add new cron jobs to order-service.
type Job interface {
	// Name returns a unique identifier for the job (used in logs).
	Name() string
	// Interval returns how frequently the job should run.
	Interval() time.Duration
	// Run executes one cycle of the job.
	Run(ctx context.Context) error
}
