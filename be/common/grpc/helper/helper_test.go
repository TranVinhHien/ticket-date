package grpcHelper

import (
	"testing"
	"time"
)

func TestStructPBConversion(t *testing.T) {
	data := map[string]any{
		"event_name":  "Rock Concert 2026",
		"price":       150.50,
		"is_sold_out": false,
		"tags":        []any{"rock", "live"},
	}

	st, err := ConvertMapToStructPB(data)
	if err != nil {
		t.Fatalf("ConvertMapToStructPB failed: %v", err)
	}

	converted := ConvertStructPBToMap(st)
	if converted["event_name"] != "Rock Concert 2026" {
		t.Errorf("expected event_name 'Rock Concert 2026', got %v", converted["event_name"])
	}
	if converted["price"] != 150.50 {
		t.Errorf("expected price 150.50, got %v", converted["price"])
	}
	if converted["is_sold_out"] != false {
		t.Errorf("expected is_sold_out false, got %v", converted["is_sold_out"])
	}

	// Test nil inputs
	nilSt, err := ConvertMapToStructPB(nil)
	if err != nil || nilSt == nil {
		t.Errorf("expected non-nil struct for nil map, got %v, err=%v", nilSt, err)
	}

	nilMap := ConvertStructPBToMap(nil)
	if nilMap == nil || len(nilMap) != 0 {
		t.Errorf("expected empty map for nil struct, got %v", nilMap)
	}
}

func TestTimestampConversion(t *testing.T) {
	now := time.Now().Truncate(time.Microsecond)
	ts := TimeToTimestamp(now)
	if ts == nil {
		t.Fatal("expected non-nil timestamp")
	}

	converted := TimestampToTime(ts)
	if !converted.Equal(now) {
		t.Errorf("expected %v, got %v", now, converted)
	}

	// Zero time test
	if zeroTs := TimeToTimestamp(time.Time{}); zeroTs != nil {
		t.Errorf("expected nil for zero time, got %v", zeroTs)
	}

	// Nil timestamp test
	if zeroTime := TimestampToTime(nil); !zeroTime.IsZero() {
		t.Errorf("expected zero time for nil timestamp, got %v", zeroTime)
	}
}
