package interceptor

import (
	"context"
	"time"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryClientLogging returns a unary client interceptor that logs outbound gRPC calls with Zerolog.
func UnaryClientLogging(logger *zerolog.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if logger == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		latency := time.Since(start)

		code := status.Code(err)
		event := getLogEvent(logger, code)

		reqID, _ := ctx.Value(constants.CtxKeyRequestID).(string)
		userID, _ := ctx.Value(constants.CtxKeyUserID).(string)

		event.
			Str("target", cc.Target()).
			Str("method", method).
			Str("code", code.String()).
			Uint32("code_num", uint32(code)).
			Dur("latency", latency).
			Str("request_id", reqID).
			Str("user_id", userID)

		if err != nil {
			event.Err(err).Msg("gRPC Client Call Error")
		} else {
			event.Msg("gRPC Client Call Success")
		}

		return err
	}
}

// StreamClientLogging returns a streaming client interceptor that logs outbound calls.
func StreamClientLogging(logger *zerolog.Logger) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if logger == nil {
			return streamer(ctx, desc, cc, method, opts...)
		}

		start := time.Now()
		cs, err := streamer(ctx, desc, cc, method, opts...)
		latency := time.Since(start)

		code := status.Code(err)
		event := getLogEvent(logger, code)

		reqID, _ := ctx.Value(constants.CtxKeyRequestID).(string)

		event.
			Str("target", cc.Target()).
			Str("method", method).
			Str("code", code.String()).
			Dur("init_latency", latency).
			Str("request_id", reqID)

		if err != nil {
			event.Err(err).Msg("gRPC Stream Client Init Error")
		} else {
			event.Msg("gRPC Stream Client Init Success")
		}

		return cs, err
	}
}
