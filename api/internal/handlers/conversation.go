package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

	filter, err := h.myConversationFilter(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	h.listConversations(c, user, filter)
}

func (h *Handler) listTeamConversations(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	filter, forbidden, err := h.teamConversationFilter(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if forbidden {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	h.listConversations(c, user, filter)
}

func (h *Handler) listConversations(c *gin.Context, user models.User, filter bson.M) {
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
	if err := h.enrichConversationReadState(ctx, user.ID, conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load read state"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conversations})
}

func (h *Handler) chatNotifications(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	filter, err := h.myConversationFilter(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	limit := parseLimit(c.Query("limit"), 20, 50)
	cursor, err := h.db.C("conversations").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "last_message_at", Value: -1}}).SetLimit(200))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list notifications"})
		return
	}
	defer cursor.Close(ctx)

	var conversations []models.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode notifications"})
		return
	}
	if err := h.enrichConversationReadState(ctx, user.ID, conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load read state"})
		return
	}

	items := []chatNotificationItem{}
	totalUnread := 0
	missedCount := 0
	var latestAt *time.Time
	for _, conversation := range conversations {
		if conversation.UnreadCount <= 0 {
			continue
		}
		totalUnread += conversation.UnreadCount
		missedCount++
		lastInbound, err := h.lastInboundMessage(ctx, conversation.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load notification message"})
			return
		}
		if latestAt == nil || conversation.LastMessageAt.After(*latestAt) {
			value := conversation.LastMessageAt
			latestAt = &value
		}
		if len(items) < limit {
			items = append(items, chatNotificationItem{
				ConversationID:   conversation.ID,
				CustomerName:     firstNonEmpty(conversation.CustomerName, conversation.CustomerRef, conversation.ExternalConversationID),
				CustomerRef:      firstNonEmpty(conversation.CustomerRef, conversation.ExternalConversationID),
				ChannelAccountID: conversation.ChannelAccountID,
				LastMessageText:  lastInbound.Text,
				LastMessageAt:    lastInbound.EventTime,
				UnreadCount:      conversation.UnreadCount,
			})
		}
	}

	c.JSON(http.StatusOK, chatNotificationSummary{
		TotalUnread: totalUnread,
		MissedCount: missedCount,
		LatestAt:    latestAt,
		Items:       items,
	})
}

func (h *Handler) listChatChannelAccounts(c *gin.Context) {
	if _, ok := currentUserOrAbort(c); !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	cursor, err := h.db.C("channel_accounts").Find(ctx, bson.M{}, options.Find().
		SetProjection(bson.M{"credential_ref": 0, "webhook_secret_ref": 0}).
		SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list channel accounts"})
		return
	}
	defer cursor.Close(ctx)

	var accounts []models.ChannelAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode channel accounts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

func (h *Handler) getConversation(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	h.audit(c, "conversation.view", "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"data": conversation})
}

func (h *Handler) conversationTyping(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	path := "/typing/" + url.PathEscape(conversation.ChannelAccountID) + "/" + url.PathEscape(conversation.ExternalConversationID)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, strings.TrimRight(h.cfg.WhatsAppAdapterURL, "/")+path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build adapter request"})
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"typing": false, "adapter_up": false})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
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

type updateConversationTagsRequest struct {
	Tags []string `json:"tags"`
}

func (h *Handler) updateConversationTags(c *gin.Context) {
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

	allowedAssign, err := h.rbac.Has(ctx, user, "conversation:assign")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	allowedSend, err := h.rbac.CanSendMessage(ctx, user, conversation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowedAssign && !allowedSend {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	var req updateConversationTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tags := normalizeConversationTags(req.Tags)
	result, err := h.db.C("conversations").UpdateByID(ctx, conversation.ID, updateTimeSet(bson.M{"tags": tags}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	h.audit(c, "conversation.tags.update", "conversation", conversation.ID, map[string]interface{}{"tags": tags})
	conversation.Tags = tags
	c.JSON(http.StatusOK, gin.H{"data": conversation})
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

	now := time.Now().UTC()
	messageID := c.Query("message_id")
	lastSeenAt := now
	if messageID != "" {
		var message models.Message
		err := h.db.C("messages").FindOne(ctx, bson.M{"_id": messageID, "conversation_id": conversation.ID}).Decode(&message)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load message"})
			return
		}
		lastSeenAt = message.EventTime
	}

	if _, err := h.db.C("conversation_members").UpdateOne(ctx,
		bson.M{"conversation_id": conversation.ID, "user_id": user.ID},
		bson.M{
			"$set": bson.M{
				"last_seen_message_id": messageID,
				"last_seen_at":         lastSeenAt,
				"updated_at":           now,
			},
			"$setOnInsert": bson.M{"_id": newBase().ID, "created_at": now, "access_level": "viewer", "source": "read_marker"},
		},
		options.Update().SetUpsert(true),
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update read marker"})
		return
	}
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

type chatNotificationSummary struct {
	TotalUnread int                    `json:"total_unread"`
	MissedCount int                    `json:"missed_count"`
	LatestAt    *time.Time             `json:"latest_at,omitempty"`
	Items       []chatNotificationItem `json:"items"`
}

type chatNotificationItem struct {
	ConversationID   string    `json:"conversation_id"`
	CustomerName     string    `json:"customer_name"`
	CustomerRef      string    `json:"customer_ref"`
	ChannelAccountID string    `json:"channel_account_id"`
	LastMessageText  string    `json:"last_message_text"`
	LastMessageAt    time.Time `json:"last_message_at"`
	UnreadCount      int       `json:"unread_count"`
}

func (h *Handler) myConversationFilter(ctx context.Context, user models.User) (bson.M, error) {
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	allowedAll, err := h.rbac.Has(ctx, user, "conversation:view_all")
	if err != nil {
		return nil, err
	}
	if !allowedAll {
		filter["$or"] = []bson.M{
			{"assigned_user_id": user.ID},
			{"_id": bson.M{"$in": h.memberConversationIDs(ctx, user.ID)}},
		}
	}
	return filter, nil
}

func (h *Handler) teamConversationFilter(ctx context.Context, user models.User) (bson.M, bool, error) {
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}
	allowedAll, err := h.rbac.Has(ctx, user, "conversation:view_all")
	if err != nil {
		return nil, false, err
	}
	if allowedAll {
		return filter, false, nil
	}
	allowedTeam, err := h.rbac.Has(ctx, user, "conversation:view_team")
	if err != nil {
		return nil, false, err
	}
	if !allowedTeam {
		return nil, true, nil
	}
	filter["assigned_team_id"] = bson.M{"$in": user.TeamIDs}
	return filter, false, nil
}

func (h *Handler) enrichConversationReadState(ctx context.Context, userID string, conversations []models.Conversation) error {
	if len(conversations) == 0 {
		return nil
	}
	ids := make([]string, 0, len(conversations))
	for _, conversation := range conversations {
		ids = append(ids, conversation.ID)
	}
	members := map[string]models.ConversationMember{}
	cursor, err := h.db.C("conversation_members").Find(ctx, bson.M{"user_id": userID, "conversation_id": bson.M{"$in": ids}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var rows []models.ConversationMember
	if err := cursor.All(ctx, &rows); err != nil {
		return err
	}
	for _, row := range rows {
		members[row.ConversationID] = row
	}
	for index := range conversations {
		member := members[conversations[index].ID]
		conversations[index].LastSeenAt = member.LastSeenAt
		count, err := h.unreadCount(ctx, conversations[index].ID, member.LastSeenAt)
		if err != nil {
			return err
		}
		conversations[index].UnreadCount = int(count)
		conversations[index].HasUnread = count > 0
		if lastInbound, err := h.lastInboundMessage(ctx, conversations[index].ID); err != nil {
			return err
		} else {
			conversations[index].LastMessageText = lastInbound.Text
		}
	}
	return nil
}

func (h *Handler) unreadCount(ctx context.Context, conversationID string, lastSeenAt *time.Time) (int64, error) {
	return h.db.C("messages").CountDocuments(ctx, unreadMessageFilter(conversationID, lastSeenAt))
}

func unreadMessageFilter(conversationID string, lastSeenAt *time.Time) bson.M {
	filter := bson.M{"conversation_id": conversationID, "direction": "inbound"}
	if lastSeenAt != nil {
		filter["event_time"] = bson.M{"$gt": *lastSeenAt}
	}
	return filter
}

func (h *Handler) lastInboundMessage(ctx context.Context, conversationID string) (models.Message, error) {
	var message models.Message
	err := h.db.C("messages").FindOne(ctx,
		bson.M{"conversation_id": conversationID, "direction": "inbound"},
		options.FindOne().SetSort(bson.D{{Key: "event_time", Value: -1}, {Key: "_id", Value: -1}}),
	).Decode(&message)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.Message{}, nil
	}
	return message, err
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func normalizeConversationTags(values []string) []string {
	const maxTags = 20
	const maxTagLength = 32
	seen := map[string]bool{}
	tags := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.ToLower(strings.TrimSpace(value))
		if tag == "" {
			continue
		}
		if len(tag) > maxTagLength {
			tag = tag[:maxTagLength]
		}
		if seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
		if len(tags) == maxTags {
			break
		}
	}
	return tags
}

type createConversationRequest struct {
	ChannelAccountID string `json:"channel_account_id" binding:"required"`
	CustomerRef      string `json:"customer_ref" binding:"required"`
}

func (h *Handler) createConversation(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var account models.ChannelAccount
	err := h.db.C("channel_accounts").FindOne(ctx, bson.M{"_id": req.ChannelAccountID}).Decode(&account)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel account not found"})
		return
	}

	var channel models.Channel
	err = h.db.C("channels").FindOne(ctx, bson.M{"_id": account.ChannelID}).Decode(&channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel not found"})
		return
	}

	externalConvID := strings.TrimSpace(req.CustomerRef)
	if channel.Code == "whatsapp" {
		// Clean phone number: keep only digits
		var digits []rune
		for _, r := range externalConvID {
			if r >= '0' && r <= '9' {
				digits = append(digits, r)
			}
		}
		cleanPhone := string(digits)
		if strings.HasPrefix(cleanPhone, "0") {
			cleanPhone = "84" + cleanPhone[1:]
		}
		req.CustomerRef = cleanPhone
		if !strings.Contains(externalConvID, "@") {
			externalConvID = cleanPhone + "@s.whatsapp.net"
		} else {
			parts := strings.Split(externalConvID, "@")
			num := parts[0]
			var numDigits []rune
			for _, r := range num {
				if r >= '0' && r <= '9' {
					numDigits = append(numDigits, r)
				}
			}
			cleanNum := string(numDigits)
			if strings.HasPrefix(cleanNum, "0") {
				cleanNum = "84" + cleanNum[1:]
			}
			externalConvID = cleanNum + "@s.whatsapp.net"
		}
	}

	var existing models.Conversation
	err = h.db.C("conversations").FindOne(ctx, bson.M{
		"channel_account_id":       req.ChannelAccountID,
		"external_conversation_id": externalConvID,
	}).Decode(&existing)

	if err == nil {
		convs := []models.Conversation{existing}
		if err := h.enrichConversationReadState(ctx, user.ID, convs); err == nil {
			existing = convs[0]
		}
		c.JSON(http.StatusOK, gin.H{"data": existing})
		return
	}

	now := time.Now().UTC()
	conversation := models.Conversation{
		Base:                   newBase(),
		ChannelAccountID:       req.ChannelAccountID,
		ExternalConversationID: externalConvID,
		CustomerRef:            req.CustomerRef,
		CustomerName:           req.CustomerRef,
		Status:                 "open",
		LastMessageAt:          now,
		UnreadCount:            0,
		Tags:                   []string{},
	}
	conversation.AssignedUserID = user.ID
	if len(user.TeamIDs) > 0 {
		conversation.AssignedTeamID = user.TeamIDs[0]
	}

	if _, err := h.db.C("conversations").InsertOne(ctx, conversation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create conversation"})
		return
	}

	h.audit(c, "conversation.create", "conversation", conversation.ID, map[string]interface{}{
		"channel_account_id":       req.ChannelAccountID,
		"external_conversation_id": externalConvID,
	})

	c.JSON(http.StatusOK, gin.H{"data": conversation})
}

func (h *Handler) checkChannelAccountPhone(c *gin.Context) {
	accountId := c.Param("accountId")
	phone := strings.TrimSpace(c.Query("phone"))
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone query parameter is required"})
		return
	}

	ctx, cancel := timeout(c)
	defer cancel()

	var account models.ChannelAccount
	err := h.db.C("channel_accounts").FindOne(ctx, bson.M{"_id": accountId}).Decode(&account)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel account not found"})
		return
	}

	var channel models.Channel
	err = h.db.C("channels").FindOne(ctx, bson.M{"_id": account.ChannelID}).Decode(&channel)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	if channel.Code == "whatsapp" {
		// Clean phone number: keep only digits
		var digits []rune
		for _, r := range phone {
			if r >= '0' && r <= '9' {
				digits = append(digits, r)
			}
		}
		cleanPhone := string(digits)
		if strings.HasPrefix(cleanPhone, "0") {
			cleanPhone = "84" + cleanPhone[1:]
		}
		path := "/on-whatsapp/" + url.PathEscape(accountId) + "/" + url.PathEscape(cleanPhone)
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, strings.TrimRight(h.cfg.WhatsAppAdapterURL, "/")+path, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build adapter request"})
			return
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"exists": false, "error": "whatsapp adapter unavailable"})
			return
		}
		defer resp.Body.Close()

		var result struct {
			Exists bool   `json:"exists"`
			Error  string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(http.StatusOK, gin.H{"exists": false, "error": "could not decode adapter response"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"exists": result.Exists})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": true})
}

func (h *Handler) deleteConversation(c *gin.Context) {
	conversation, allowed := h.loadConversationWithView(c)
	if !allowed {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	now := time.Now().UTC()
	result, err := h.db.C("conversations").UpdateByID(ctx, conversation.ID, updateTimeSet(bson.M{
		"status":     "deleted",
		"deleted_at": now,
	}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete conversation"})
		return
	}

	h.audit(c, "conversation.delete", "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) restoreConversation(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	var conversation models.Conversation
	err := h.db.C("conversations").FindOne(ctx, bson.M{"_id": c.Param("conversationId")}).Decode(&conversation)
	if errors.Is(err, mongo.ErrNoDocuments) {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load conversation"})
		return
	}

	allowed, err := h.rbac.CanViewConversation(ctx, user, conversation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	tags := conversation.Tags
	hasTrashTag := false
	for _, t := range tags {
		if t == "trash" {
			hasTrashTag = true
			break
		}
	}
	if !hasTrashTag {
		tags = append(tags, "trash")
	}

	var lastMsg models.Message
	lastMsgTime := time.Now().UTC()
	err = h.db.C("messages").FindOne(ctx,
		bson.M{"conversation_id": conversation.ID},
		options.FindOne().SetSort(bson.D{{Key: "event_time", Value: -1}}),
	).Decode(&lastMsg)
	if err == nil {
		lastMsgTime = lastMsg.EventTime
	}

	set := bson.M{
		"status":          "open",
		"deleted_at":      nil,
		"tags":            tags,
		"last_message_at": lastMsgTime,
	}
	_, err = h.db.C("conversations").UpdateByID(ctx, conversation.ID, updateTimeSet(set))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not restore conversation"})
		return
	}

	conversation.Status = "open"
	conversation.DeletedAt = nil
	conversation.Tags = tags
	conversation.LastMessageAt = lastMsgTime

	h.audit(c, "conversation.restore", "conversation", conversation.ID, nil)
	c.JSON(http.StatusOK, gin.H{"data": conversation})
}

func (h *Handler) listTrashConversations(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	filter := bson.M{"status": "deleted"}
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

	limit := parseLimit(c.Query("limit"), 50, 200)
	cursor, err := h.db.C("conversations").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "deleted_at", Value: -1}}).SetLimit(int64(limit)))
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
	if err := h.enrichConversationReadState(ctx, user.ID, conversations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load read state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": conversations})
}

func (h *Handler) getChannelAccountAvatar(c *gin.Context) {
	accountId := c.Param("accountId")
	jid := strings.TrimSpace(c.Query("jid"))
	if jid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jid query parameter is required"})
		return
	}

	ctx, cancel := timeout(c)
	defer cancel()

	var account models.ChannelAccount
	err := h.db.C("channel_accounts").FindOne(ctx, bson.M{"_id": accountId}).Decode(&account)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel account not found"})
		return
	}

	var channel models.Channel
	err = h.db.C("channels").FindOne(ctx, bson.M{"_id": account.ChannelID}).Decode(&channel)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	if channel.Code == "whatsapp" {
		path := "/avatar/" + url.PathEscape(accountId) + "/" + url.PathEscape(jid)
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, strings.TrimRight(h.cfg.WhatsAppAdapterURL, "/")+path, nil)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"url": ""})
			return
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"url": "", "error": "whatsapp adapter unavailable"})
			return
		}
		defer resp.Body.Close()

		var result struct {
			URL   string `json:"url"`
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(http.StatusOK, gin.H{"url": ""})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": result.URL})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": ""})
}
