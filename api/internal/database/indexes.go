package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndexes(ctx context.Context, db *Mongo) error {
	indexes := map[string][]mongo.IndexModel{
		"users": {
			uniqueIndex(bson.D{{Key: "email", Value: 1}}),
			plainIndex(bson.D{{Key: "status", Value: 1}}),
			plainIndex(bson.D{{Key: "team_ids", Value: 1}}),
		},
		"roles": {
			uniqueIndex(bson.D{{Key: "code", Value: 1}}),
		},
		"permissions": {
			uniqueIndex(bson.D{{Key: "code", Value: 1}}),
			plainIndex(bson.D{{Key: "resource", Value: 1}, {Key: "action", Value: 1}}),
		},
		"teams": {
			plainIndex(bson.D{{Key: "parent_team_id", Value: 1}}),
			plainIndex(bson.D{{Key: "manager_user_ids", Value: 1}}),
		},
		"team_members": {
			uniqueIndex(bson.D{{Key: "team_id", Value: 1}, {Key: "user_id", Value: 1}}),
			plainIndex(bson.D{{Key: "user_id", Value: 1}}),
			plainIndex(bson.D{{Key: "manager_user_id", Value: 1}}),
		},
		"channels": {
			uniqueIndex(bson.D{{Key: "code", Value: 1}}),
		},
		"channel_accounts": {
			plainIndex(bson.D{{Key: "channel_id", Value: 1}}),
			plainIndex(bson.D{{Key: "owner_team_id", Value: 1}}),
			plainIndex(bson.D{{Key: "shared_team_ids", Value: 1}}),
			plainIndex(bson.D{{Key: "shared_user_ids", Value: 1}}),
			plainIndex(bson.D{{Key: "enabled", Value: 1}, {Key: "session_status", Value: 1}}),
		},
		"conversations": {
			uniqueIndex(bson.D{{Key: "channel_account_id", Value: 1}, {Key: "external_conversation_id", Value: 1}}),
			plainIndex(bson.D{{Key: "assigned_user_id", Value: 1}, {Key: "status", Value: 1}}),
			plainIndex(bson.D{{Key: "assigned_team_id", Value: 1}, {Key: "status", Value: 1}}),
			plainIndex(bson.D{{Key: "last_message_at", Value: -1}}),
		},
		"conversation_members": {
			uniqueIndex(bson.D{{Key: "conversation_id", Value: 1}, {Key: "user_id", Value: 1}}),
			plainIndex(bson.D{{Key: "user_id", Value: 1}}),
		},
		"messages": {
			sparseUniqueIndex(bson.D{{Key: "channel_message_key", Value: 1}}),
			plainIndex(bson.D{{Key: "conversation_id", Value: 1}, {Key: "event_time", Value: 1}}),
			plainIndex(bson.D{{Key: "conversation_id", Value: 1}, {Key: "_id", Value: 1}}),
			plainIndex(bson.D{{Key: "status", Value: 1}}),
		},
		"message_attachments": {
			plainIndex(bson.D{{Key: "message_id", Value: 1}}),
			plainIndex(bson.D{{Key: "conversation_id", Value: 1}, {Key: "status", Value: 1}}),
			plainIndex(bson.D{{Key: "status", Value: 1}}),
		},
		"inbound_events": {
			uniqueIndex(bson.D{{Key: "idempotency_key", Value: 1}}),
			plainIndex(bson.D{{Key: "status", Value: 1}, {Key: "queued_at", Value: 1}}),
			plainIndex(bson.D{{Key: "channel_account_id", Value: 1}, {Key: "event_time", Value: 1}}),
		},
		"outbound_events": {
			uniqueIndex(bson.D{{Key: "idempotency_key", Value: 1}}),
			plainIndex(bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: 1}}),
			plainIndex(bson.D{{Key: "message_id", Value: 1}}),
		},
		"queue_jobs": {
			plainIndex(bson.D{{Key: "queue_name", Value: 1}, {Key: "status", Value: 1}, {Key: "next_run_at", Value: 1}}),
			plainIndex(bson.D{{Key: "ref_id", Value: 1}}),
		},
		"sync_checkpoints": {
			uniqueIndex(bson.D{{Key: "channel_account_id", Value: 1}, {Key: "conversation_id", Value: 1}}),
			plainIndex(bson.D{{Key: "last_synced_at", Value: 1}}),
		},
		"audit_logs": {
			plainIndex(bson.D{{Key: "actor_user_id", Value: 1}, {Key: "created_at", Value: -1}}),
			plainIndex(bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}}),
			plainIndex(bson.D{{Key: "action", Value: 1}, {Key: "created_at", Value: -1}}),
		},
	}

	for collection, models := range indexes {
		if len(models) == 0 {
			continue
		}
		if _, err := db.C(collection).Indexes().CreateMany(ctx, models); err != nil {
			return err
		}
	}
	return nil
}

func plainIndex(keys bson.D) mongo.IndexModel {
	return mongo.IndexModel{Keys: keys}
}

func uniqueIndex(keys bson.D) mongo.IndexModel {
	return mongo.IndexModel{Keys: keys, Options: options.Index().SetUnique(true)}
}

func sparseUniqueIndex(keys bson.D) mongo.IndexModel {
	return mongo.IndexModel{Keys: keys, Options: options.Index().SetUnique(true).SetSparse(true)}
}
