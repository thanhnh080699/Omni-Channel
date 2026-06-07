package channel

import (
	"context"
	"time"
)

type ChannelAccount struct {
	ID               string
	ChannelID        string
	Name             string
	CredentialRef    string
	WebhookSecretRef string
}

type Attachment struct {
	ID          string `json:"id,omitempty" bson:"id,omitempty"`
	Type        string `json:"type" bson:"type"`
	URL         string `json:"url,omitempty" bson:"url,omitempty"`
	MimeType    string `json:"mime_type,omitempty" bson:"mime_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty" bson:"size_bytes,omitempty"`
	Filename    string `json:"filename,omitempty" bson:"filename,omitempty"`
	ContentHash string `json:"content_hash,omitempty" bson:"content_hash,omitempty"`
}

type NormalizedInboundMessage struct {
	Channel                string       `json:"channel" bson:"channel"`
	ChannelAccountID       string       `json:"channel_account_id" bson:"channel_account_id"`
	ExternalConversationID string       `json:"external_conversation_id" bson:"external_conversation_id"`
	ExternalMessageID      string       `json:"external_message_id" bson:"external_message_id"`
	SenderExternalID       string       `json:"sender_external_id" bson:"sender_external_id"`
	SenderDisplayName      string       `json:"sender_display_name,omitempty" bson:"sender_display_name,omitempty"`
	Direction              string       `json:"direction" bson:"direction"`
	Text                   string       `json:"text,omitempty" bson:"text,omitempty"`
	Attachments            []Attachment `json:"attachments,omitempty" bson:"attachments,omitempty"`
	EventTime              time.Time    `json:"event_time" bson:"event_time"`
	RawEventID             string       `json:"raw_event_id" bson:"raw_event_id"`
	RawPayload             any          `json:"raw_payload,omitempty" bson:"raw_payload,omitempty"`
}

type OutboundMessage struct {
	MessageID              string       `json:"message_id"`
	ChannelAccountID       string       `json:"channel_account_id"`
	ExternalConversationID string       `json:"external_conversation_id"`
	Text                   string       `json:"text,omitempty"`
	Attachments            []Attachment `json:"attachments,omitempty"`
	IdempotencyKey         string       `json:"idempotency_key"`
	ExpiresAt              time.Time    `json:"expires_at"`
}

type SendResult struct {
	ChannelMessageID string    `json:"channel_message_id,omitempty"`
	Status           string    `json:"status"`
	SentAt           time.Time `json:"sent_at,omitempty"`
	RawResponse      any       `json:"raw_response,omitempty"`
}

type SyncCursor struct {
	AccountID string `json:"account_id"`
	Cursor    string `json:"cursor,omitempty"`
}

type SyncResult struct {
	Messages   []NormalizedInboundMessage `json:"messages"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

type ChannelHealth struct {
	Status     string    `json:"status"`
	Session    string    `json:"session"`
	LastSyncAt time.Time `json:"last_sync_at,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

type ChannelAdapter interface {
	ChannelCode() string
	ValidateWebhook(ctx context.Context, account ChannelAccount, headers map[string]string, body []byte) error
	NormalizeInbound(ctx context.Context, account ChannelAccount, raw []byte) ([]NormalizedInboundMessage, error)
	SendMessage(ctx context.Context, account ChannelAccount, req OutboundMessage) (SendResult, error)
	SyncConversation(ctx context.Context, account ChannelAccount, cursor SyncCursor) (SyncResult, error)
	Health(ctx context.Context, account ChannelAccount) (ChannelHealth, error)
}
