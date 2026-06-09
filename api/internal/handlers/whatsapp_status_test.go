package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestWhatsAppMessageStatusUpdatesOutboundMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("read", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}))
		handler := &Handler{db: &database.Mongo{Client: mt.Client, DB: mt.DB}, cfg: config.Config{WebhookSharedSecret: "secret"}}
		router := gin.New()
		router.POST("/internal/whatsapp/message-status", handler.whatsAppMessageStatus)

		body := bytes.NewBufferString(`{"channel_account_id":"ca_1","channel_message_id":"wa_1","status":"read","event_time":"2026-06-08T12:00:00Z"}`)
		req := httptest.NewRequest(http.MethodPost, "/internal/whatsapp/message-status", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Secret", "secret")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			mt.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}
