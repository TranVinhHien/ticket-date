package cron

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	commonKafka "github.com/TranVinhHien/ticket-date/be/common/kafka"
	"github.com/TranVinhHien/ticket-date/be/common/kafka/events"
	"github.com/TranVinhHien/ticket-date/be/inventory-service/internal/database/db"
)

type mockOutboxStore struct {
	events []db.OutboxEvent
}

func (m *mockOutboxStore) ClaimPendingOutboxEvents(ctx context.Context, limit int32) ([]db.OutboxEvent, error) {
	var claimed []db.OutboxEvent
	for i := range m.events {
		if m.events[i].Status == OutboxStatusPending && int32(len(claimed)) < limit {
			m.events[i].Status = OutboxStatusSending
			claimed = append(claimed, m.events[i])
		}
	}
	return claimed, nil
}

func (m *mockOutboxStore) MarkOutboxEventSuccess(ctx context.Context, id int64) (db.OutboxEvent, error) {
	for i := range m.events {
		if m.events[i].ID == id {
			m.events[i].Status = OutboxStatusSuccess
			now := time.Now()
			m.events[i].PublishedAt = &now
			return m.events[i], nil
		}
	}
	return db.OutboxEvent{}, errors.New("not found")
}

func (m *mockOutboxStore) MarkOutboxEventFailed(ctx context.Context, arg db.MarkOutboxEventFailedParams) (db.OutboxEvent, error) {
	for i := range m.events {
		if m.events[i].ID == arg.ID {
			m.events[i].RetryCount += arg.RetryCount
			m.events[i].Status = OutboxStatusFailed
			m.events[i].LastError = arg.LastError
			return m.events[i], nil
		}
	}
	return db.OutboxEvent{}, errors.New("not found")
}

func (m *mockOutboxStore) MarkOutboxEventRetryOrFailed(ctx context.Context, arg db.MarkOutboxEventRetryOrFailedParams) (db.OutboxEvent, error) {
	for i := range m.events {
		if m.events[i].ID == arg.ID {
			m.events[i].RetryCount++
			if m.events[i].RetryCount >= arg.RetryCount {
				m.events[i].Status = OutboxStatusFailed
			} else {
				m.events[i].Status = OutboxStatusPending
			}
			m.events[i].LastError = arg.LastError
			return m.events[i], nil
		}
	}
	return db.OutboxEvent{}, errors.New("not found")
}

func (m *mockOutboxStore) RecoverStaleSendingOutboxEvents(ctx context.Context, arg db.RecoverStaleSendingOutboxEventsParams) (int64, error) {
	return 0, nil
}

type mockKafkaProducer struct {
	err       error
	published []string
}

func (m *mockKafkaProducer) Publish(ctx context.Context, topic, key string, env *commonKafka.EventEnvelope) error {
	return m.err
}
func (m *mockKafkaProducer) PublishSync(ctx context.Context, topic, key string, env *commonKafka.EventEnvelope) error {
	return m.err
}
func (m *mockKafkaProducer) PublishRawSync(ctx context.Context, topic, key, eventID, eventType, source string, rawData json.RawMessage) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, eventID+":"+topic+":"+key+":"+source)
	return nil
}
func (m *mockKafkaProducer) Close(ctx context.Context) error { return nil }

func TestOutboxRelayJob_SuccessAndRetriesToFailed(t *testing.T) {
	store := &mockOutboxStore{
		events: []db.OutboxEvent{
			{ID: 1, AggregateID: "ord-1", EventType: events.EventInventoryReserved, Payload: `{"order_id":"ord-1"}`, Status: OutboxStatusPending},
			{ID: 2, AggregateID: "ord-2", EventType: events.EventInventoryReservationFailed, Payload: `{"order_id":"ord-2","message":"Sản phẩm đã hết hàng"}`, Status: OutboxStatusPending},
		},
	}

	prodOK := &mockKafkaProducer{}
	job := NewOutboxRelayJobWithStore(store, prodOK, time.Second, 100)
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected Run error: %v", err)
	}
	if store.events[0].Status != OutboxStatusSuccess || store.events[1].Status != OutboxStatusSuccess {
		t.Fatalf("expected status=2 (SUCCESS), got %d and %d", store.events[0].Status, store.events[1].Status)
	}
	if len(prodOK.published) != 2 {
		t.Fatalf("expected 2 published messages, got %d", len(prodOK.published))
	}
	if prodOK.published[0] != "outbox-1:"+events.EventInventoryReserved+":ord-1:inventory-service" {
		t.Fatalf("unexpected published[0]: %s", prodOK.published[0])
	}

	store.events = append(store.events, db.OutboxEvent{
		ID: 3, AggregateID: "ord-3", EventType: events.EventInventoryReserved, Payload: `{"order_id":"ord-3"}`, Status: OutboxStatusPending,
	})
	prodFail := &mockKafkaProducer{err: errors.New("broker unavailable")}
	jobFail := NewOutboxRelayJobWithStore(store, prodFail, time.Second, 100)
	jobFail.retryBackoff = 1 * time.Millisecond

	if err := jobFail.Run(context.Background()); err != nil {
		t.Fatalf("unexpected Run error: %v", err)
	}
	if store.events[2].Status != OutboxStatusFailed {
		t.Fatalf("expected status=3 (FAILED), got %d", store.events[2].Status)
	}
	if store.events[2].RetryCount != defaultOutboxMaxRetries {
		t.Fatalf("expected retry_count=%d, got %d", defaultOutboxMaxRetries, store.events[2].RetryCount)
	}
}
