package workers

import (
	"testing"
	"time"
)

func TestPartitionForConversationIsStable(t *testing.T) {
	first := PartitionForConversation("conv_123", 8)
	for i := 0; i < 20; i++ {
		if got := PartitionForConversation("conv_123", 8); got != first {
			t.Fatalf("partition changed from %d to %d", first, got)
		}
	}
	if first < 0 || first >= 8 {
		t.Fatalf("partition out of range: %d", first)
	}
}

func TestRetryDelaySchedule(t *testing.T) {
	cases := []struct {
		attempt int
		delay   time.Duration
		ok      bool
	}{
		{1, 5 * time.Second, true},
		{2, 30 * time.Second, true},
		{3, 5 * time.Minute, true},
		{4, 0, false},
	}
	for _, tc := range cases {
		delay, ok := RetryDelay(tc.attempt)
		if delay != tc.delay || ok != tc.ok {
			t.Fatalf("attempt %d: got %s/%v want %s/%v", tc.attempt, delay, ok, tc.delay, tc.ok)
		}
	}
}

func TestShouldSendOutboundHonorsExpiresAt(t *testing.T) {
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	if !ShouldSendOutbound(now, now.Add(time.Minute)) {
		t.Fatal("expected unexpired outbound to send")
	}
	if ShouldSendOutbound(now, now.Add(-time.Second)) {
		t.Fatal("expected expired outbound to be skipped")
	}
}
