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
)

// HandlerFunc processes a decoded EventEnvelope.
// Returning an error triggers retries and (if configured) DLQ routing; returning nil commits the offset.
type HandlerFunc func(ctx context.Context, env *EventEnvelope) error

// Consumer defines the shared Kafka consumer contract for all microservices.
type Consumer interface {
	RegisterHandler(eventType string, handler HandlerFunc)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type partitionKey struct {
	topic     string
	partition int32
}

type partitionWorker struct {
	ch     chan *kgo.Record
	cancel context.CancelFunc
	done   chan struct{}
}

type franzConsumer struct {
	client      *kgo.Client
	dlqProducer Producer
	cfg         ConsumerConfig
	logger      zerolog.Logger
	metricsHook MetricsHook

	mu       sync.RWMutex
	handlers map[string]HandlerFunc

	workersMu sync.Mutex
	workers   map[partitionKey]*partitionWorker

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewConsumer creates a Kafka consumer group with per-partition ordered workers,
// exponential-backoff retries, optional DLQ publishing, and safe offset commits.
func NewConsumer(cfg ConsumerConfig, logger zerolog.Logger) (Consumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka consumer: brokers are required")
	}
	if cfg.GroupID == "" {
		return nil, errors.New("kafka consumer: group_id is required")
	}
	if len(cfg.Topics) == 0 {
		return nil, errors.New("kafka consumer: at least one topic is required")
	}
	cfg = cfg.WithDefaults()

	c := &franzConsumer{
		cfg:         cfg,
		logger:      logger,
		metricsHook: cfg.MetricsHook,
		handlers:    make(map[string]HandlerFunc),
		workers:     make(map[partitionKey]*partitionWorker),
	}

	if cfg.DLQEnabled {
		dlqProd, err := NewProducer(ProducerConfig{
			Brokers:  cfg.Brokers,
			ClientID: cfg.ClientID + "-dlq",
			SASL:     cfg.SASL,
		}, logger)
		if err != nil {
			return nil, fmt.Errorf("kafka consumer: init dlq producer: %w", err)
		}
		c.dlqProducer = dlqProd
	}

	resetOffset := kgo.NewOffset().AtStart()
	if cfg.StartFromEnd {
		resetOffset = kgo.NewOffset().AtEnd()
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.ConsumeResetOffset(resetOffset),
		kgo.Balancers(kgo.CooperativeStickyBalancer()),
		kgo.SessionTimeout(cfg.SessionTimeout),
		kgo.AutoCommitMarks(),
		kgo.OnPartitionsRevoked(c.onPartitionsRevoked),
		kgo.OnPartitionsLost(c.onPartitionsRevoked),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}
	if saslOpt, err := buildSASLOption(cfg.SASL); err != nil {
		return nil, err
	} else if saslOpt != nil {
		opts = append(opts, saslOpt)
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		if c.dlqProducer != nil {
			_ = c.dlqProducer.Close(context.Background())
		}
		return nil, fmt.Errorf("kafka consumer: init client: %w", err)
	}
	c.client = client
	return c, nil
}

func (c *franzConsumer) RegisterHandler(eventType string, handler HandlerFunc) {
	if eventType == "" || handler == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

func (c *franzConsumer) Start(ctx context.Context) error {
	c.mu.RLock()
	handlerCount := len(c.handlers)
	c.mu.RUnlock()
	if handlerCount == 0 {
		return errors.New("kafka consumer: no handlers registered")
	}

	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go c.pollLoop(runCtx)

	c.logger.Info().
		Str("group_id", c.cfg.GroupID).
		Strs("topics", c.cfg.Topics).
		Msg("kafka consumer started")
	return nil
}

func (c *franzConsumer) Stop(ctx context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	c.stopAllWorkers()

	if c.client != nil {
		if err := c.client.CommitMarkedOffsets(ctx); err != nil {
			c.logger.Error().Err(err).Msg("kafka consumer: commit marked offsets on stop failed")
		}
		c.client.Close()
	}
	if c.dlqProducer != nil {
		_ = c.dlqProducer.Close(ctx)
	}
	c.logger.Info().Str("group_id", c.cfg.GroupID).Msg("kafka consumer stopped")
	return nil
}

func (c *franzConsumer) pollLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				if errors.Is(fe.Err, context.Canceled) {
					return
				}
				c.logger.Error().
					Err(fe.Err).
					Str("topic", fe.Topic).
					Int32("partition", fe.Partition).
					Msg("kafka consumer fetch error")
			}
		}

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			if len(p.Records) == 0 {
				return
			}
			worker := c.getOrCreateWorker(ctx, partitionKey{
				topic:     p.Topic,
				partition: p.Partition,
			})
			for _, rec := range p.Records {
				select {
				case worker.ch <- rec:
				case <-ctx.Done():
					return
				}
			}
		})
	}
}

func (c *franzConsumer) getOrCreateWorker(ctx context.Context, pk partitionKey) *partitionWorker {
	c.workersMu.Lock()
	defer c.workersMu.Unlock()

	if w, ok := c.workers[pk]; ok {
		return w
	}

	wCtx, wCancel := context.WithCancel(ctx)
	w := &partitionWorker{
		ch:     make(chan *kgo.Record, 256),
		cancel: wCancel,
		done:   make(chan struct{}),
	}
	c.workers[pk] = w

	go func() {
		defer close(w.done)
		for {
			select {
			case <-wCtx.Done():
				return
			case rec, ok := <-w.ch:
				if !ok {
					return
				}
				c.handleRecordWithRetry(wCtx, rec)
			}
		}
	}()

	return w
}

func (c *franzConsumer) onPartitionsRevoked(ctx context.Context, cl *kgo.Client, revoked map[string][]int32) {
	c.workersMu.Lock()
	var toStop []*partitionWorker
	for topic, partitions := range revoked {
		for _, p := range partitions {
			pk := partitionKey{topic: topic, partition: p}
			if w, ok := c.workers[pk]; ok {
				toStop = append(toStop, w)
				delete(c.workers, pk)
			}
		}
	}
	c.workersMu.Unlock()

	for _, w := range toStop {
		w.cancel()
		<-w.done
	}

	if err := cl.CommitMarkedOffsets(ctx); err != nil {
		c.logger.Error().Err(err).Msg("kafka consumer: commit marked offsets on rebalance failed")
	}
}

func (c *franzConsumer) stopAllWorkers() {
	c.workersMu.Lock()
	toStop := make([]*partitionWorker, 0, len(c.workers))
	for pk, w := range c.workers {
		toStop = append(toStop, w)
		delete(c.workers, pk)
	}
	c.workersMu.Unlock()

	for _, w := range toStop {
		w.cancel()
		<-w.done
	}
}

func (c *franzConsumer) handleRecordWithRetry(ctx context.Context, rec *kgo.Record) {
	start := time.Now()
	env, handler, err := c.decodeAndResolve(rec)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("topic", rec.Topic).
			Int32("partition", rec.Partition).
			Int64("offset", rec.Offset).
			Msg("kafka consumer: non-retryable decode/routing error")
		c.sendToDLQAndCommit(ctx, rec, "", err)
		return
	}

	if env.TraceID != "" {
		ctx = context.WithValue(ctx, "request_id", env.TraceID)
	}

	var lastErr error
	backoff := c.cfg.RetryBackoff
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			backoff *= 2
		}

		lastErr = handler(ctx, env)
		if lastErr == nil {
			if c.metricsHook != nil {
				c.metricsHook(rec.Topic, env.EventType, nil, time.Since(start))
			}
			if c.client != nil {
				c.client.MarkCommitRecords(rec)
			}
			return
		}

		c.logger.Warn().
			Err(lastErr).
			Str("topic", rec.Topic).
			Str("event_type", env.EventType).
			Str("event_id", env.EventID).
			Int("attempt", attempt+1).
			Int("max_retries", c.cfg.MaxRetries).
			Msg("kafka consumer: handler failed, retrying")
	}

	if c.metricsHook != nil {
		c.metricsHook(rec.Topic, env.EventType, lastErr, time.Since(start))
	}
	c.sendToDLQAndCommit(ctx, rec, env.EventType, lastErr)
}

func (c *franzConsumer) decodeAndResolve(rec *kgo.Record) (*EventEnvelope, HandlerFunc, error) {
	var env EventEnvelope
	if err := json.Unmarshal(rec.Value, &env); err != nil {
		return nil, nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.EventType == "" {
		return nil, nil, errors.New("missing event_type in envelope")
	}

	c.mu.RLock()
	handler, ok := c.handlers[env.EventType]
	c.mu.RUnlock()
	if !ok {
		return &env, nil, fmt.Errorf("no handler registered for event_type=%q", env.EventType)
	}
	return &env, handler, nil
}

func (c *franzConsumer) sendToDLQAndCommit(ctx context.Context, rec *kgo.Record, eventType string, cause error) {
	if c.cfg.DLQEnabled && c.dlqProducer != nil {
		dlqTopic := rec.Topic + c.cfg.DLQSuffix
		dlqPayload := map[string]any{
			"original_topic":     rec.Topic,
			"original_partition": rec.Partition,
			"original_offset":    rec.Offset,
			"event_type":         eventType,
			"error":              cause.Error(),
			"failed_at":          time.Now().UTC(),
			"raw_value":          json.RawMessage(rec.Value),
		}
		if env, err := NewEnvelope("dlq."+eventType, c.cfg.GroupID, "", dlqPayload); err == nil {
			if pubErr := c.dlqProducer.PublishSync(ctx, dlqTopic, string(rec.Key), env); pubErr != nil {
				c.logger.Error().
					Err(pubErr).
					Str("dlq_topic", dlqTopic).
					Msg("kafka consumer: failed to publish to DLQ, skipping offset commit")
				return
			}
		}
	} else {
		c.logger.Error().
			Err(cause).
			Str("topic", rec.Topic).
			Int32("partition", rec.Partition).
			Int64("offset", rec.Offset).
			Msg("kafka consumer: retries exhausted and DLQ disabled, committing offset to prevent partition stall")
	}

	if c.client != nil {
		c.client.MarkCommitRecords(rec)
	}
}
