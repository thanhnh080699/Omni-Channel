package workers

import (
	"context"
	"testing"
	"time"

	"omni-channel/backend/internal/channel"
	"omni-channel/backend/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestConversationUpsertUpdateDoesNotConflictOperators(t *testing.T) {
	update := conversationUpsertUpdate("conv_1", channel.NormalizedInboundMessage{
		ChannelAccountID:       "ca_1",
		ExternalConversationID: "84900000000@s.whatsapp.net",
		SenderExternalID:       "84900000000@s.whatsapp.net",
		SenderDisplayName:      "Customer Name",
		EventTime:              time.Date(2026, 6, 8, 4, 0, 0, 0, time.UTC),
	}, time.Date(2026, 6, 8, 4, 0, 1, 0, time.UTC))

	operators := []string{"$setOnInsert", "$set", "$max"}
	seen := map[string]string{}
	for _, operator := range operators {
		fields, ok := update[operator].(bson.M)
		if !ok {
			t.Fatalf("expected %s to be bson.M, got %#v", operator, update[operator])
		}
		for field := range fields {
			if previous := seen[field]; previous != "" {
				t.Fatalf("field %q appears in both %s and %s", field, previous, operator)
			}
			seen[field] = operator
		}
	}

	if update["$set"].(bson.M)["customer_name"] != "Customer Name" {
		t.Fatalf("expected customer_name to be updated from sender display name")
	}
}

func TestGetNestedStringField(t *testing.T) {
	// Test with map[string]any
	raw := map[string]any{
		"key": map[string]any{
			"remoteJid":    "12345678@lid",
			"remoteJidAlt": "84923861999@s.whatsapp.net",
		},
	}
	jid := getNestedStringField(raw, "key", "remoteJid")
	if jid != "12345678@lid" {
		t.Fatalf("expected '12345678@lid', got %q", jid)
	}

	jidAlt := getNestedStringField(raw, "key", "remoteJidAlt")
	if jidAlt != "84923861999@s.whatsapp.net" {
		t.Fatalf("expected '84923861999@s.whatsapp.net', got %q", jidAlt)
	}

	// Test with bson.M
	rawBson := bson.M{
		"key": bson.M{
			"remoteJid": "12345678@lid",
		},
	}
	jidBson := getNestedStringField(rawBson, "key", "remoteJid")
	if jidBson != "12345678@lid" {
		t.Fatalf("expected '12345678@lid' (bson), got %q", jidBson)
	}
}

func TestReconcileLidConversation(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	
	mt.Run("rename-conversation", func(mt *mtest.T) {
		now := time.Now().UTC()
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: "conv_lid_1"},
				{Key: "channel_account_id", Value: "ca_1"},
				{Key: "external_conversation_id", Value: "12345678@lid"},
				{Key: "customer_ref", Value: "12345678@lid"},
			}),
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch), // empty cursor -> ErrNoDocuments
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
		)

		p := NewConversationProcessor(&database.Mongo{Client: mt.Client, DB: mt.DB})
		rawPayload := map[string]any{
			"key": map[string]any{
				"remoteJid": "12345678@lid",
			},
		}
		err := p.reconcileLidConversation(context.Background(), "ca_1", "84923861999@s.whatsapp.net", rawPayload, now)
		if err != nil {
			mt.Fatalf("expected reconcile to succeed, got %v", err)
		}
	})

	mt.Run("merge-conversation", func(mt *mtest.T) {
		now := time.Now().UTC()
		mt.AddMockResponses(
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: "conv_lid_1"},
				{Key: "channel_account_id", Value: "ca_1"},
				{Key: "external_conversation_id", Value: "12345678@lid"},
			}),
			mtest.CreateCursorResponse(0, "omni_test.conversations", mtest.FirstBatch, bson.D{
				{Key: "_id", Value: "conv_phone_1"},
				{Key: "channel_account_id", Value: "ca_1"},
				{Key: "external_conversation_id", Value: "84923861999@s.whatsapp.net"},
			}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}),
		)

		p := NewConversationProcessor(&database.Mongo{Client: mt.Client, DB: mt.DB})
		rawPayload := map[string]any{
			"key": map[string]any{
				"remoteJid": "12345678@lid",
			},
		}
		err := p.reconcileLidConversation(context.Background(), "ca_1", "84923861999@s.whatsapp.net", rawPayload, now)
		if err != nil {
			mt.Fatalf("expected reconcile to succeed, got %v", err)
		}
	})
}
