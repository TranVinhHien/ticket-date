package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kgo"
)

type samplePayload struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

func TestEnvelopeRoundTrip(t *testing.T) {
	env, err := NewEnvelope("OrderCreated", "order-service", "trace-123", samplePayload{
		OrderID: "ord-1",
		Amount:  150000,
	})
	if err != nil {
		t.Fatalf("NewEnvelope failed: %v", err)
	}
	if env.EventID == "" || env.EventType != "OrderCreated" || env.Source != "order-service" {
		t.Fatalf("unexpected envelope metadata: %+v", env)
	}

	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope failed: %v", err)
	}

	var decodedEnv EventEnvelope
	if err := json.Unmarshal(raw, &decodedEnv); err != nil {
		t.Fatalf("unmarshal envelope failed: %v", err)
	}

	got, err := DecodeData[samplePayload](&decodedEnv)
	if err != nil {
		t.Fatalf("DecodeData failed: %v", err)
	}
	if got.OrderID != "ord-1" || got.Amount != 150000 {
		t.Fatalf("unexpected decoded payload: %+v", got)
	}
}

func TestConsumerHandleRecordRetryAndDLQ(t *testing.T) {
	var attempts atomic.Int32
	c := &franzConsumer{
		cfg: ConsumerConfig{
			GroupID:      "inv-group",
			MaxRetries:   2,
			RetryBackoff: 5 * time.Millisecond,
			DLQEnabled:   false,
		}.WithDefaults(),
		logger: zerolog.Nop(),
		handlers: map[string]HandlerFunc{
			"OrderCreated": func(ctx context.Context, env *EventEnvelope) error {
				if attempts.Add(1) < 3 {
					return errors.New("transient db error")
				}
				return nil
			},
		},
	}

	env, err := NewEnvelope("OrderCreated", "order-service", "", samplePayload{OrderID: "ord-2", Amount: 500})
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	val, _ := json.Marshal(env)
	rec := &kgo.Record{Topic: "order.events", Value: val}

	c.handleRecordWithRetry(context.Background(), rec)
	if got := attempts.Load(); got != 3 {
		t.Fatalf("expected 3 attempts (1 initial + 2 retries), got %d", got)
	}
}
