package kafka

import (
	"strings"
	"time"
)

const (
	CompressionNone   = "none"
	CompressionGzip   = "gzip"
	CompressionSnappy = "snappy"
	CompressionLz4    = "lz4"
	CompressionZstd   = "zstd"

	DefaultBatchMaxBytes      int32         = 1048576 // 1 MiB
	DefaultLingerMs           int           = 10
	DefaultMaxBufferedRecords int           = 100000
	DefaultRequestTimeout     time.Duration = 10 * time.Second
	DefaultProducerRetries    int           = 5

	DefaultMaxRetries   int           = 3
	DefaultRetryBackoff time.Duration = 500 * time.Millisecond
	DefaultDLQSuffix    string        = ".dlq"
)

// SASLConfig holds optional SASL authentication settings.
type SASLConfig struct {
	Mechanism string `mapstructure:"mechanism" json:"mechanism"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	Username  string `mapstructure:"username" json:"username"`
	Password  string `mapstructure:"password" json:"password"`
}

// MetricsHook is an optional callback invoked after each produce or consume operation.
type MetricsHook func(topic string, eventType string, err error, duration time.Duration)

// ProducerConfig configures a shared Kafka producer instance.
type ProducerConfig struct {
	Brokers            []string      `mapstructure:"brokers" json:"brokers"`
	ClientID           string        `mapstructure:"client_id" json:"client_id"`
	Compression        string        `mapstructure:"compression" json:"compression"`
	BatchMaxBytes      int32         `mapstructure:"batch_max_bytes" json:"batch_max_bytes"`
	LingerMs           int           `mapstructure:"linger_ms" json:"linger_ms"`
	MaxBufferedRecords int           `mapstructure:"max_buffered_records" json:"max_buffered_records"`
	RequestTimeout     time.Duration `mapstructure:"request_timeout" json:"request_timeout"`
	Retries            int           `mapstructure:"retries" json:"retries"`
	DisableIdempotence bool          `mapstructure:"disable_idempotence" json:"disable_idempotence"`
	AllowAutoTopic     bool          `mapstructure:"allow_auto_topic" json:"allow_auto_topic"`
	SASL               *SASLConfig   `mapstructure:"sasl" json:"sasl"`
	MetricsHook        MetricsHook   `mapstructure:"-" json:"-"`
}

// WithDefaults returns a copy of ProducerConfig with sensible high-throughput defaults applied.
func (c ProducerConfig) WithDefaults() ProducerConfig {
	if c.Compression == "" {
		c.Compression = CompressionLz4
	} else {
		c.Compression = strings.ToLower(c.Compression)
	}
	if c.BatchMaxBytes <= 0 {
		c.BatchMaxBytes = DefaultBatchMaxBytes
	}
	if c.LingerMs < 0 {
		c.LingerMs = 0
	} else if c.LingerMs == 0 {
		c.LingerMs = DefaultLingerMs
	}
	if c.MaxBufferedRecords <= 0 {
		c.MaxBufferedRecords = DefaultMaxBufferedRecords
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = DefaultRequestTimeout
	}
	if c.Retries <= 0 {
		c.Retries = DefaultProducerRetries
	}
	return c
}

// ConsumerConfig configures a shared Kafka consumer group instance.
type ConsumerConfig struct {
	Brokers        []string      `mapstructure:"brokers" json:"brokers"`
	ClientID       string        `mapstructure:"client_id" json:"client_id"`
	GroupID        string        `mapstructure:"group_id" json:"group_id"`
	Topics         []string      `mapstructure:"topics" json:"topics"`
	StartFromEnd   bool          `mapstructure:"start_from_end" json:"start_from_end"`
	MaxRetries     int           `mapstructure:"max_retries" json:"max_retries"`
	RetryBackoff   time.Duration `mapstructure:"retry_backoff" json:"retry_backoff"`
	DLQEnabled     bool          `mapstructure:"dlq_enabled" json:"dlq_enabled"`
	DLQSuffix      string        `mapstructure:"dlq_suffix" json:"dlq_suffix"`
	SessionTimeout time.Duration `mapstructure:"session_timeout" json:"session_timeout"`
	SASL           *SASLConfig   `mapstructure:"sasl" json:"sasl"`
	MetricsHook    MetricsHook   `mapstructure:"-" json:"-"`
}

// WithDefaults returns a copy of ConsumerConfig with sensible defaults applied.
func (c ConsumerConfig) WithDefaults() ConsumerConfig {
	if c.MaxRetries <= 0 {
		c.MaxRetries = DefaultMaxRetries
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = DefaultRetryBackoff
	}
	if c.DLQSuffix == "" {
		c.DLQSuffix = DefaultDLQSuffix
	}
	if c.SessionTimeout <= 0 {
		c.SessionTimeout = 45 * time.Second
	}
	return c
}
