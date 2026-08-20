package database

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenInvalidDSN(t *testing.T) {
	_, err := Open(context.Background(), "not a postgres dsn")
	if err == nil || !strings.Contains(err.Error(), "parse dsn") {
		t.Fatalf("err=%v, want parse dsn error", err)
	}
}

func TestOpenPingFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	_, err := Open(ctx, "host=127.0.0.1 port=1 user=test password=test dbname=test sslmode=disable")
	if err == nil || !strings.Contains(err.Error(), "ping") {
		t.Fatalf("err=%v, want ping error", err)
	}
}
