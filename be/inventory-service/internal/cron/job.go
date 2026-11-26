package cron

import (
	"context"
	"time"
)

// Job represents a periodic background task managed by Manager.
type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}
