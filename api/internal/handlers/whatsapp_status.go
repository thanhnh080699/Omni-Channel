package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type whatsAppMessageStatusRequest struct {
	ChannelAccountID string `json:"channel_account_id"`
	ChannelMessageID string `json:"channel_message_id"`
	Status           string `json:"status"`
	EventTime        string `json:"event_time"`
}

func (h *Handler) whatsAppMessageStatus(c *gin.Context) {
	if h.cfg.WebhookSharedSecret != "" && c.GetHeader("X-Webhook-Secret") != h.cfg.WebhookSharedSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req whatsAppMessageStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ChannelAccountID == "" || req.ChannelMessageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel_account_id and channel_message_id are required"})
		return
	}
	eventTime := time.Now().UTC()
	if req.EventTime != "" {
		if parsed, err := time.Parse(time.RFC3339, req.EventTime); err == nil {
			eventTime = parsed
		}
	}
	set := bson.M{"updated_at": time.Now().UTC()}
	switch req.Status {
	case "read", "seen":
		set["status"] = "seen"
		set["read_at"] = eventTime
		set["delivered_at"] = eventTime
	case "delivered":
		set["status"] = "delivered"
		set["delivered_at"] = eventTime
	case "sent":
		set["status"] = "sent"
		set["sent_at"] = eventTime
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported status"})
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("messages").UpdateOne(ctx, bson.M{
		"channel_message_key": req.ChannelAccountID + ":" + req.ChannelMessageID,
		"direction":           "outbound",
	}, bson.M{"$set": set})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update message status"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
