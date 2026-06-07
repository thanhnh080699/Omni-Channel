package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/queue"

	"github.com/gin-gonic/gin"
)

type fakePublisher struct {
	inbound []queue.InboundEventPayload
}

func (f *fakePublisher) PublishInbound(_ context.Context, payload queue.InboundEventPayload) error {
	f.inbound = append(f.inbound, payload)
	return nil
}

func (f *fakePublisher) PublishConversation(context.Context, int, queue.ConversationEventPayload) error {
	return nil
}

func (f *fakePublisher) PublishOutbound(context.Context, queue.OutboundEventPayload) error {
	return nil
}

func (f *fakePublisher) PublishOutboundRetry(context.Context, int, queue.OutboundEventPayload) error {
	return nil
}

func (f *fakePublisher) PublishDLQ(context.Context, queue.DLQPayload) error {
	return nil
}

func (f *fakePublisher) Close() error {
	return nil
}

func TestReceiveWebhookEnqueuesWithoutMongo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	publisher := &fakePublisher{}
	handler := &Handler{cfg: config.Config{}, queue: publisher}
	router := gin.New()
	router.POST("/webhooks/:channel/:accountId", handler.receiveWebhook)

	body := []byte(`{"event_id":"evt_1","external_conversation_id":"conv_1","event_time":"2026-06-06T01:00:00Z","text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp/ca_1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(publisher.inbound) != 1 {
		t.Fatalf("expected one inbound publish, got %d", len(publisher.inbound))
	}
	got := publisher.inbound[0]
	if got.Channel != "whatsapp" || got.ChannelAccountID != "ca_1" {
		t.Fatalf("unexpected channel envelope: %#v", got)
	}
	if got.IdempotencyKey != "ca_1:evt_1" {
		t.Fatalf("unexpected idempotency key %q", got.IdempotencyKey)
	}
}
