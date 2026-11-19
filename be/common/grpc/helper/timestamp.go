package grpcHelper

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// TimeToTimestamp converts time.Time to *timestamppb.Timestamp.
// If the input time is zero, it returns nil.
func TimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// TimestampToTime converts *timestamppb.Timestamp to time.Time.
// If the input is nil or invalid, it returns time.Time zero value.
func TimestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}
