package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/models"
	"omni-channel/backend/internal/rbac"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestDeleteConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			// 1. FindOne conversation in loadConversationWithView
			cursorResponse(mt, "conversations", conversationDoc("conv_1", "open")),
			// 2. Find roles in CanViewConversation
			cursorResponse(mt, "roles", adminRoleDoc()),
			// 3. UpdateByID conversations
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			// 4. InsertOne audit_logs
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)

		db := &database.Mongo{Client: mt.Client, DB: mt.DB}
		handler := &Handler{db: db, rbac: rbac.NewChecker(db)}
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CurrentUserKey, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
			c.Next()
		})
		router.DELETE("/api/conversations/:conversationId", handler.deleteConversation)

		rec := performRequest(router, http.MethodDelete, "/api/conversations/conv_1", bytes.NewBuffer(nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestRestoreConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			// 1. FindOne conversation
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch, conversationDoc("conv_1", "deleted")),
			// 2. Find roles in CanViewConversation
			cursorResponse(mt, "roles", adminRoleDoc()),
			// 3. FindOne message (mock no messages for simplicity)
			mtest.CreateCursorResponse(0, "omni_test.messages", mtest.FirstBatch), // empty cursor -> ErrNoDocuments
			// 4. UpdateByID conversations
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			// 5. InsertOne audit_logs
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)

		db := &database.Mongo{Client: mt.Client, DB: mt.DB}
		handler := &Handler{db: db, rbac: rbac.NewChecker(db)}
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CurrentUserKey, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
			c.Next()
		})
		router.POST("/api/conversations/:conversationId/restore", handler.restoreConversation)

		rec := performRequest(router, http.MethodPost, "/api/conversations/conv_1/restore", bytes.NewBuffer(nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}

		var payload struct {
			Data models.Conversation `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("could not decode response: %v", err)
		}
		if payload.Data.Status != "open" {
			t.Fatalf("expected status open, got %s", payload.Data.Status)
		}
		// Check that tag "trash" was added
		hasTrash := false
		for _, tag := range payload.Data.Tags {
			if tag == "trash" {
				hasTrash = true
				break
			}
		}
		if !hasTrash {
			t.Fatalf("expected tag 'trash' to be added")
		}
	})
}

func TestListTrashConversations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			// 1. Has perm check roles
			cursorResponse(mt, "roles", adminRoleDoc()),
			// 2. Find conversations (return 1 doc)
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch, conversationDoc("conv_1", "deleted")),
			// 3. enrichConversationReadState: Find members
			mtest.CreateCursorResponse(0, "omni_test.conversation_members", mtest.FirstBatch),
			// 4. enrichConversationReadState: Count messages
			countResponse(mt, "messages", 0),
			// 5. enrichConversationReadState: Find last message
			mtest.CreateCursorResponse(0, "omni_test.messages", mtest.FirstBatch),
		)

		db := &database.Mongo{Client: mt.Client, DB: mt.DB}
		handler := &Handler{db: db, rbac: rbac.NewChecker(db)}
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.CurrentUserKey, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
			c.Next()
		})
		router.GET("/api/conversations/trash", handler.listTrashConversations)

		rec := performRequest(router, http.MethodGet, "/api/conversations/trash", bytes.NewBuffer(nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}


