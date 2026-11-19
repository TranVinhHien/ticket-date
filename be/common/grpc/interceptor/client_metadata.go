package interceptor

import (
	"context"

	grpcContext "github.com/TranVinhHien/ticket-date/be/common/grpc/context"
	"google.golang.org/grpc"
)

// UnaryClientMetadata ensures tracking headers (Request ID, User ID, Email, Roles)
// are automatically propagated to outgoing gRPC calls.
func UnaryClientMetadata() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = propagateMetadata(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientMetadata propagates metadata for streaming RPC calls.
func StreamClientMetadata() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = propagateMetadata(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func propagateMetadata(ctx context.Context) context.Context {
	return grpcContext.PropagateContextToOutgoing(ctx)
}
