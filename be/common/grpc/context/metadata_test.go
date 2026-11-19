package grpcContext

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/TranVinhHien/ticket-date/be/common/constants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestFromGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testReqID := "test-request-id-123"
	testUserID := uuid.New().String()
	testEmail := "test@example.com"
	testRoles := []string{"admin", "user"}

	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set(constants.CtxKeyRequestID, testReqID)
	c.Set(constants.CtxKeyUserID, testUserID)
	c.Set(constants.CtxKeyUserEmail, testEmail)
	c.Set(constants.CtxKeyUserRoles, testRoles)

	grpcCtx := FromGinContext(c)
	md, ok := metadata.FromOutgoingContext(grpcCtx)
	if !ok {
		t.Fatal("expected outgoing metadata in context")
	}

	if got := md.Get(MDKeyRequestID); len(got) == 0 || got[0] != testReqID {
		t.Errorf("expected request_id %s, got %v", testReqID, got)
	}
	if got := md.Get(MDKeyUserID); len(got) == 0 || got[0] != testUserID {
		t.Errorf("expected user_id %s, got %v", testUserID, got)
	}
	if got := md.Get(MDKeyUserEmail); len(got) == 0 || got[0] != testEmail {
		t.Errorf("expected user_email %s, got %v", testEmail, got)
	}
	if got := md.Get(MDKeyUserRoles); len(got) == 0 || got[0] != "admin,user" {
		t.Errorf("expected user_roles admin,user, got %v", got)
	}
}

func TestWithOutgoingMetadata(t *testing.T) {
	ctx := context.Background()
	ctx = WithOutgoingMetadata(ctx, "custom-key", "custom-value")

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}

	if got := md.Get("custom-key"); len(got) == 0 || got[0] != "custom-value" {
		t.Errorf("expected custom-value, got %v", got)
	}
}

func TestToOutgoingContext(t *testing.T) {
	ctx := context.Background()
	testUUID := uuid.New()
	ctx = context.WithValue(ctx, constants.CtxKeyRequestID, "req-999")
	ctx = context.WithValue(ctx, constants.CtxKeyUserID, testUUID)
	ctx = context.WithValue(ctx, constants.CtxKeyUserRoles, []string{"user", "manager"})

	outgoingCtx := ToOutgoingContext(ctx, constants.CtxKeyRequestID, constants.CtxKeyUserID, constants.CtxKeyUserRoles)
	md, ok := metadata.FromOutgoingContext(outgoingCtx)
	if !ok {
		t.Fatal("expected metadata in outgoing context")
	}

	if got := md.Get(constants.CtxKeyRequestID); len(got) == 0 || got[0] != "req-999" {
		t.Errorf("expected req-999, got %v", got)
	}
	if got := md.Get(constants.CtxKeyUserID); len(got) == 0 || got[0] != testUUID.String() {
		t.Errorf("expected %s, got %v", testUUID.String(), got)
	}
	if got := md.Get(constants.CtxKeyUserRoles); len(got) == 0 || got[0] != "user,manager" {
		t.Errorf("expected user,manager, got %v", got)
	}
}

func TestGettersFromIncomingContext(t *testing.T) {
	testUUID := uuid.New()
	md := metadata.Pairs(
		MDKeyRequestID, "trace-abc-123",
		MDKeyUserID, testUUID.String(),
		MDKeyUserEmail, "alice@test.com",
		MDKeyUserRoles, "user,editor",
	)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	if got := GetRequestID(ctx); got != "trace-abc-123" {
		t.Errorf("GetRequestID: expected trace-abc-123, got %s", got)
	}

	uid, ok := GetUserID(ctx)
	if !ok || uid != testUUID {
		t.Errorf("GetUserID: expected %s, got %s (ok=%v)", testUUID, uid, ok)
	}

	if got := GetUserEmail(ctx); got != "alice@test.com" {
		t.Errorf("GetUserEmail: expected alice@test.com, got %s", got)
	}

	roles := GetUserRoles(ctx)
	if len(roles) != 2 || roles[0] != "user" || roles[1] != "editor" {
		t.Errorf("GetUserRoles: expected [user, editor], got %v", roles)
	}
}
