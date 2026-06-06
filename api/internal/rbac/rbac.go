package rbac

import (
	"context"

	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

type Checker struct {
	db *database.Mongo
}

func NewChecker(db *database.Mongo) *Checker {
	return &Checker{db: db}
}

func (c *Checker) PermissionCodes(ctx context.Context, user models.User) (map[string]bool, error) {
	codes := map[string]bool{}
	if len(user.RoleIDs) == 0 {
		return codes, nil
	}
	cursor, err := c.db.C("roles").Find(ctx, bson.M{"_id": bson.M{"$in": user.RoleIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var role models.Role
		if err := cursor.Decode(&role); err != nil {
			return nil, err
		}
		for _, code := range role.PermissionCodes {
			codes[code] = true
		}
	}
	return codes, cursor.Err()
}

func (c *Checker) Has(ctx context.Context, user models.User, permission string) (bool, error) {
	codes, err := c.PermissionCodes(ctx, user)
	if err != nil {
		return false, err
	}
	return codes[permission] || codes["admin:manage"], nil
}

func (c *Checker) CanViewConversation(ctx context.Context, user models.User, conversation models.Conversation) (bool, error) {
	codes, err := c.PermissionCodes(ctx, user)
	if err != nil {
		return false, err
	}
	if codes["conversation:view_all"] || codes["admin:manage"] {
		return true, nil
	}
	if codes["conversation:view_assigned"] && conversation.AssignedUserID == user.ID {
		return true, nil
	}
	if codes["conversation:view_team"] && contains(user.TeamIDs, conversation.AssignedTeamID) {
		return true, nil
	}
	if codes["conversation:view_team"] {
		count, err := c.db.C("team_members").CountDocuments(ctx, bson.M{
			"manager_user_id": user.ID,
			"user_id":         conversation.AssignedUserID,
		})
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	count, err := c.db.C("conversation_members").CountDocuments(ctx, bson.M{
		"conversation_id": conversation.ID,
		"user_id":         user.ID,
	})
	return count > 0, err
}

func (c *Checker) CanSendMessage(ctx context.Context, user models.User, conversation models.Conversation) (bool, error) {
	if conversation.Status != "open" {
		return false, nil
	}
	allowedView, err := c.CanViewConversation(ctx, user, conversation)
	if err != nil || !allowedView {
		return allowedView, err
	}
	codes, err := c.PermissionCodes(ctx, user)
	if err != nil {
		return false, err
	}
	if codes["admin:manage"] || codes["message:send_team"] {
		return true, nil
	}
	return codes["message:send_assigned"] && conversation.AssignedUserID == user.ID, nil
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle && value != "" {
			return true
		}
	}
	return false
}
