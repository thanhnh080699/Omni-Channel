package queue

import (
	"context"
	"time"

	"omni-channel/backend/internal/channel"
)

const (
	InboundExchange  = "inbound.exchange"
	OutboundExchange = "outbound.exchange"
	RetryExchange    = "retry.exchange"
	DLQExchange      = "dlq.exchange"

	DispatcherQueue = "dispatcher.queue"
	OutboundQueue   = "outbound.queue"
	DLQQueue        = "dlq.queue"

	InboundRoutingKey    = "inbound.dispatcher"
	OutboundRoutingKey   = "outbound.send"
	DLQRoutingKey        = "dead"
	ConversationQueueFmt = "conv.queue.%d"
)

type Publisher interface {
	PublishInbound(ctx context.Context, payload InboundEventPayload) error
	PublishConversation(ctx context.Context, partition int, payload ConversationEventPayload) error
	PublishOutbound(ctx context.Context, payload OutboundEventPayload) error
	PublishOutboundRetry(ctx context.Context, attempt int, payload OutboundEventPayload) error
	PublishDLQ(ctx context.Context, payload DLQPayload) error
	Close() error
}

type InboundEventPayload struct {
	EventID           string                            `json:"event_id"`
	IdempotencyKey    string                            `json:"idempotency_key"`
	Channel           string                            `json:"channel"`
	ChannelAccountID  string                            `json:"channel_account_id"`
	ConversationID    string                            `json:"conversation_id,omitempty"`
	RawPayload        map[string]any                    `json:"raw_payload"`
	GatewayReceivedAt time.Time                         `json:"gateway_received_at"`
	EventTime         time.Time                         `json:"event_time"`
	QueuedAt          time.Time                         `json:"queued_at"`
	Attempt           int                               `json:"attempt"`
	TraceID           string                            `json:"trace_id"`
	Normalized        *channel.NormalizedInboundMessage `json:"normalized,omitempty"`
}

type ConversationEventPayload struct {
	InboundEventPayload
	Partition int `json:"partition"`
}

type OutboundEventPayload struct {
	MessageID        string    `json:"message_id"`
	OutboundEventID  string    `json:"outbound_event_id"`
	ChannelAccountID string    `json:"channel_account_id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	Attempt          int       `json:"attempt"`
	QueuedAt         time.Time `json:"queued_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type DLQPayload struct {
	Kind       string    `json:"kind"`
	Reason     string    `json:"reason"`
	FailedAt   time.Time `json:"failed_at"`
	Attempts   int       `json:"attempts"`
	TraceID    string    `json:"trace_id,omitempty"`
	RawPayload any       `json:"raw_payload"`
}
