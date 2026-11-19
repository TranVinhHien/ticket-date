package interceptor

import (
	"context"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	grpcContext "github.com/TranVinhHien/ticket-date/be/common/grpc/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerTrace extracts the Request ID from incoming gRPC metadata.
// If missing, it generates a new UUIDv4 and injects it into both the Go context and outgoing metadata.
func UnaryServerTrace() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		reqID := grpcContext.GetRequestID(ctx)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, constants.CtxKeyRequestID, reqID)
		_ = grpc.SetHeader(ctx, metadata.Pairs(grpcContext.MDKeyRequestID, reqID))

		return handler(ctx, req)
	}
}

// StreamServerTrace extracts the Request ID from incoming metadata for streaming RPCs.
func StreamServerTrace() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		reqID := grpcContext.GetRequestID(ctx)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, constants.CtxKeyRequestID, reqID)
		_ = ss.SetHeader(metadata.Pairs(grpcContext.MDKeyRequestID, reqID))

		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}
		return handler(srv, wrapped)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
