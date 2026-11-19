package interceptor

import (
	"context"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	grpcContext "github.com/TranVinhHien/ticket-date/be/common/grpc/context"
	"google.golang.org/grpc"
)

// UnaryServerAuth extracts authenticated user context from incoming gRPC metadata
// (X-User-ID, X-User-Email, X-User-Roles) and binds them into the Go context.
func UnaryServerAuth() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx = populateAuthContext(ctx)
		return handler(ctx, req)
	}
}

// StreamServerAuth extracts authenticated user context for streaming RPCs.
func StreamServerAuth() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := populateAuthContext(ss.Context())
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}
		return handler(srv, wrapped)
	}
}

func populateAuthContext(ctx context.Context) context.Context {
	// User ID
	if uid, ok := grpcContext.GetUserID(ctx); ok {
		ctx = context.WithValue(ctx, constants.CtxKeyUserID, uid.String())
		ctx = context.WithValue(ctx, "user_uuid", uid)
	}

	// Email
	if email := grpcContext.GetUserEmail(ctx); email != "" {
		ctx = context.WithValue(ctx, constants.CtxKeyUserEmail, email)
	}

	// Roles
	if roles := grpcContext.GetUserRoles(ctx); len(roles) > 0 {
		ctx = context.WithValue(ctx, constants.CtxKeyUserRoles, roles)
	}

	return ctx
}
