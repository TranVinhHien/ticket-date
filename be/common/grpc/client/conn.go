package client

import (
	"context"
	"fmt"
	"sync/atomic"

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

	// 2. KeepAlive & HTTP/2 flow-control settings
	if options.KeepAliveTime > 0 {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                options.KeepAliveTime,
			Timeout:             options.KeepAliveTimeout,
			PermitWithoutStream: true,
		}))
	}
	dialOpts = append(dialOpts,
		grpc.WithInitialWindowSize(1<<20),     // 1 MiB per stream
		grpc.WithInitialConnWindowSize(1<<21), // 2 MiB per connection
	)

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

// ClientPool distributes RPCs round-robin across multiple *grpc.ClientConn instances
// to avoid single-connection HTTP/2 stream limits and writer contention under high load.
type ClientPool struct {
	conns []*grpc.ClientConn
	next  atomic.Uint64
}

// NewClientPool creates a pool of size gRPC connections to target.
func NewClientPool(target string, size int, opts ...Option) (*ClientPool, error) {
	if size <= 1 {
		size = 1
	}
	conns := make([]*grpc.ClientConn, 0, size)
	for i := 0; i < size; i++ {
		conn, err := NewClientConn(target, opts...)
		if err != nil {
			for _, c := range conns {
				_ = c.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
	}
	return &ClientPool{conns: conns}, nil
}

func (p *ClientPool) pick() *grpc.ClientConn {
	idx := p.next.Add(1) - 1
	return p.conns[idx%uint64(len(p.conns))]
}

// Invoke implements grpc.ClientConnInterface.
func (p *ClientPool) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	return p.pick().Invoke(ctx, method, args, reply, opts...)
}

// NewStream implements grpc.ClientConnInterface.
func (p *ClientPool) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return p.pick().NewStream(ctx, desc, method, opts...)
}

// Close closes all underlying gRPC connections in the pool.
func (p *ClientPool) Close() error {
	var firstErr error
	for _, c := range p.conns {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
