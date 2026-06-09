package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/models"
	"omni-channel/backend/internal/rbac"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestUnreadMessageFilterCountsOnlyInboundAfterLastSeen(t *testing.T) {
	lastSeen := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	filter := unreadMessageFilter("conv_1", &lastSeen)

	if filter["conversation_id"] != "conv_1" || filter["direction"] != "inbound" {
		t.Fatalf("unexpected base filter: %#v", filter)
	}
	eventTime, ok := filter["event_time"].(bson.M)
	if !ok || eventTime["$gt"] != lastSeen {
		t.Fatalf("expected event_time greater than last_seen_at, got %#v", filter["event_time"])
	}
}

func TestMarkConversationReadSetsLastSeenAtFromMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("sets marker", func(mt *mtest.T) {
		messageAt := time.Date(2026, 6, 8, 11, 30, 0, 0, time.UTC)
		mt.AddMockResponses(
			cursorResponse(mt, "conversations", conversationDoc("conv_1", "open")),
			cursorResponse(mt, "roles", adminRoleDoc()),
			cursorResponse(mt, "messages", messageDoc("msg_1", "conv_1", "inbound", messageAt)),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		router := conversationNotificationsRouter(mt, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
		rec := performRequest(router, http.MethodPost, "/api/conversations/conv_1/read?message_id=msg_1", bytes.NewBuffer(nil))

		if rec.Code != http.StatusOK {
			mt.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestChatNotificationsReturnsUnreadVisibleItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("summary", func(mt *mtest.T) {
		now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
		lastSeen := now.Add(-10 * time.Minute)
		mt.AddMockResponses(
			cursorResponse(mt, "roles", adminRoleDoc()),
			cursorResponse(mt, "conversations", conversationDoc("conv_1", "open"), conversationDoc("conv_2", "open")),
			cursorResponse(mt, "conversation_members", bson.D{
				{Key: "_id", Value: "member_1"},
				{Key: "created_at", Value: now},
				{Key: "updated_at", Value: now},
				{Key: "conversation_id", Value: "conv_1"},
				{Key: "user_id", Value: "usr_admin"},
				{Key: "access_level", Value: "viewer"},
				{Key: "source", Value: "read_marker"},
				{Key: "last_seen_at", Value: lastSeen},
			}),
			countResponse(mt, "messages", 2),
			cursorResponse(mt, "messages", messageDoc("msg_2", "conv_1", "inbound", now)),
			countResponse(mt, "messages", 0),
			cursorResponse(mt, "messages"),
			cursorResponse(mt, "messages", messageDoc("msg_2", "conv_1", "inbound", now)),
		)
		router := conversationNotificationsRouter(mt, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
		rec := performRequest(router, http.MethodGet, "/api/notifications/chat?limit=20", bytes.NewBuffer(nil))

		if rec.Code != http.StatusOK {
			mt.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload chatNotificationSummary
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			mt.Fatalf("could not decode response: %v", err)
		}
		if payload.TotalUnread != 2 || payload.MissedCount != 1 || len(payload.Items) != 1 {
			mt.Fatalf("unexpected summary: %#v", payload)
		}
		if payload.Items[0].ConversationID != "conv_1" || payload.Items[0].UnreadCount != 2 {
			mt.Fatalf("unexpected item: %#v", payload.Items[0])
		}
	})
}

func conversationNotificationsRouter(mt *mtest.T, user models.User) *gin.Engine {
	db := &database.Mongo{Client: mt.Client, DB: mt.DB}
	handler := &Handler{db: db, rbac: rbac.NewChecker(db)}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CurrentUserKey, user)
		c.Next()
	})
	router.POST("/api/conversations/:conversationId/read", handler.markConversationRead)
	router.GET("/api/notifications/chat", handler.chatNotifications)
	return router
}

func messageDoc(id string, conversationID string, direction string, eventTime time.Time) bson.D {
	return bson.D{
		{Key: "_id", Value: id},
		{Key: "created_at", Value: eventTime},
		{Key: "updated_at", Value: eventTime},
		{Key: "conversation_id", Value: conversationID},
		{Key: "direction", Value: direction},
		{Key: "sender_type", Value: "customer"},
		{Key: "text", Value: "hello"},
		{Key: "status", Value: "delivered"},
		{Key: "event_time", Value: eventTime},
	}
}

func countResponse(mt *mtest.T, collection string, n int32) bson.D {
	return mtest.CreateCursorResponse(0, mt.DB.Name()+"."+collection, mtest.FirstBatch, bson.D{{Key: "n", Value: n}})
}
