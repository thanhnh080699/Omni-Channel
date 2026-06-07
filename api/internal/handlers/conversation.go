package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"omni-channel/backend/internal/models"
	"omni-channel/backend/internal/queue"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (h *Handler) listMyConversations(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	allowedAll, err := h.rbac.Has(ctx, user, "conversation:view_all")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowedAll {
		filter["$or"] = []bson.M{
			{"assigned_user_id": user.ID},
			{"_id": bson.M{"$in": h.memberConversationIDs(ctx, user.ID)}},
		}
	}
	h.listConversations(c, filter)
}

func (h *Handler) listTeamConversations(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	allowedAll, err := h.rbac.Has(ctx, user, "conversation:view_all")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	if !allowedAll {
		allowedTeam, err := h.rbac.Has(ctx, user, "conversation:view_team")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !allowedTeam {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		filter["assigned_team_id"] = bson.M{"$in": user.TeamIDs}
	}
	h.listConversations(c, filter)
}

func (h *Handler) listConversations(c *gin.Context, filter bson.M) {
	ctx, cancel := timeout(c)
	defer cancel()

	limit := parseLimit(c.Query("limit"), 50, 200)
	cursor, err := h.db.C("conversations").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "last_message_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list conversations"})
		return
	}
	defer cursor.Close(ctx)

	var conversations []models.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode conversations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conversations})
}

func (h *Handler) getConversation(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	h.audit(c, "conversation.view", "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"data": conversation})
}

type assignConversationRequest struct {
	AssignedUserID string `json:"assigned_user_id"`
	AssignedTeamID string `json:"assigned_team_id"`
}

func (h *Handler) assignConversation(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	allowed, err := h.rbac.Has(ctx, user, "conversation:assign")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req assignConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	set := bson.M{"assigned_user_id": req.AssignedUserID, "assigned_team_id": req.AssignedTeamID}
	result, err := h.db.C("conversations").UpdateByID(ctx, c.Param("conversationId"), updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	h.audit(c, "conversation.assign", "conversation", c.Param("conversationId"), set)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) closeConversation(c *gin.Context) {
	h.setConversationStatus(c, "closed", "conversation.close")
}

func (h *Handler) reopenConversation(c *gin.Context) {
	h.setConversationStatus(c, "open", "conversation.reopen")
}

func (h *Handler) setConversationStatus(c *gin.Context, status string, action string) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	result, err := h.db.C("conversations").UpdateByID(ctx, conversation.ID, updateTimeSet(bson.M{"status": status}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	h.audit(c, action, "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) listMessages(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	filter := bson.M{"conversation_id": conversation.ID}
	if after := c.Query("after"); after != "" {
		filter["_id"] = bson.M{"$gt": after}
	}
	limit := parseLimit(c.Query("limit"), 50, 200)
	cursor, err := h.db.C("messages").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "event_time", Value: 1}, {Key: "_id", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list messages"})
		return
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode messages"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": messages})
}

type sendMessageRequest struct {
	Text string `json:"text" binding:"required"`
}

func (h *Handler) sendMessage(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	conversation, allowedView := h.loadConversationWithView(c)
	if !allowedView {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	allowedSend, err := h.rbac.CanSendMessage(ctx, user, conversation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowedSend {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot send message in this conversation"})
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC()
	message := models.Message{
		Base:           newBase(),
		ConversationID: conversation.ID,
		Direction:      "outbound",
		SenderType:     "agent",
		SenderUserID:   user.ID,
		Text:           req.Text,
		Status:         "pending",
		EventTime:      now,
	}
	outbound := models.OutboundEvent{
		Base:             newBase(),
		MessageID:        message.ID,
		ChannelAccountID: conversation.ChannelAccountID,
		IdempotencyKey:   message.ID + ":send",
		Status:           "pending",
		AttemptCount:     0,
	}

	if _, err := h.db.C("messages").InsertOne(ctx, message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save message"})
		return
	}
	if _, err := h.db.C("outbound_events").InsertOne(ctx, outbound); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create outbound event"})
		return
	}
	if h.queue != nil {
		if err := h.queue.PublishOutbound(ctx, queue.OutboundEventPayload{
			MessageID:        message.ID,
			OutboundEventID:  outbound.ID,
			ChannelAccountID: conversation.ChannelAccountID,
			IdempotencyKey:   outbound.IdempotencyKey,
			Attempt:          0,
			QueuedAt:         now,
			ExpiresAt:        now.Add(h.cfg.OutboundTTL),
		}); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not enqueue outbound message"})
			return
		}
	}
	_, _ = h.db.C("conversations").UpdateByID(ctx, conversation.ID, updateTimeSet(bson.M{"last_message_at": now}))
	h.audit(c, "message.send", "conversation", conversation.ID, map[string]interface{}{"message_id": message.ID})
	c.JSON(http.StatusAccepted, gin.H{"data": message, "outbound_event": outbound})
}

func (h *Handler) markConversationRead(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	user, _ := currentUserOrAbort(c)
	ctx, cancel := timeout(c)
	defer cancel()

	_, _ = h.db.C("conversation_members").UpdateOne(ctx,
		bson.M{"conversation_id": conversation.ID, "user_id": user.ID},
		bson.M{"$set": bson.M{"last_seen_message_id": c.Query("message_id"), "updated_at": time.Now().UTC()}, "$setOnInsert": bson.M{"_id": newBase().ID, "created_at": time.Now().UTC(), "access_level": "viewer", "source": "read_marker"}},
		options.Update().SetUpsert(true),
	)
	h.audit(c, "conversation.read", "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) retryMessage(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	var message models.Message
	if err := h.db.C("messages").FindOne(ctx, bson.M{"_id": c.Param("messageId")}).Decode(&message); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	var conversation models.Conversation
	if err := h.db.C("conversations").FindOne(ctx, bson.M{"_id": message.ConversationID}).Decode(&conversation); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	allowed, err := h.rbac.CanSendMessage(ctx, user, conversation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	_, _ = h.db.C("messages").UpdateByID(ctx, message.ID, updateTimeSet(bson.M{"status": "pending"}))
	_, _ = h.db.C("outbound_events").UpdateOne(ctx, bson.M{"message_id": message.ID}, updateTimeSet(bson.M{"status": "pending", "last_error": ""}))
	if h.queue != nil {
		if err := h.queue.PublishOutbound(ctx, queue.OutboundEventPayload{
			MessageID:        message.ID,
			ChannelAccountID: conversation.ChannelAccountID,
			IdempotencyKey:   message.ID + ":send",
			Attempt:          0,
			QueuedAt:         time.Now().UTC(),
			ExpiresAt:        time.Now().UTC().Add(h.cfg.OutboundTTL),
		}); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not enqueue retry"})
			return
		}
	}
	h.audit(c, "message.retry", "message", message.ID, nil)
	c.JSON(http.StatusAccepted, gin.H{"status": "queued"})
}

func (h *Handler) loadConversationWithView(c *gin.Context) (models.Conversation, bool) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return models.Conversation{}, false
	}
	ctx, cancel := timeout(c)
	defer cancel()

	var conversation models.Conversation
	err := h.db.C("conversations").FindOne(ctx, bson.M{"_id": c.Param("conversationId")}).Decode(&conversation)
	if errors.Is(err, mongo.ErrNoDocuments) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return models.Conversation{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load conversation"})
		return models.Conversation{}, false
	}
	allowed, err := h.rbac.CanViewConversation(ctx, user, conversation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return models.Conversation{}, false
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return models.Conversation{}, false
	}
	return conversation, true
}

func (h *Handler) memberConversationIDs(ctx context.Context, userID string) []string {
	cursor, err := h.db.C("conversation_members").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)
	var members []models.ConversationMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ConversationID)
	}
	return ids
}

func parseLimit(raw string, fallback int, max int) int {
	if raw == "" {
		return fallback
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > max {
		return max
	}
	return limit
}
