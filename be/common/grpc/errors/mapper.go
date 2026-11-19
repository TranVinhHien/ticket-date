package grpcErrors

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Code returns the gRPC status code of an error, or codes.OK if nil.
func Code(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	return status.Code(err)
}

// Message returns the gRPC error message, or empty string if nil.
func Message(err error) string {
	if err == nil {
		return ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}
	return st.Message()
}

// ToHTTPStatus maps a gRPC error or status code to its corresponding HTTP status code.
// This is especially useful for API Gateway or services returning REST responses from gRPC calls.
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	code := status.Code(err)
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
