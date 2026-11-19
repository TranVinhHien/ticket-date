package client

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNewClientConn_Validation(t *testing.T) {
	_, err := NewClientConn("")
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}

func TestNewClientConn_Success(t *testing.T) {
	logger := zerolog.Nop()
	conn, err := NewClientConn(
		"passthrough:///127.0.0.1:50051",
		WithTimeout(3*time.Second),
		WithLogger(&logger),
		WithInsecure(),
		WithKeepAlive(5*time.Second, 2*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create client connection: %v", err)
	}
	defer conn.Close()

	if conn.Target() != "passthrough:///127.0.0.1:50051" {
		t.Errorf("expected target passthrough:///127.0.0.1:50051, got %s", conn.Target())
	}
}
