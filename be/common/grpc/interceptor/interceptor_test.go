package interceptor

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	grpcContext "github.com/TranVinhHien/ticket-date/be/common/grpc/context"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryServerRecovery(t *testing.T) {
	logger := zerolog.Nop()
	interceptor := UnaryServerRecovery(&logger)

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/PanicMethod"}
	panicHandler := func(ctx context.Context, req any) (any, error) {
		panic("something went completely wrong")
	}

	resp, err := interceptor(context.Background(), nil, info, panicHandler)
	if resp != nil {
		t.Fatalf("expected nil resp, got %v", resp)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Errorf("expected codes.Internal, got %v", st.Code())
	}
}

func TestUnaryServerTrace(t *testing.T) {
	interceptor := UnaryServerTrace()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/TraceMethod"}

	// Case 1: When RequestID already present in incoming metadata
	existingReqID := "custom-req-id-777"
	md := metadata.Pairs(grpcContext.MDKeyRequestID, existingReqID)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var capturedReqID string
	handler := func(c context.Context, req any) (any, error) {
		capturedReqID = c.Value(constants.CtxKeyRequestID).(string)
		return "ok", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReqID != existingReqID {
		t.Errorf("expected %s, got %s", existingReqID, capturedReqID)
	}

	// Case 2: When RequestID is missing, should auto-generate UUIDv4
	capturedReqID = ""
	_, err = interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uuid.Parse(capturedReqID); err != nil {
		t.Errorf("expected generated UUIDv4, got %s", capturedReqID)
	}
}

func TestUnaryServerAuth(t *testing.T) {
	interceptor := UnaryServerAuth()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/AuthMethod"}

	testUUID := uuid.New()
	md := metadata.Pairs(
		grpcContext.MDKeyUserID, testUUID.String(),
		grpcContext.MDKeyUserEmail, "user@ticket.vn",
		grpcContext.MDKeyUserRoles, "admin,organizer",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var (
		gotUserID  string
		gotEmail   string
		gotRoles   []string
		gotUserUID uuid.UUID
	)

	handler := func(c context.Context, req any) (any, error) {
		if v := c.Value(constants.CtxKeyUserID); v != nil {
			gotUserID = v.(string)
		}
		if v := c.Value("user_uuid"); v != nil {
			gotUserUID = v.(uuid.UUID)
		}
		if v := c.Value(constants.CtxKeyUserEmail); v != nil {
			gotEmail = v.(string)
		}
		if v := c.Value(constants.CtxKeyUserRoles); v != nil {
			gotRoles = v.([]string)
		}
		return "done", nil
	}

	_, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotUserID != testUUID.String() || gotUserUID != testUUID {
		t.Errorf("expected user id %s, got %s / %s", testUUID, gotUserID, gotUserUID)
	}
	if gotEmail != "user@ticket.vn" {
		t.Errorf("expected email user@ticket.vn, got %s", gotEmail)
	}
	if len(gotRoles) != 2 || gotRoles[0] != "admin" || gotRoles[1] != "organizer" {
		t.Errorf("expected [admin, organizer], got %v", gotRoles)
	}
}

func TestUnaryServerLogging(t *testing.T) {
	logger := zerolog.New(io.Discard)
	interceptor := UnaryServerLogging(&logger)
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/LogMethod"}

	// Success case
	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Error case
	_, err = interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "resource not found")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUnaryClientMetadata(t *testing.T) {
	interceptor := UnaryClientMetadata()

	ctx := context.Background()
	ctx = context.WithValue(ctx, constants.CtxKeyRequestID, "trace-client-1")
	ctx = context.WithValue(ctx, constants.CtxKeyUserID, "user-client-1")

	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			return errors.New("metadata missing")
		}
		if got := md.Get(grpcContext.MDKeyRequestID); len(got) == 0 || got[0] != "trace-client-1" {
			t.Errorf("expected trace-client-1, got %v", got)
		}
		if got := md.Get(grpcContext.MDKeyUserID); len(got) == 0 || got[0] != "user-client-1" {
			t.Errorf("expected user-client-1, got %v", got)
		}
		return nil
	}

	err := interceptor(ctx, "/test.Service/Method", nil, nil, nil, invoker)
	if err != nil {
		t.Fatalf("interceptor failed: %v", err)
	}
}

func TestDefaultChains(t *testing.T) {
	logger := zerolog.Nop()
	serverUnary := DefaultServerUnaryInterceptors(&logger)
	if len(serverUnary) != 4 {
		t.Errorf("expected 4 server unary interceptors, got %d", len(serverUnary))
	}

	serverStream := DefaultServerStreamInterceptors(&logger)
	if len(serverStream) != 4 {
		t.Errorf("expected 4 server stream interceptors, got %d", len(serverStream))
	}

	clientUnary := DefaultClientUnaryInterceptors(&logger)
	if len(clientUnary) != 2 {
		t.Errorf("expected 2 client unary interceptors, got %d", len(clientUnary))
	}

	clientStream := DefaultClientStreamInterceptors(&logger)
	if len(clientStream) != 2 {
		t.Errorf("expected 2 client stream interceptors, got %d", len(clientStream))
	}
}
