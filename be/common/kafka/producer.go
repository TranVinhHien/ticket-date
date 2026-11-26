package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Producer defines the shared Kafka producer contract for all microservices.
type Producer interface {
	// Publish enqueues an EventEnvelope asynchronously using franz-go's batching buffer.
	Publish(ctx context.Context, topic string, key string, env *EventEnvelope) error
	// PublishSync publishes an EventEnvelope and blocks until broker acknowledgement.
	PublishSync(ctx context.Context, topic string, key string, env *EventEnvelope) error
	// PublishRawSync wraps a pre-serialized JSON payload (e.g. from outbox_event) into an EventEnvelope and publishes synchronously.
	PublishRawSync(ctx context.Context, topic string, key string, eventID string, eventType string, source string, rawData json.RawMessage) error
	// Close flushes buffered records and closes the underlying Kafka client.
	Close(ctx context.Context) error
}

type franzProducer struct {
	client      *kgo.Client
	cfg         ProducerConfig
	logger      zerolog.Logger
	metricsHook MetricsHook
	closeOnce   sync.Once
}

// NewProducer creates a high-throughput, idempotent Kafka producer backed by twmb/franz-go.
func NewProducer(cfg ProducerConfig, logger zerolog.Logger) (Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka producer: brokers are required")
	}
	cfg = cfg.WithDefaults()

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ProducerBatchCompression(parseCompression(cfg.Compression)),
		kgo.ProducerLinger(time.Duration(cfg.LingerMs) * time.Millisecond),
		kgo.ProducerBatchMaxBytes(cfg.BatchMaxBytes),
		kgo.MaxBufferedRecords(cfg.MaxBufferedRecords),
		kgo.ProduceRequestTimeout(cfg.RequestTimeout),
		kgo.RecordRetries(cfg.Retries),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}
	if cfg.DisableIdempotence {
		opts = append(opts, kgo.DisableIdempotentWrite())
	}
	if cfg.AllowAutoTopic {
		opts = append(opts, kgo.AllowAutoTopicCreation())
	}
	if saslOpt, err := buildSASLOption(cfg.SASL); err != nil {
		return nil, err
	} else if saslOpt != nil {
		opts = append(opts, saslOpt)
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: init client: %w", err)
	}

	return &franzProducer{
		client:      client,
		cfg:         cfg,
		logger:      logger,
		metricsHook: cfg.MetricsHook,
	}, nil
}

func (p *franzProducer) Publish(ctx context.Context, topic string, key string, env *EventEnvelope) error {
	rec, err := buildRecord(topic, key, env)
	if err != nil {
		return err
	}

	start := time.Now()
	p.client.Produce(ctx, rec, func(r *kgo.Record, produceErr error) {
		dur := time.Since(start)
		if p.metricsHook != nil {
			p.metricsHook(topic, env.EventType, produceErr, dur)
		}
		if produceErr != nil {
			p.logger.Error().
				Err(produceErr).
				Str("topic", topic).
				Str("event_type", env.EventType).
				Str("event_id", env.EventID).
				Msg("kafka async produce failed")
		}
	})
	return nil
}

func (p *franzProducer) PublishSync(ctx context.Context, topic string, key string, env *EventEnvelope) error {
	rec, err := buildRecord(topic, key, env)
	if err != nil {
		return err
	}

	start := time.Now()
	results := p.client.ProduceSync(ctx, rec)
	dur := time.Since(start)
	err = results.FirstErr()
	if p.metricsHook != nil {
		p.metricsHook(topic, env.EventType, err, dur)
	}
	if err != nil {
		return fmt.Errorf("kafka sync produce topic=%s event_type=%s: %w", topic, env.EventType, err)
	}
	return nil
}

func (p *franzProducer) PublishRawSync(
	ctx context.Context,
	topic string,
	key string,
	eventID string,
	eventType string,
	source string,
	rawData json.RawMessage,
) error {
	if eventType == "" {
		return errors.New("kafka producer: event_type is required")
	}
	traceID, _ := ctx.Value("request_id").(string)
	env := NewRawEnvelope(eventID, eventType, source, traceID, rawData)
	return p.PublishSync(ctx, topic, key, env)
}

func (p *franzProducer) Close(ctx context.Context) error {
	var flushErr error
	p.closeOnce.Do(func() {
		flushErr = p.client.Flush(ctx)
		p.client.Close()
	})
	return flushErr
}

func buildRecord(topic string, key string, env *EventEnvelope) (*kgo.Record, error) {
	if topic == "" {
		return nil, errors.New("kafka producer: topic is required")
	}
	if env == nil {
		return nil, errors.New("kafka producer: envelope is required")
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: marshal envelope: %w", err)
	}

	headers := []kgo.RecordHeader{
		{Key: HeaderEventID, Value: []byte(env.EventID)},
		{Key: HeaderEventType, Value: []byte(env.EventType)},
		{Key: HeaderSource, Value: []byte(env.Source)},
	}
	if env.TraceID != "" {
		headers = append(headers, kgo.RecordHeader{Key: HeaderTraceID, Value: []byte(env.TraceID)})
	}

	rec := &kgo.Record{
		Topic:   topic,
		Value:   payload,
		Headers: headers,
	}
	if key != "" {
		rec.Key = []byte(key)
	}
	return rec, nil
}

func parseCompression(c string) kgo.CompressionCodec {
	switch c {
	case CompressionGzip:
		return kgo.GzipCompression()
	case CompressionSnappy:
		return kgo.SnappyCompression()
	case CompressionZstd:
		return kgo.ZstdCompression()
	case CompressionNone:
		return kgo.NoCompression()
	default:
		return kgo.Lz4Compression()
	}
}

func buildSASLOption(cfg *SASLConfig) (kgo.Opt, error) {
	if cfg == nil || cfg.Username == "" || cfg.Password == "" {
		return nil, nil
	}
	switch cfg.Mechanism {
	case "", "PLAIN":
		return kgo.SASL(plain.Auth{
			User: cfg.Username,
			Pass: cfg.Password,
		}.AsMechanism()), nil
	case "SCRAM-SHA-256":
		return kgo.SASL(scram.Auth{
			User: cfg.Username,
			Pass: cfg.Password,
		}.AsSha256Mechanism()), nil
	case "SCRAM-SHA-512":
		return kgo.SASL(scram.Auth{
			User: cfg.Username,
			Pass: cfg.Password,
		}.AsSha512Mechanism()), nil
	default:
		return nil, fmt.Errorf("unsupported SASL mechanism: %s", cfg.Mechanism)
	}
}
