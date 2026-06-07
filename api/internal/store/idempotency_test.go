package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestIdempotencyGateFirstSeen(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewRedisClient(server.Addr(), "", 0)
	defer client.Close()

	gate := NewIdempotencyGate(client, time.Hour)
	ctx := context.Background()
	first, err := gate.FirstSeen(ctx, "inbound:ca_1:evt_1")
	if err != nil {
		t.Fatal(err)
	}
	if !first {
		t.Fatal("expected first call to be accepted")
	}
	second, err := gate.FirstSeen(ctx, "inbound:ca_1:evt_1")
	if err != nil {
		t.Fatal(err)
	}
	if second {
		t.Fatal("expected duplicate call to be rejected")
	}
}
