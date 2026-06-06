package database

import (
	"context"
	"errors"
	"strings"
	"time"

	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var defaultPermissions = []models.Permission{
	{Base: newBase("perm_conversation_view_all"), Code: "conversation:view_all", Resource: "conversation", Action: "view_all", Description: "View all conversations"},
	{Base: newBase("perm_conversation_view_team"), Code: "conversation:view_team", Resource: "conversation", Action: "view_team", Description: "View team conversations"},
	{Base: newBase("perm_conversation_view_assigned"), Code: "conversation:view_assigned", Resource: "conversation", Action: "view_assigned", Description: "View assigned conversations"},
	{Base: newBase("perm_conversation_assign"), Code: "conversation:assign", Resource: "conversation", Action: "assign", Description: "Assign or transfer conversations"},
	{Base: newBase("perm_message_send_assigned"), Code: "message:send_assigned", Resource: "message", Action: "send_assigned", Description: "Send messages in assigned conversations"},
	{Base: newBase("perm_message_send_team"), Code: "message:send_team", Resource: "message", Action: "send_team", Description: "Send messages in team conversations"},
	{Base: newBase("perm_attachment_view"), Code: "attachment:view", Resource: "attachment", Action: "view", Description: "View permitted attachments"},
	{Base: newBase("perm_admin_manage"), Code: "admin:manage", Resource: "admin", Action: "manage", Description: "Manage users, roles, teams, and channels"},
	{Base: newBase("perm_audit_view"), Code: "audit:view", Resource: "audit", Action: "view", Description: "View audit logs"},
}

var defaultRoles = []models.Role{
	{
		Base: newBase("role_admin"), Name: "Admin", Code: "admin", IsSystem: true,
		PermissionCodes: []string{"conversation:view_all", "conversation:assign", "message:send_team", "message:send_assigned", "attachment:view", "admin:manage", "audit:view"},
	},
	{
		Base: newBase("role_manager"), Name: "Manager", Code: "manager", IsSystem: true,
		PermissionCodes: []string{"conversation:view_team", "conversation:assign", "message:send_team", "attachment:view"},
	},
	{
		Base: newBase("role_staff"), Name: "Staff", Code: "staff", IsSystem: true,
		PermissionCodes: []string{"conversation:view_assigned", "message:send_assigned", "attachment:view"},
	},
	{
		Base: newBase("role_auditor"), Name: "Auditor", Code: "auditor", IsSystem: true,
		PermissionCodes: []string{"conversation:view_team", "attachment:view", "audit:view"},
	},
}

var defaultChannels = []models.Channel{
	{Base: newBase("ch_whatsapp"), Code: "whatsapp", Name: "WhatsApp", Kind: "whatsapp", OfficialAPIAvailable: true, Status: "enabled"},
	{Base: newBase("ch_telegram"), Code: "telegram", Name: "Telegram", Kind: "telegram", OfficialAPIAvailable: true, Status: "enabled"},
	{Base: newBase("ch_facebook_page"), Code: "facebook_page", Name: "Facebook Page", Kind: "facebook", OfficialAPIAvailable: true, Status: "enabled"},
	{Base: newBase("ch_facebook_personal"), Code: "facebook_personal", Name: "Facebook Personal", Kind: "facebook", OfficialAPIAvailable: false, Status: "enabled"},
	{Base: newBase("ch_zalo_personal"), Code: "zalo_personal", Name: "Zalo Personal", Kind: "zalo", OfficialAPIAvailable: false, Status: "enabled"},
}

func SeedDefaults(ctx context.Context, db *Mongo, cfg config.Config) error {
	for _, permission := range defaultPermissions {
		if err := upsertByCode(ctx, db.C("permissions"), permission.Code, permission); err != nil {
			return err
		}
	}
	for _, role := range defaultRoles {
		if err := upsertByCode(ctx, db.C("roles"), role.Code, role); err != nil {
			return err
		}
	}
	for _, channel := range defaultChannels {
		if err := upsertByCode(ctx, db.C("channels"), channel.Code, channel); err != nil {
			return err
		}
	}

	email := strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	if email == "" {
		return nil
	}
	var existing models.User
	err := db.C("users").FindOne(ctx, bson.M{"email": email}).Decode(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	admin := models.User{
		Base:         models.Base{ID: "usr_admin", CreatedAt: now, UpdatedAt: now},
		Email:        email,
		PasswordHash: string(passwordHash),
		DisplayName:  "System Admin",
		Status:       "active",
		RoleIDs:      []string{"role_admin"},
		TeamIDs:      []string{},
	}
	_, err = db.C("users").InsertOne(ctx, admin)
	return err
}

func upsertByCode(ctx context.Context, collection *mongo.Collection, code string, doc interface{}) error {
	_, err := collection.UpdateOne(ctx, bson.M{"code": code}, bson.M{"$setOnInsert": doc}, mongoOptionsUpsert())
	return err
}

func mongoOptionsUpsert() *options.UpdateOptions {
	return options.Update().SetUpsert(true)
}

func newBase(id string) models.Base {
	now := time.Now().UTC()
	if id == "" {
		id = uuid.NewString()
	}
	return models.Base{ID: id, CreatedAt: now, UpdatedAt: now}
}
