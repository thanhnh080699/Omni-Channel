package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestNormalizeConversationTagsDedupesAndLowercases(t *testing.T) {
	got := normalizeConversationTags([]string{" VIP ", "vip", "Lead", "", " lead "})
	want := []string{"vip", "lead"}
	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %#v", len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected tag %d to be %q, got %q", i, want[i], got[i])
		}
	}
}

func TestNormalizeConversationTagsCapsLengthAndCount(t *testing.T) {
	values := make([]string, 0, 25)
	values = append(values, "abcdefghijklmnopqrstuvwxyz0123456789")
	for i := 0; i < 24; i++ {
		values = append(values, "tag"+string(rune('a'+i)))
	}
	got := normalizeConversationTags(values)
	if len(got) != 20 {
		t.Fatalf("expected 20 tags, got %d", len(got))
	}
	if len(got[0]) != 32 {
		t.Fatalf("expected first tag length 32, got %d: %q", len(got[0]), got[0])
	}
}

func TestUpdateConversationTagsNormalizesAndSaves(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(
			cursorResponse(mt, "conversations", conversationDoc("conv_1", "open")),
			cursorResponse(mt, "roles", adminRoleDoc()),
			cursorResponse(mt, "roles", adminRoleDoc()),
			cursorResponse(mt, "roles", adminRoleDoc()),
			cursorResponse(mt, "roles", adminRoleDoc()),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)
		router := conversationTagsRouter(mt, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
		body := bytes.NewBufferString(`{"tags":[" VIP ","vip","Lead"]}`)
		rec := performRequest(router, http.MethodPatch, "/api/conversations/conv_1/tags", body)

		if rec.Code != http.StatusOK {
			mt.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Data models.Conversation `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			mt.Fatalf("could not decode response: %v", err)
		}
		if len(payload.Data.Tags) != 2 || payload.Data.Tags[0] != "vip" || payload.Data.Tags[1] != "lead" {
			mt.Fatalf("unexpected tags: %#v", payload.Data.Tags)
		}
	})
}

func TestUpdateConversationTagsRejectsUnauthorizedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("forbidden", func(mt *mtest.T) {
		mt.AddMockResponses(
			cursorResponse(mt, "conversations", conversationDoc("conv_1", "open")),
			cursorResponse(mt, "conversation_members", bson.D{{Key: "n", Value: 1}}),
			cursorResponse(mt, "conversation_members", bson.D{{Key: "n", Value: 1}}),
		)
		router := conversationTagsRouter(mt, models.User{Base: testBase("usr_member")})
		body := bytes.NewBufferString(`{"tags":["vip"]}`)
		rec := performRequest(router, http.MethodPatch, "/api/conversations/conv_1/tags", body)

		if rec.Code != http.StatusForbidden {
			mt.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestUpdateConversationTagsReturnsNotFoundForHiddenConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("not found", func(mt *mtest.T) {
		mt.AddMockResponses(cursorResponse(mt, "conversations"))
		router := conversationTagsRouter(mt, models.User{Base: testBase("usr_admin"), RoleIDs: []string{"role_admin"}})
		body := bytes.NewBufferString(`{"tags":["vip"]}`)
		rec := performRequest(router, http.MethodPatch, "/api/conversations/missing/tags", body)

		if rec.Code != http.StatusNotFound {
			mt.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func conversationTagsRouter(mt *mtest.T, user models.User) *gin.Engine {
	db := &database.Mongo{Client: mt.Client, DB: mt.DB}
	handler := &Handler{db: db, rbac: rbac.NewChecker(db)}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CurrentUserKey, user)
		c.Next()
	})
	router.PATCH("/api/conversations/:conversationId/tags", handler.updateConversationTags)
	return router
}

func performRequest(router *gin.Engine, method string, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func cursorResponse(mt *mtest.T, collection string, docs ...bson.D) bson.D {
	return mtest.CreateCursorResponse(0, mt.DB.Name()+"."+collection, mtest.FirstBatch, docs...)
}

func conversationDoc(id string, status string) bson.D {
	now := time.Now().UTC()
	return bson.D{
		{Key: "_id", Value: id},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "channel_account_id", Value: "ca_1"},
		{Key: "external_conversation_id", Value: "customer_1"},
		{Key: "customer_ref", Value: "customer_1"},
		{Key: "status", Value: status},
		{Key: "last_message_at", Value: now},
		{Key: "unread_count", Value: 0},
		{Key: "tags", Value: bson.A{}},
	}
}

func adminRoleDoc() bson.D {
	now := time.Now().UTC()
	return bson.D{
		{Key: "_id", Value: "role_admin"},
		{Key: "created_at", Value: now},
		{Key: "updated_at", Value: now},
		{Key: "name", Value: "Admin"},
		{Key: "code", Value: "admin"},
		{Key: "permission_codes", Value: bson.A{"admin:manage", "conversation:assign", "message:send_team"}},
		{Key: "is_system", Value: true},
	}
}

func testBase(id string) models.Base {
	now := time.Now().UTC()
	return models.Base{ID: id, CreatedAt: now, UpdatedAt: now}
}
