package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/models"
	"omni-channel/backend/internal/queue"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
)

type Gate interface {
	FirstSeen(ctx context.Context, key string) (bool, error)
}

type Runner struct {
	cfg       config.Config
	db        *database.Mongo
	publisher *queue.RabbitPublisher
	gate      Gate
}

func NewRunner(cfg config.Config, db *database.Mongo, publisher *queue.RabbitPublisher, gate Gate) *Runner {
	return &Runner{cfg: cfg, db: db, publisher: publisher, gate: gate}
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.startDispatcher(ctx); err != nil {
		return err
	}
	for i := 0; i < r.cfg.QueuePartitions; i++ {
		if err := r.startConversationWorker(ctx, i); err != nil {
			return err
		}
	}
	if err := r.startOutboundWorker(ctx); err != nil {
		return err
	}
	go r.resyncLoop(ctx)
	go r.startTrashCleanupLoop(ctx)
	<-ctx.Done()
	return ctx.Err()
}

func (r *Runner) startDispatcher(ctx context.Context) error {
	deliveries, err := r.publisher.Consume(queue.DispatcherQueue, "dispatcher", 1)
	if err != nil {
		return err
	}
	go func() {
		for delivery := range deliveries {
			var payload queue.InboundEventPayload
			if err := decodeDelivery(delivery, &payload); err != nil {
				log.Printf("dispatcher decode delivery: %v", err)
				_ = delivery.Nack(false, false)
				continue
			}
			first, err := r.gate.FirstSeen(ctx, "inbound:"+payload.IdempotencyKey)
			if err != nil {
				log.Printf("dispatcher idempotency key=%s: %v", payload.IdempotencyKey, err)
				_ = delivery.Nack(false, true)
				continue
			}
			if !first {
				_ = delivery.Ack(false)
				continue
			}
			conversationID := payload.ConversationID
			if conversationID == "" {
				conversationID = stringField(payload.RawPayload, "external_conversation_id")
			}
			partition := PartitionForConversation(conversationID, r.cfg.QueuePartitions)
			if err := r.publisher.PublishConversation(ctx, partition, queue.ConversationEventPayload{InboundEventPayload: payload, Partition: partition}); err != nil {
				log.Printf("dispatcher publish conversation partition=%d key=%s: %v", partition, payload.IdempotencyKey, err)
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}()
	return nil
}

func (r *Runner) startConversationWorker(ctx context.Context, partition int) error {
	queueName := fmt.Sprintf(queue.ConversationQueueFmt, partition)
	deliveries, err := r.publisher.Consume(queueName, fmt.Sprintf("conv-%d", partition), 1)
	if err != nil {
		return err
	}
	processor := NewConversationProcessor(r.db)
	go func() {
		for delivery := range deliveries {
			var payload queue.ConversationEventPayload
			if err := decodeDelivery(delivery, &payload); err != nil {
				log.Printf("conversation worker partition=%d decode delivery: %v", partition, err)
				_ = delivery.Nack(false, false)
				continue
			}
			if err := processor.Process(ctx, payload); err != nil {
				log.Printf("conversation worker partition=%d process key=%s: %v", partition, payload.IdempotencyKey, err)
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}()
	return nil
}

func (r *Runner) startOutboundWorker(ctx context.Context) error {
	deliveries, err := r.publisher.Consume(queue.OutboundQueue, "outbound", 1)
	if err != nil {
		return err
	}
	go func() {
		for delivery := range deliveries {
			var payload queue.OutboundEventPayload
			if err := decodeDelivery(delivery, &payload); err != nil {
				log.Printf("outbound decode delivery: %v", err)
				_ = delivery.Nack(false, false)
				continue
			}
			if !ShouldSendOutbound(time.Now().UTC(), payload.ExpiresAt) {
				log.Printf("outbound expired message_id=%s", payload.MessageID)
				_ = delivery.Ack(false)
				continue
			}
			first, err := r.gate.FirstSeen(ctx, fmt.Sprintf("outbound:%s:%d", payload.IdempotencyKey, payload.Attempt))
			if err != nil {
				log.Printf("outbound idempotency key=%s: %v", payload.IdempotencyKey, err)
				_ = delivery.Nack(false, true)
				continue
			}
			if !first {
				_ = delivery.Ack(false)
				continue
			}
			if channelMessageID, err := r.sendOutbound(ctx, payload); err != nil {
				nextAttempt := payload.Attempt + 1
				delay, retry := RetryDelay(nextAttempt)
				if !retry {
					_ = r.markOutboundFailed(ctx, payload, err.Error(), nextAttempt)
					_ = r.publisher.PublishDLQ(ctx, queue.DLQPayload{Kind: "outbound", Reason: err.Error(), FailedAt: time.Now().UTC(), Attempts: nextAttempt, RawPayload: payload})
					_ = delivery.Ack(false)
					continue
				}
				_ = r.markOutboundRetry(ctx, payload, err.Error(), nextAttempt)
				payload.Attempt = nextAttempt
				payload.QueuedAt = time.Now().UTC()
				log.Printf("outbound retry scheduled message_id=%s attempt=%d delay=%s", payload.MessageID, nextAttempt, delay)
				if err := r.publisher.PublishOutboundRetry(ctx, nextAttempt, payload); err != nil {
					_ = delivery.Nack(false, true)
					continue
				}
			} else if err := r.markOutboundSent(ctx, payload, channelMessageID); err != nil {
				log.Printf("outbound mark sent message_id=%s: %v", payload.MessageID, err)
				_ = delivery.Nack(false, true)
				continue
			}
			_ = delivery.Ack(false)
		}
	}()
	return nil
}

func (r *Runner) sendOutbound(ctx context.Context, payload queue.OutboundEventPayload) (string, error) {
	adapterPayload, err := r.buildOutboundAdapterPayload(ctx, payload)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(adapterPayload)
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, r.cfg.WhatsAppAdapterURL+"/send", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("adapter returned %s: %s", resp.Status, string(body))
	}
	var payloadResponse struct {
		ChannelMessageID string `json:"channel_message_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&payloadResponse)
	return payloadResponse.ChannelMessageID, nil
}

func (r *Runner) buildOutboundAdapterPayload(ctx context.Context, payload queue.OutboundEventPayload) (map[string]any, error) {
	var message models.Message
	if err := r.db.C("messages").FindOne(ctx, bson.M{"_id": payload.MessageID}).Decode(&message); err != nil {
		return nil, fmt.Errorf("load outbound message %s: %w", payload.MessageID, err)
	}
	var conversation models.Conversation
	if err := r.db.C("conversations").FindOne(ctx, bson.M{"_id": message.ConversationID}).Decode(&conversation); err != nil {
		return nil, fmt.Errorf("load outbound conversation %s: %w", message.ConversationID, err)
	}
	if conversation.ExternalConversationID == "" {
		return nil, fmt.Errorf("conversation %s has no external_conversation_id", conversation.ID)
	}
	return map[string]any{
		"message_id":               payload.MessageID,
		"outbound_event_id":        payload.OutboundEventID,
		"channel_account_id":       payload.ChannelAccountID,
		"idempotency_key":          payload.IdempotencyKey,
		"attempt":                  payload.Attempt,
		"queued_at":                payload.QueuedAt,
		"expires_at":               payload.ExpiresAt,
		"external_conversation_id": conversation.ExternalConversationID,
		"text":                     message.Text,
	}, nil
}

func (r *Runner) markOutboundSent(ctx context.Context, payload queue.OutboundEventPayload, channelMessageID string) error {
	now := time.Now().UTC()
	set := bson.M{
		"status":     "sent",
		"sent_at":    now,
		"updated_at": now,
	}
	if channelMessageID != "" {
		set["channel_message_id"] = channelMessageID
		set["channel_message_key"] = payload.ChannelAccountID + ":" + channelMessageID
	}
	if _, err := r.db.C("messages").UpdateByID(ctx, payload.MessageID, bson.M{"$set": set}); err != nil {
		return err
	}
	_, err := r.db.C("outbound_events").UpdateByID(ctx, payload.OutboundEventID, bson.M{"$set": bson.M{
		"status":        "sent",
		"attempt_count": payload.Attempt + 1,
		"sent_at":       now,
		"last_error":    "",
		"updated_at":    now,
	}})
	return err
}

func (r *Runner) markOutboundRetry(ctx context.Context, payload queue.OutboundEventPayload, reason string, nextAttempt int) error {
	now := time.Now().UTC()
	if _, err := r.db.C("messages").UpdateByID(ctx, payload.MessageID, bson.M{"$set": bson.M{
		"status":     "pending",
		"updated_at": now,
	}}); err != nil {
		return err
	}
	_, err := r.db.C("outbound_events").UpdateByID(ctx, payload.OutboundEventID, bson.M{"$set": bson.M{
		"status":        "pending",
		"attempt_count": nextAttempt,
		"last_error":    reason,
		"updated_at":    now,
	}})
	return err
}

func (r *Runner) markOutboundFailed(ctx context.Context, payload queue.OutboundEventPayload, reason string, attempts int) error {
	now := time.Now().UTC()
	if _, err := r.db.C("messages").UpdateByID(ctx, payload.MessageID, bson.M{"$set": bson.M{
		"status":     "failed",
		"updated_at": now,
	}}); err != nil {
		return err
	}
	_, err := r.db.C("outbound_events").UpdateByID(ctx, payload.OutboundEventID, bson.M{"$set": bson.M{
		"status":        "failed",
		"attempt_count": attempts,
		"last_error":    reason,
		"updated_at":    now,
	}})
	return err
}

func (r *Runner) resyncLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resyncCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			req, err := http.NewRequestWithContext(resyncCtx, http.MethodPost, r.cfg.WhatsAppAdapterURL+"/resync", nil)
			if err != nil {
				cancel()
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
			cancel()
		}
	}
}

func decodeDelivery(delivery amqp.Delivery, target any) error {
	return json.Unmarshal(delivery.Body, target)
}

func stringField(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func (r *Runner) startTrashCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.cleanupExpiredTrash(ctx)
		}
	}
}

func (r *Runner) cleanupExpiredTrash(ctx context.Context) {
	expiryTime := time.Now().UTC().Add(-30 * 24 * time.Hour)
	filter := bson.M{
		"status":     "deleted",
		"deleted_at": bson.M{"$lt": expiryTime},
	}

	cursor, err := r.db.C("conversations").Find(ctx, filter)
	if err != nil {
		log.Printf("[Trash Cleanup] Find conversations error: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var conversations []models.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		log.Printf("[Trash Cleanup] Decode conversations error: %v", err)
		return
	}

	for _, conv := range conversations {
		// 1. Delete all messages
		_, err := r.db.C("messages").DeleteMany(ctx, bson.M{"conversation_id": conv.ID})
		if err != nil {
			log.Printf("[Trash Cleanup] Delete messages error for conversation %s: %v", conv.ID, err)
			continue
		}

		// 2. Delete all members
		_, err = r.db.C("conversation_members").DeleteMany(ctx, bson.M{"conversation_id": conv.ID})
		if err != nil {
			log.Printf("[Trash Cleanup] Delete members error for conversation %s: %v", conv.ID, err)
			continue
		}

		// 3. Delete conversation document
		_, err = r.db.C("conversations").DeleteOne(ctx, bson.M{"_id": conv.ID})
		if err != nil {
			log.Printf("[Trash Cleanup] Delete conversation error for conversation %s: %v", conv.ID, err)
			continue
		}

		log.Printf("[Trash Cleanup] Permanently deleted conversation %s and all its messages (expired 30 days)", conv.ID)
	}
}
