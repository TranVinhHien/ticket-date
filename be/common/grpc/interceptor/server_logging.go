package interceptor

import (
	"context"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// UnaryServerLogging returns a unary server interceptor that logs gRPC calls using Zerolog.
func UnaryServerLogging(logger *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if logger == nil {
			return handler(ctx, req)
		}

		start := time.Now()
		resp, err := handler(ctx, req)
		latency := time.Since(start)

		code := status.Code(err)
		event := getLogEvent(logger, code)

		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		reqID, _ := ctx.Value(constants.CtxKeyRequestID).(string)
		userID, _ := ctx.Value(constants.CtxKeyUserID).(string)

		event.
			Str("method", info.FullMethod).
			Str("code", code.String()).
			Uint32("code_num", uint32(code)).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("request_id", reqID).
			Str("user_id", userID)

		if err != nil {
			event.Err(err).Msg("gRPC Server Error")
		} else {
			event.Msg("gRPC Server Request")
		}

		return resp, err
	}
}

// StreamServerLogging returns a streaming server interceptor that logs gRPC calls using Zerolog.
func StreamServerLogging(logger *zerolog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if logger == nil {
			return handler(srv, ss)
		}

		start := time.Now()
		err := handler(srv, ss)
		latency := time.Since(start)

		ctx := ss.Context()
		code := status.Code(err)
		event := getLogEvent(logger, code)

		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		reqID, _ := ctx.Value(constants.CtxKeyRequestID).(string)
		userID, _ := ctx.Value(constants.CtxKeyUserID).(string)

		event.
			Str("method", info.FullMethod).
			Str("code", code.String()).
			Uint32("code_num", uint32(code)).
			Dur("latency", latency).
			Str("ip", clientIP).
			Str("request_id", reqID).
			Str("user_id", userID)

		if err != nil {
			event.Err(err).Msg("gRPC Stream Server Error")
		} else {
			event.Msg("gRPC Stream Server Request")
		}

		return err
	}
}

func getLogEvent(logger *zerolog.Logger, code codes.Code) *zerolog.Event {
	switch code {
	case codes.OK, codes.NotFound:
		return logger.Info()
	case codes.InvalidArgument, codes.Unauthenticated, codes.PermissionDenied, codes.AlreadyExists, codes.FailedPrecondition:
		return logger.Warn()
	default:
		return logger.Error()
	}
}
