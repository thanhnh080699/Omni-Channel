package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitPublisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitPublisher(ctx context.Context, url string, partitions int) (*RabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	p := &RabbitPublisher{conn: conn, ch: ch}
	if err := p.DeclareTopology(ctx, partitions); err != nil {
		_ = p.Close()
		return nil, err
	}
	return p, nil
}

func (p *RabbitPublisher) DeclareTopology(_ context.Context, partitions int) error {
	for _, exchange := range []string{InboundExchange, OutboundExchange, RetryExchange, DLQExchange} {
		if err := p.ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
			return err
		}
	}
	if _, err := p.ch.QueueDeclare(DispatcherQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := p.ch.QueueBind(DispatcherQueue, InboundRoutingKey, InboundExchange, false, nil); err != nil {
		return err
	}
	for i := 0; i < partitions; i++ {
		name := fmt.Sprintf(ConversationQueueFmt, i)
		if _, err := p.ch.QueueDeclare(name, true, false, false, false, nil); err != nil {
			return err
		}
	}
	retryTTLs := []int32{5000, 30000, 300000}
	for attempt, ttl := range retryTTLs {
		name := fmt.Sprintf("outbound.retry.%d", attempt+1)
		args := amqp.Table{
			"x-message-ttl":             ttl,
			"x-dead-letter-exchange":    OutboundExchange,
			"x-dead-letter-routing-key": OutboundRoutingKey,
		}
		if _, err := p.ch.QueueDeclare(name, true, false, false, false, args); err != nil {
			return err
		}
	}
	if _, err := p.ch.QueueDeclare(OutboundQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := p.ch.QueueBind(OutboundQueue, OutboundRoutingKey, OutboundExchange, false, nil); err != nil {
		return err
	}
	if _, err := p.ch.QueueDeclare(DLQQueue, true, false, false, false, nil); err != nil {
		return err
	}
	return p.ch.QueueBind(DLQQueue, DLQRoutingKey, DLQExchange, false, nil)
}

func (p *RabbitPublisher) PublishInbound(ctx context.Context, payload InboundEventPayload) error {
	return p.publish(ctx, InboundExchange, InboundRoutingKey, payload)
}

func (p *RabbitPublisher) PublishConversation(ctx context.Context, partition int, payload ConversationEventPayload) error {
	return p.publish(ctx, "", fmt.Sprintf(ConversationQueueFmt, partition), payload)
}

func (p *RabbitPublisher) PublishOutbound(ctx context.Context, payload OutboundEventPayload) error {
	return p.publish(ctx, OutboundExchange, OutboundRoutingKey, payload)
}

func (p *RabbitPublisher) PublishOutboundRetry(ctx context.Context, attempt int, payload OutboundEventPayload) error {
	return p.publish(ctx, "", fmt.Sprintf("outbound.retry.%d", attempt), payload)
}

func (p *RabbitPublisher) PublishDLQ(ctx context.Context, payload DLQPayload) error {
	return p.publish(ctx, DLQExchange, DLQRoutingKey, payload)
}

func (p *RabbitPublisher) publish(ctx context.Context, exchange string, routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
}

func (p *RabbitPublisher) Consume(queueName string, consumerTag string, prefetch int) (<-chan amqp.Delivery, error) {
	if prefetch <= 0 {
		prefetch = 1
	}
	if err := p.ch.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}
	return p.ch.Consume(queueName, consumerTag, false, false, false, false, nil)
}

func (p *RabbitPublisher) Close() error {
	var err error
	if p.ch != nil {
		err = p.ch.Close()
	}
	if p.conn != nil {
		if closeErr := p.conn.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}
