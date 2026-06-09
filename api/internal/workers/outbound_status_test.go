package workers

import (
	"context"
	"testing"
	"time"

	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/queue"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMarkOutboundSentUpdatesMessageAndOutboundEvent(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("sent", func(mt *mtest.T) {
		mt.AddMockResponses(
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
			mtest.CreateSuccessResponse(bson.E{Key: "n", Value: 1}, bson.E{Key: "nModified", Value: 1}),
		)
		runner := &Runner{db: &database.Mongo{Client: mt.Client, DB: mt.DB}}
		err := runner.markOutboundSent(context.Background(), queue.OutboundEventPayload{
			MessageID:        "msg_1",
			OutboundEventID:  "out_1",
			ChannelAccountID: "ca_1",
			Attempt:          0,
			QueuedAt:         time.Now().UTC(),
			ExpiresAt:        time.Now().UTC().Add(time.Minute),
		}, "wa_msg_1")
		if err != nil {
			mt.Fatalf("expected mark sent to succeed: %v", err)
		}
	})
}

func TestBuildOutboundAdapterPayloadLoadsConversationTargetAndText(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock).DatabaseName("omni_test"))
	mt.Run("payload", func(mt *mtest.T) {
		now := time.Now().UTC()
		mt.AddMockResponses(
			workerCursorResponse(mt, "messages", bson.D{
				{Key: "_id", Value: "msg_1"},
				{Key: "created_at", Value: now},
				{Key: "updated_at", Value: now},
				{Key: "conversation_id", Value: "conv_1"},
				{Key: "direction", Value: "outbound"},
				{Key: "sender_type", Value: "agent"},
				{Key: "text", Value: "hello"},
				{Key: "status", Value: "pending"},
				{Key: "event_time", Value: now},
			}),
			workerCursorResponse(mt, "conversations", bson.D{
				{Key: "_id", Value: "conv_1"},
				{Key: "created_at", Value: now},
				{Key: "updated_at", Value: now},
				{Key: "channel_account_id", Value: "ca_1"},
				{Key: "external_conversation_id", Value: "84900000000@s.whatsapp.net"},
				{Key: "status", Value: "open"},
				{Key: "last_message_at", Value: now},
				{Key: "unread_count", Value: 0},
				{Key: "tags", Value: bson.A{}},
			}),
		)
		runner := &Runner{db: &database.Mongo{Client: mt.Client, DB: mt.DB}}
		payload, err := runner.buildOutboundAdapterPayload(context.Background(), queue.OutboundEventPayload{
			MessageID:        "msg_1",
			OutboundEventID:  "out_1",
			ChannelAccountID: "ca_1",
			IdempotencyKey:   "msg_1:send",
			Attempt:          1,
			QueuedAt:         now,
			ExpiresAt:        now.Add(time.Minute),
		})
		if err != nil {
			mt.Fatalf("expected payload build to succeed: %v", err)
		}
		if payload["external_conversation_id"] != "84900000000@s.whatsapp.net" || payload["text"] != "hello" {
			mt.Fatalf("unexpected adapter payload: %#v", payload)
		}
	})
}

func workerCursorResponse(mt *mtest.T, collection string, docs ...bson.D) bson.D {
	return mtest.CreateCursorResponse(0, mt.DB.Name()+"."+collection, mtest.FirstBatch, docs...)
}
