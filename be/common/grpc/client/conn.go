package client

import (
	"fmt"

	"github.com/TranVinhHien/ticket-date/be/common/grpc/interceptor"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// NewClientConn creates a standardized, production-ready *grpc.ClientConn
// with built-in metadata propagation, keepalive, and structured logging.
func NewClientConn(target string, opts ...Option) (*grpc.ClientConn, error) {
	if target == "" {
		return nil, fmt.Errorf("gRPC target address cannot be empty")
	}

	options := ClientOptions{
		Timeout:          DefaultConnectTimeout,
		Insecure:         true,
		KeepAliveTime:    DefaultKeepAliveTime,
		KeepAliveTimeout: DefaultKeepAliveTimeout,
	}

	for _, opt := range opts {
		opt(&options)
	}

	dialOpts := make([]grpc.DialOption, 0)

	// 1. Credentials
	if options.Creds != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(options.Creds))
	} else if options.Insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// 2. KeepAlive settings
	if options.KeepAliveTime > 0 {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                options.KeepAliveTime,
			Timeout:             options.KeepAliveTimeout,
			PermitWithoutStream: true,
		}))
	}

	// 3. Unary Interceptors Chain (built-ins first, then user-supplied)
	unaryChain := make([]grpc.UnaryClientInterceptor, 0)
	unaryChain = append(unaryChain, interceptor.DefaultClientUnaryInterceptors(options.Logger)...)
	unaryChain = append(unaryChain, options.UnaryInterceptors...)
	dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(unaryChain...))

	// 4. Stream Interceptors Chain
	streamChain := make([]grpc.StreamClientInterceptor, 0)
	streamChain = append(streamChain, interceptor.DefaultClientStreamInterceptors(options.Logger)...)
	streamChain = append(streamChain, options.StreamInterceptors...)
	dialOpts = append(dialOpts, grpc.WithChainStreamInterceptor(streamChain...))

	// 5. Custom dial options
	dialOpts = append(dialOpts, options.DialOptions...)

	// Use modern grpc.NewClient
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", target, err)
	}

	return conn, nil
}
