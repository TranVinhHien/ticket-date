package client

import (
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Default connection parameters
const (
	DefaultConnectTimeout   = 5 * time.Second
	DefaultKeepAliveTime    = 30 * time.Second
	DefaultKeepAliveTimeout = 5 * time.Second
)

// ClientOptions configures the gRPC client connection.
type ClientOptions struct {
	Timeout            time.Duration
	Logger             *zerolog.Logger
	Insecure           bool
	Creds              credentials.TransportCredentials
	UnaryInterceptors  []grpc.UnaryClientInterceptor
	StreamInterceptors []grpc.StreamClientInterceptor
	DialOptions        []grpc.DialOption
	KeepAliveTime      time.Duration
	KeepAliveTimeout   time.Duration
}

// Option is a functional option for configuring ClientOptions.
type Option func(*ClientOptions)

// WithTimeout sets a default connection timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *ClientOptions) {
		o.Timeout = d
	}
}

// WithLogger sets the Zerolog logger for client interceptors.
func WithLogger(logger *zerolog.Logger) Option {
	return func(o *ClientOptions) {
		o.Logger = logger
	}
}

// WithInsecure specifies that the connection should use insecure transport credentials.
func WithInsecure() Option {
	return func(o *ClientOptions) {
		o.Insecure = true
		o.Creds = nil
	}
}

// WithTransportCredentials sets custom transport credentials (e.g., TLS).
func WithTransportCredentials(creds credentials.TransportCredentials) Option {
	return func(o *ClientOptions) {
		o.Creds = creds
		o.Insecure = false
	}
}

// WithUnaryInterceptors appends custom unary client interceptors.
func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(o *ClientOptions) {
		o.UnaryInterceptors = append(o.UnaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors appends custom streaming client interceptors.
func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) Option {
	return func(o *ClientOptions) {
		o.StreamInterceptors = append(o.StreamInterceptors, interceptors...)
	}
}

// WithDialOptions appends raw gRPC dial options.
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *ClientOptions) {
		o.DialOptions = append(o.DialOptions, opts...)
	}
}

// WithKeepAlive sets keepalive probe intervals.
func WithKeepAlive(time, timeout time.Duration) Option {
	return func(o *ClientOptions) {
		o.KeepAliveTime = time
		o.KeepAliveTimeout = timeout
	}
}
