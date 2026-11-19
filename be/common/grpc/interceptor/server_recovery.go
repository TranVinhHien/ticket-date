package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryServerRecovery catches any panic that occurs during unary RPC execution,
// logs the stack trace, and returns an Internal error to the client instead of crashing the server.
func UnaryServerRecovery(logger *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				if logger != nil {
					logger.Error().
						Str("method", info.FullMethod).
						Any("panic", r).
						Str("stack", stack).
						Msg("gRPC Server Panic Recovered")
				}
				err = status.Errorf(codes.Internal, "panic: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

// StreamServerRecovery catches any panic that occurs during streaming RPC execution.
func StreamServerRecovery(logger *zerolog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				if logger != nil {
					logger.Error().
						Str("method", info.FullMethod).
						Any("panic", r).
						Str("stack", stack).
						Msg("gRPC Stream Server Panic Recovered")
				}
				err = status.Errorf(codes.Internal, "panic: %v", fmt.Sprint(r))
			}
		}()

		return handler(srv, ss)
	}
}
