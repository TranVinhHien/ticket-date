package interceptor

import (
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

// DefaultServerUnaryInterceptors returns the standard recommended chain of server unary interceptors:
// 1. Recovery (prevents server crashes on panics)
// 2. Trace (extracts or generates request_id)
// 3. Auth (extracts user_id, user_email, user_roles)
// 4. Logging (structured zerolog latency and status logging)
func DefaultServerUnaryInterceptors(logger *zerolog.Logger) []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		UnaryServerRecovery(logger),
		UnaryServerTrace(),
		UnaryServerAuth(),
		UnaryServerLogging(logger),
	}
}

// DefaultServerStreamInterceptors returns the standard recommended chain of server stream interceptors.
func DefaultServerStreamInterceptors(logger *zerolog.Logger) []grpc.StreamServerInterceptor {
	return []grpc.StreamServerInterceptor{
		StreamServerRecovery(logger),
		StreamServerTrace(),
		StreamServerAuth(),
		StreamServerLogging(logger),
	}
}

// DefaultClientUnaryInterceptors returns the standard recommended chain of client unary interceptors:
// 1. Metadata (propagates request_id, user identity from context)
// 2. Logging (logs target, latency, and status code)
func DefaultClientUnaryInterceptors(logger *zerolog.Logger) []grpc.UnaryClientInterceptor {
	return []grpc.UnaryClientInterceptor{
		UnaryClientMetadata(),
		UnaryClientLogging(logger),
	}
}

// DefaultClientStreamInterceptors returns the standard recommended chain of client stream interceptors.
func DefaultClientStreamInterceptors(logger *zerolog.Logger) []grpc.StreamClientInterceptor {
	return []grpc.StreamClientInterceptor{
		StreamClientMetadata(),
		StreamClientLogging(logger),
	}
}
