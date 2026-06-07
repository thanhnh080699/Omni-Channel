package workers

import (
	"context"
	"time"

	"omni-channel/backend/internal/channel"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/models"
	"omni-channel/backend/internal/queue"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConversationProcessor struct {
	db *database.Mongo
}

func NewConversationProcessor(db *database.Mongo) *ConversationProcessor {
	return &ConversationProcessor{db: db}
}

func (p *ConversationProcessor) Process(ctx context.Context, payload queue.ConversationEventPayload) error {
	normalized := normalizePayload(payload)
	now := time.Now().UTC()
	inbound := models.InboundEvent{
		Base:              base(),
		ChannelAccountID:  payload.ChannelAccountID,
		EventID:           payload.EventID,
		IdempotencyKey:    payload.IdempotencyKey,
		RawPayload:        payload.RawPayload,
		Status:            "processing",
		EventTime:         normalized.EventTime,
		GatewayReceivedAt: payload.GatewayReceivedAt,
		QueuedAt:          &payload.QueuedAt,
		AttemptCount:      payload.Attempt,
	}
	_, err := p.db.C("inbound_events").UpdateOne(ctx,
		bson.M{"idempotency_key": payload.IdempotencyKey},
		bson.M{"$setOnInsert": inbound},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	conversationID, err := p.upsertConversation(ctx, normalized, now)
	if err != nil {
		return err
	}
	if err := p.upsertMessage(ctx, conversationID, normalized, now); err != nil {
		return err
	}
	processedAt := time.Now().UTC()
	_, err = p.db.C("inbound_events").UpdateOne(ctx,
		bson.M{"idempotency_key": payload.IdempotencyKey},
		bson.M{"$set": bson.M{"status": "processed", "processed_at": processedAt, "updated_at": processedAt}},
	)
	return err
}

func (p *ConversationProcessor) upsertConversation(ctx context.Context, msg channel.NormalizedInboundMessage, now time.Time) (string, error) {
	conversationID := uuid.NewString()
	update := bson.M{
		"$setOnInsert": models.Conversation{
			Base:                   models.Base{ID: conversationID, CreatedAt: now, UpdatedAt: now},
			ChannelAccountID:       msg.ChannelAccountID,
			ExternalConversationID: msg.ExternalConversationID,
			CustomerRef:            msg.SenderExternalID,
			Status:                 "open",
			LastMessageAt:          msg.EventTime,
			UnreadCount:            0,
			Tags:                   []string{},
		},
		"$max": bson.M{"last_message_at": msg.EventTime},
	}
	result := p.db.C("conversations").FindOneAndUpdate(ctx,
		bson.M{"channel_account_id": msg.ChannelAccountID, "external_conversation_id": msg.ExternalConversationID},
		update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	)
	var conversation models.Conversation
	if err := result.Decode(&conversation); err != nil {
		return "", err
	}
	return conversation.ID, nil
}

func (p *ConversationProcessor) upsertMessage(ctx context.Context, conversationID string, msg channel.NormalizedInboundMessage, now time.Time) error {
	key := msg.ChannelAccountID + ":" + msg.ExternalMessageID
	message := models.Message{
		Base:              models.Base{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now},
		ConversationID:    conversationID,
		Direction:         "inbound",
		SenderType:        "customer",
		ChannelMessageID:  msg.ExternalMessageID,
		ChannelMessageKey: key,
		Text:              msg.Text,
		Status:            "delivered",
		EventTime:         msg.EventTime,
		DeliveredAt:       &now,
	}
	_, err := p.db.C("messages").UpdateOne(ctx,
		bson.M{"channel_message_key": key},
		bson.M{"$setOnInsert": message},
		options.Update().SetUpsert(true),
	)
	return err
}

func normalizePayload(payload queue.ConversationEventPayload) channel.NormalizedInboundMessage {
	if payload.Normalized != nil {
		return *payload.Normalized
	}
	eventTime := payload.EventTime
	if eventTime.IsZero() {
		eventTime = time.Now().UTC()
	}
	externalMessageID := stringField(payload.RawPayload, "external_message_id")
	if externalMessageID == "" {
		externalMessageID = payload.EventID
	}
	return channel.NormalizedInboundMessage{
		Channel:                payload.Channel,
		ChannelAccountID:       payload.ChannelAccountID,
		ExternalConversationID: firstNonEmpty(stringField(payload.RawPayload, "external_conversation_id"), payload.ConversationID),
		ExternalMessageID:      externalMessageID,
		SenderExternalID:       stringField(payload.RawPayload, "sender_external_id"),
		SenderDisplayName:      stringField(payload.RawPayload, "sender_display_name"),
		Direction:              "inbound",
		Text:                   stringField(payload.RawPayload, "text"),
		EventTime:              eventTime,
		RawEventID:             payload.EventID,
		RawPayload:             payload.RawPayload,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}

func base() models.Base {
	now := time.Now().UTC()
	return models.Base{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now}
}
