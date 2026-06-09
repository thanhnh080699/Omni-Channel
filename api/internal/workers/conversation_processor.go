package workers

import (
	"context"
	"log"
	"strings"
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

	// Reconcile LID conversation if this is a WhatsApp channel and contains LID mapping
	if normalized.Channel == "whatsapp" {
		if err := p.reconcileLidConversation(ctx, normalized.ChannelAccountID, normalized.ExternalConversationID, normalized.RawPayload, now); err != nil {
			log.Printf("[LID Reconcile] Error reconciling LID conversation: %v", err)
		}
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
	update := conversationUpsertUpdate(conversationID, msg, now)
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

func conversationUpsertUpdate(conversationID string, msg channel.NormalizedInboundMessage, now time.Time) bson.M {
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":                      conversationID,
			"created_at":               now,
			"channel_account_id":       msg.ChannelAccountID,
			"external_conversation_id": msg.ExternalConversationID,
			"customer_ref":             msg.SenderExternalID,
			"status":                   "open",
			"unread_count":             0,
			"tags":                     []string{},
		},
		"$set": bson.M{"updated_at": now},
		"$max": bson.M{"last_message_at": msg.EventTime},
	}
	if msg.SenderDisplayName != "" {
		update["$set"].(bson.M)["customer_name"] = msg.SenderDisplayName
	}
	return update
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

func (p *ConversationProcessor) reconcileLidConversation(ctx context.Context, accountID string, targetPhoneJID string, rawPayload any, now time.Time) error {
	if !strings.HasSuffix(targetPhoneJID, "@s.whatsapp.net") {
		return nil
	}

	lidJID := getNestedStringField(rawPayload, "key", "remoteJid")
	if lidJID == "" || !strings.HasSuffix(lidJID, "@lid") {
		return nil
	}

	// 1. Find if there is an existing conversation for the LID JID
	var lidConv models.Conversation
	err := p.db.C("conversations").FindOne(ctx, bson.M{
		"channel_account_id":       accountID,
		"external_conversation_id": lidJID,
	}).Decode(&lidConv)
	if err != nil {
		// If no LID conversation exists, nothing to reconcile
		return nil
	}

	// 2. Check if a conversation for the phone JID already exists
	var phoneConv models.Conversation
	err = p.db.C("conversations").FindOne(ctx, bson.M{
		"channel_account_id":       accountID,
		"external_conversation_id": targetPhoneJID,
	}).Decode(&phoneConv)

	cleanPhone := strings.Split(targetPhoneJID, "@")[0]

	if err != nil {
		// Phone conversation does NOT exist yet: rename LID conversation to phone JID
		_, err = p.db.C("conversations").UpdateOne(ctx,
			bson.M{"_id": lidConv.ID},
			bson.M{"$set": bson.M{
				"external_conversation_id": targetPhoneJID,
				"customer_ref":             cleanPhone,
				"updated_at":               now,
			}},
		)
		if err != nil {
			return err
		}
		log.Printf("[LID Reconcile] Renamed conversation %s from LID %s to phone %s", lidConv.ID, lidJID, targetPhoneJID)
	} else {
		// Phone conversation DOES exist: merge LID conversation into phone conversation
		// Update all messages from LID conversation to phone conversation
		_, err = p.db.C("messages").UpdateMany(ctx,
			bson.M{"conversation_id": lidConv.ID},
			bson.M{"$set": bson.M{
				"conversation_id": phoneConv.ID,
				"updated_at":      now,
			}},
		)
		if err != nil {
			return err
		}

		// Delete the old LID conversation
		_, err = p.db.C("conversations").DeleteOne(ctx, bson.M{"_id": lidConv.ID})
		if err != nil {
			return err
		}

		// Delete the old LID conversation members
		_, _ = p.db.C("conversation_members").DeleteMany(ctx, bson.M{"conversation_id": lidConv.ID})

		log.Printf("[LID Reconcile] Merged LID conversation %s (%s) into phone conversation %s (%s)", lidConv.ID, lidJID, phoneConv.ID, targetPhoneJID)
	}

	return nil
}

func getNestedStringField(rawPayload any, path ...string) string {
	m, ok := rawPayload.(map[string]any)
	if !ok {
		// Check if it's bson.M (which is map[string]interface{})
		bm, ok := rawPayload.(bson.M)
		if !ok {
			return ""
		}
		m = map[string]any(bm)
	}
	for i, key := range path {
		val := m[key]
		if i == len(path)-1 {
			if str, ok := val.(string); ok {
				return str
			}
			return ""
		}
		nextMap, ok := val.(map[string]any)
		if !ok {
			nextBson, ok := val.(bson.M)
			if !ok {
				return ""
			}
			m = map[string]any(nextBson)
		} else {
			m = nextMap
		}
	}
	return ""
}
