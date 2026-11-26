package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/repository"
)

const (
	OutboxStatusPending int16 = 0
	OutboxStatusSending int16 = 1
	OutboxStatusSuccess int16 = 2
	OutboxStatusFailed  int16 = 3

	defaultOutboxInterval   = 1 * time.Second
	defaultOutboxBatch      = int32(1000)
	defaultOutboxMaxRetries = int16(3)
	defaultRetryBackoff     = 100 * time.Millisecond
	defaultStaleSendingSec  = int64(120)
	outboxRelayJobName      = "outbox-relay"
	inventoryServiceSource  = "inventory-service"
)

// OutboxStore abstracts the outbox DB operations for testability.
type OutboxStore interface {
	ClaimPendingOutboxEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error)
	MarkOutboxEventSuccess(ctx context.Context, id int64) (db.OutboxEvent, error)
	MarkOutboxEventFailed(ctx context.Context, arg db.MarkOutboxEventFailedParams) (db.OutboxEvent, error)
	MarkOutboxEventRetryOrFailed(ctx context.Context, arg db.MarkOutboxEventRetryOrFailedParams) (db.OutboxEvent, error)
	RecoverStaleSendingOutboxEvents(ctx context.Context, arg db.RecoverStaleSendingOutboxEventsParams) (int64, error)
}

// OutboxRelayJob polls outbox_event (status = 0 PENDING), marks rows as SENDING (1),
// publishes to Kafka, and updates status to SUCCESS (2) or retries up to 3 times before FAILED (3).
type OutboxRelayJob struct {
	store           OutboxStore
	producer        commonKafka.Producer
	interval        time.Duration
	batchSize       int32
	maxRetries      int16
	retryBackoff    time.Duration
	staleSendingSec int64
}

// NewOutboxRelayJob creates an OutboxRelayJob backed by InventoryRepository.
func NewOutboxRelayJob(
	repo repository.InventoryRepository,
	producer commonKafka.Producer,
	interval time.Duration,
	batchSize int32,
) *OutboxRelayJob {
	var store OutboxStore
	if repo != nil && repo.Queries() != nil {
		store = repo.Queries()
	}
	return NewOutboxRelayJobWithStore(store, producer, interval, batchSize)
}

// NewOutboxRelayJobWithStore creates an OutboxRelayJob with an explicit OutboxStore (useful for unit tests).
func NewOutboxRelayJobWithStore(
	store OutboxStore,
	producer commonKafka.Producer,
	interval time.Duration,
	batchSize int32,
) *OutboxRelayJob {
	if interval <= 0 {
		interval = defaultOutboxInterval
	}
	if batchSize <= 0 {
		batchSize = defaultOutboxBatch
	}
	return &OutboxRelayJob{
		store:           store,
		producer:        producer,
		interval:        interval,
		batchSize:       batchSize,
		maxRetries:      defaultOutboxMaxRetries,
		retryBackoff:    defaultRetryBackoff,
		staleSendingSec: defaultStaleSendingSec,
	}
}

func (j *OutboxRelayJob) Name() string {
	return outboxRelayJobName
}

func (j *OutboxRelayJob) Interval() time.Duration {
	return j.interval
}

func (j *OutboxRelayJob) Run(ctx context.Context) error {
	if j.store == nil {
		return fmt.Errorf("outbox store is not initialized")
	}
	if j.producer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	if recovered, err := j.store.RecoverStaleSendingOutboxEvents(ctx, db.RecoverStaleSendingOutboxEventsParams{
		RetryCount: j.maxRetries,
		Seconds:    j.staleSendingSec,
	}); err != nil {
		log.Printf("[OutboxRelayJob] Warning: failed to recover stale SENDING rows: %v", err)
	} else if recovered > 0 {
		log.Printf("[OutboxRelayJob] Recovered %d stale SENDING outbox event(s)", recovered)
	}

	claimed, err := j.store.ClaimPendingOutboxEvents(ctx, j.batchSize)
	if err != nil {
		return fmt.Errorf("claim pending outbox events: %w", err)
	}
	if len(claimed) == 0 {
		return nil
	}

	var successCount, failCount int
	for _, ev := range claimed {
		eventID := fmt.Sprintf("outbox-%d", ev.ID)
		targetTopic := ev.EventType

		var pubErr error
		backoff := j.retryBackoff
		for attempt := int16(1); attempt <= j.maxRetries; attempt++ {
			if attempt > 1 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					backoff *= 2
				}
			}

			pubErr = j.producer.PublishRawSync(
				ctx,
				targetTopic,
				ev.AggregateID,
				eventID,
				ev.EventType,
				inventoryServiceSource,
				json.RawMessage(ev.Payload),
			)
			if pubErr == nil {
				break
			}

			log.Printf("[OutboxRelayJob] Attempt %d/%d failed to publish outbox id=%d event_type=%s aggregate_id=%s: %v",
				attempt, j.maxRetries, ev.ID, ev.EventType, ev.AggregateID, pubErr)
		}

		if pubErr != nil {
			failCount++
			errMsg := pubErr.Error()
			updated, markErr := j.store.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
				ID:         ev.ID,
				RetryCount: j.maxRetries,
				LastError:  &errMsg,
			})
			if markErr != nil {
				log.Printf("[OutboxRelayJob] Failed to mark FAILED for outbox id=%d: %v (publish err: %v)", ev.ID, markErr, pubErr)
				continue
			}
			log.Printf("[OutboxRelayJob] Publish exhausted %d retries, marked FAILED for outbox id=%d event_type=%s aggregate_id=%s retry_count=%d status=%d err=%v",
				j.maxRetries, ev.ID, ev.EventType, ev.AggregateID, updated.RetryCount, updated.Status, pubErr)
			continue
		}

		if _, markErr := j.store.MarkOutboxEventSuccess(ctx, ev.ID); markErr != nil {
			log.Printf("[OutboxRelayJob] Published outbox id=%d but failed to mark SUCCESS: %v", ev.ID, markErr)
			continue
		}
		successCount++
	}

	log.Printf("[OutboxRelayJob] Processed %d outbox event(s): success=%d failed=%d", len(claimed), successCount, failCount)
	return nil
}
