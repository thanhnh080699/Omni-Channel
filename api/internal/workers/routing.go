package workers

import (
	"hash/fnv"
	"time"
)

var outboundRetryDelays = []time.Duration{5 * time.Second, 30 * time.Second, 5 * time.Minute}

func PartitionForConversation(conversationID string, partitions int) int {
	if partitions <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(conversationID))
	return int(h.Sum32() % uint32(partitions))
}

func RetryDelay(attempt int) (time.Duration, bool) {
	if attempt < 1 || attempt > len(outboundRetryDelays) {
		return 0, false
	}
	return outboundRetryDelays[attempt-1], true
}

func ShouldSendOutbound(now time.Time, expiresAt time.Time) bool {
	return expiresAt.IsZero() || !now.After(expiresAt)
}
