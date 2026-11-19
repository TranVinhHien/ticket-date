package grpcErrors

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		err          error
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			err:          InvalidArgument("missing field %s", "name"),
			expectedCode: codes.InvalidArgument,
			expectedMsg:  "missing field name",
		},
		{
			err:          NotFound("Order", "ord-123"),
			expectedCode: codes.NotFound,
			expectedMsg:  "Order 'ord-123' not found",
		},
		{
			err:          AlreadyExists("User", "admin@ticket.vn"),
			expectedCode: codes.AlreadyExists,
			expectedMsg:  "User 'admin@ticket.vn' already exists",
		},
		{
			err:          PermissionDenied("access denied"),
			expectedCode: codes.PermissionDenied,
			expectedMsg:  "access denied",
		},
		{
			err:          Unauthenticated("invalid token"),
			expectedCode: codes.Unauthenticated,
			expectedMsg:  "invalid token",
		},
		{
			err:          Internal("database failure"),
			expectedCode: codes.Internal,
			expectedMsg:  "database failure",
		},
		{
			err:          Unavailable("server starting up"),
			expectedCode: codes.Unavailable,
			expectedMsg:  "server starting up",
		},
		{
			err:          FailedPrecondition("ticket already reserved"),
			expectedCode: codes.FailedPrecondition,
			expectedMsg:  "ticket already reserved",
		},
	}

	for _, tc := range tests {
		if Code(tc.err) != tc.expectedCode {
			t.Errorf("expected code %v, got %v", tc.expectedCode, Code(tc.err))
		}
		if Message(tc.err) != tc.expectedMsg {
			t.Errorf("expected msg %q, got %q", tc.expectedMsg, Message(tc.err))
		}
	}
}

func TestToHTTPStatus(t *testing.T) {
	cases := []struct {
		err            error
		expectedStatus int
	}{
		{nil, http.StatusOK},
		{InvalidArgument("bad"), http.StatusBadRequest},
		{NotFound("Item", 1), http.StatusNotFound},
		{AlreadyExists("Item", 1), http.StatusConflict},
		{PermissionDenied("denied"), http.StatusForbidden},
		{Unauthenticated("no auth"), http.StatusUnauthorized},
		{FailedPrecondition("pre"), http.StatusPreconditionFailed},
		{Internal("err"), http.StatusInternalServerError},
		{Unavailable("down"), http.StatusServiceUnavailable},
		{errors.New("unknown error"), http.StatusInternalServerError},
	}

	for _, c := range cases {
		got := ToHTTPStatus(c.err)
		if got != c.expectedStatus {
			t.Errorf("ToHTTPStatus(%v) = %d, expected %d", c.err, got, c.expectedStatus)
		}
	}
}
