package grpcErrors

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InvalidArgument returns a gRPC InvalidArgument error (HTTP 400).
func InvalidArgument(format string, args ...any) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}

// NotFound returns a gRPC NotFound error (HTTP 404).
func NotFound(resource string, id any) error {
	return status.Errorf(codes.NotFound, "%s '%v' not found", resource, id)
}

// AlreadyExists returns a gRPC AlreadyExists error (HTTP 409).
func AlreadyExists(resource string, id any) error {
	return status.Errorf(codes.AlreadyExists, "%s '%v' already exists", resource, id)
}

// PermissionDenied returns a gRPC PermissionDenied error (HTTP 403).
func PermissionDenied(format string, args ...any) error {
	return status.Errorf(codes.PermissionDenied, format, args...)
}

// Unauthenticated returns a gRPC Unauthenticated error (HTTP 401).
func Unauthenticated(format string, args ...any) error {
	return status.Errorf(codes.Unauthenticated, format, args...)
}

// Internal returns a gRPC Internal error (HTTP 500).
func Internal(format string, args ...any) error {
	return status.Errorf(codes.Internal, format, args...)
}

// Unavailable returns a gRPC Unavailable error (HTTP 503).
func Unavailable(format string, args ...any) error {
	return status.Errorf(codes.Unavailable, format, args...)
}

// FailedPrecondition returns a gRPC FailedPrecondition error (HTTP 412).
func FailedPrecondition(format string, args ...any) error {
	return status.Errorf(codes.FailedPrecondition, format, args...)
}

// InsufficientStock returns a gRPC FailedPrecondition error with INSUFFICIENT_STOCK prefix.
func InsufficientStock(ticketTypeID string) error {
	return status.Errorf(codes.FailedPrecondition, "INSUFFICIENT_STOCK: %s", ticketTypeID)
}

// IsInsufficientStock checks whether an error indicates insufficient stock and returns the ticket_type_id.
func IsInsufficientStock(err error) (bool, string) {
	if err == nil {
		return false, ""
	}
	msg := Message(err)
	if strings.HasPrefix(msg, "INSUFFICIENT_STOCK: ") {
		return true, strings.TrimSpace(strings.TrimPrefix(msg, "INSUFFICIENT_STOCK: "))
	}
	if strings.Contains(msg, "INSUFFICIENT_STOCK") {
		return true, msg
	}
	return false, ""
}
