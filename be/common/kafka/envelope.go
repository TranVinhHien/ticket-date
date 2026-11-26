package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultEnvelopeVersion = "v1"

	HeaderEventID   = "x-event-id"
	HeaderEventType = "x-event-type"
	HeaderSource    = "x-source-service"
	HeaderTraceID   = "x-trace-id"
)

// EventEnvelope is the standard JSON message wrapper shared across all services.
type EventEnvelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	Version    string          `json:"version"`
	Source     string          `json:"source"`
	OccurredAt time.Time       `json:"occurred_at"`
	TraceID    string          `json:"trace_id,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// NewEnvelope marshals data into an EventEnvelope.
func NewEnvelope(eventType, source, traceID string, data any) (*EventEnvelope, error) {
	if eventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope data: %w", err)
	}
	return NewRawEnvelope(uuid.NewString(), eventType, source, traceID, raw), nil
}

// NewRawEnvelope builds an EventEnvelope from an already-serialized JSON payload (e.g., from an outbox table).
func NewRawEnvelope(eventID, eventType, source, traceID string, rawData json.RawMessage) *EventEnvelope {
	if eventID == "" {
		eventID = uuid.NewString()
	}
	return &EventEnvelope{
		EventID:    eventID,
		EventType:  eventType,
		Version:    DefaultEnvelopeVersion,
		Source:     source,
		OccurredAt: time.Now().UTC(),
		TraceID:    traceID,
		Data:       rawData,
	}
}

// DecodeData unmarshals the envelope's Data field into the target type T.
func DecodeData[T any](env *EventEnvelope) (T, error) {
	var zero T
	if env == nil {
		return zero, fmt.Errorf("event envelope is nil")
	}
	if len(env.Data) == 0 {
		return zero, fmt.Errorf("event envelope data is empty for event_type=%q", env.EventType)
	}
	var out T
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return zero, fmt.Errorf("unmarshal event data for event_type=%q: %w", env.EventType, err)
	}
	return out, nil
}
