package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"omni-channel/backend/internal/queue"

	"github.com/gin-gonic/gin"
)

func (h *Handler) receiveWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read webhook body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err := h.verifyWebhook(c, body); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return
	}
	if h.queue == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "queue publisher is not configured"})
		return
	}

	channel := c.Param("channel")
	accountID := c.Param("accountId")
	payload["_source_channel"] = channel
	eventID := stringValue(payload["event_id"])
	if eventID == "" {
		eventID = stringValue(payload["id"])
	}
	idempotencyKey := accountID + ":" + eventID
	if eventID == "" {
		idempotencyKey = accountID + ":" + stablePayloadHash(payload)
	}

	receivedAt := time.Now().UTC()
	eventTime := receivedAt
	if rawEventTime := stringValue(payload["event_time"]); rawEventTime != "" {
		if parsed, err := time.Parse(time.RFC3339, rawEventTime); err == nil {
			eventTime = parsed
		}
	}

	ctx, cancel := timeout(c)
	defer cancel()
	queuedAt := time.Now().UTC()
	envelope := queue.InboundEventPayload{
		EventID:           eventID,
		IdempotencyKey:    idempotencyKey,
		Channel:           channel,
		ChannelAccountID:  accountID,
		ConversationID:    stringValue(payload["external_conversation_id"]),
		RawPayload:        payload,
		GatewayReceivedAt: receivedAt,
		EventTime:         eventTime,
		QueuedAt:          queuedAt,
		Attempt:           0,
		TraceID:           c.GetHeader("X-Request-ID"),
	}
	if err := h.queue.PublishInbound(ctx, envelope); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not enqueue inbound event"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "queued", "idempotency_key": idempotencyKey})
}

func (h *Handler) verifyWebhook(c *gin.Context, body []byte) error {
	secret := strings.TrimSpace(h.cfg.WebhookSharedSecret)
	if secret == "" {
		return nil
	}
	signature := c.GetHeader("X-Omni-Signature")
	if signature == "" {
		return errors.New("missing webhook signature")
	}
	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("invalid webhook signature")
	}
	return nil
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func stablePayloadHash(payload map[string]interface{}) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		encoded = []byte(stringValue(payload["message_id"]))
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
