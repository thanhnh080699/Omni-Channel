package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"omni-channel/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (h *Handler) receiveWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
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

	now := time.Now().UTC()
	eventTime := now
	if rawEventTime := stringValue(payload["event_time"]); rawEventTime != "" {
		if parsed, err := time.Parse(time.RFC3339, rawEventTime); err == nil {
			eventTime = parsed
		}
	}

	event := models.InboundEvent{
		Base:              newBase(),
		ChannelAccountID:  accountID,
		EventID:           eventID,
		IdempotencyKey:    idempotencyKey,
		RawPayload:        payload,
		Status:            "received",
		EventTime:         eventTime,
		GatewayReceivedAt: now,
		AttemptCount:      0,
	}

	ctx, cancel := timeout(c)
	defer cancel()

	update := bson.M{
		"$setOnInsert": event,
	}
	result, err := h.db.C("inbound_events").UpdateOne(ctx, bson.M{"idempotency_key": idempotencyKey}, update, options.Update().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		c.JSON(http.StatusAccepted, gin.H{"status": "duplicate", "idempotency_key": idempotencyKey})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store inbound event"})
		return
	}
	status := "received"
	if result.MatchedCount > 0 {
		status = "duplicate"
	}
	c.JSON(http.StatusAccepted, gin.H{"status": status, "idempotency_key": idempotencyKey})
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
