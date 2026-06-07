package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/queue"

	amqp "github.com/rabbitmq/amqp091-go"
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
				_ = delivery.Nack(false, false)
				continue
			}
			first, err := r.gate.FirstSeen(ctx, "inbound:"+payload.IdempotencyKey)
			if err != nil {
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
				_ = delivery.Nack(false, false)
				continue
			}
			if err := processor.Process(ctx, payload); err != nil {
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
				_ = delivery.Nack(false, false)
				continue
			}
			if !ShouldSendOutbound(time.Now().UTC(), payload.ExpiresAt) {
				log.Printf("outbound expired message_id=%s", payload.MessageID)
				_ = delivery.Ack(false)
				continue
			}
			first, err := r.gate.FirstSeen(ctx, "outbound:"+payload.IdempotencyKey)
			if err != nil {
				_ = delivery.Nack(false, true)
				continue
			}
			if !first {
				_ = delivery.Ack(false)
				continue
			}
			if err := r.sendOutbound(ctx, payload); err != nil {
				nextAttempt := payload.Attempt + 1
				delay, retry := RetryDelay(nextAttempt)
				if !retry {
					_ = r.publisher.PublishDLQ(ctx, queue.DLQPayload{Kind: "outbound", Reason: err.Error(), FailedAt: time.Now().UTC(), Attempts: nextAttempt, RawPayload: payload})
					_ = delivery.Ack(false)
					continue
				}
				payload.Attempt = nextAttempt
				payload.QueuedAt = time.Now().UTC()
				log.Printf("outbound retry scheduled message_id=%s attempt=%d delay=%s", payload.MessageID, nextAttempt, delay)
				if err := r.publisher.PublishOutboundRetry(ctx, nextAttempt, payload); err != nil {
					_ = delivery.Nack(false, true)
					continue
				}
			}
			_ = delivery.Ack(false)
		}
	}()
	return nil
}

func (r *Runner) sendOutbound(ctx context.Context, payload queue.OutboundEventPayload) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.WhatsAppAdapterURL+"/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("adapter returned %s", resp.Status)
	}
	return nil
}

func (r *Runner) resyncLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.cfg.WhatsAppAdapterURL+"/resync", nil)
			if err != nil {
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
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
