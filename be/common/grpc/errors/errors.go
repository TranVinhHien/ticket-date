package grpcErrors

import (
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
