package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type createUserRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	Password    string   `json:"password" binding:"required,min=8"`
	DisplayName string   `json:"display_name" binding:"required"`
	RoleIDs     []string `json:"role_ids"`
	TeamIDs     []string `json:"team_ids"`
}

type updateUserRequest struct {
	DisplayName *string  `json:"display_name"`
	Status      *string  `json:"status"`
	Password    *string  `json:"password"`
	RoleIDs     []string `json:"role_ids"`
	TeamIDs     []string `json:"team_ids"`
}

func (h *Handler) listUsers(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()

	cursor, err := h.db.C("users").Find(ctx, bson.M{"deleted_at": bson.M{"$exists": false}}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list users"})
		return
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}
	user := models.User{
		Base:         newBase(),
		Email:        strings.ToLower(req.Email),
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Status:       "active",
		RoleIDs:      req.RoleIDs,
		TeamIDs:      req.TeamIDs,
	}
	ctx, cancel := timeout(c)
	defer cancel()

	if _, err := h.db.C("users").InsertOne(ctx, user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "could not create user"})
		return
	}
	h.audit(c, "user.create", "user", user.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) updateUser(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	set := bson.M{}
	if req.DisplayName != nil {
		set["display_name"] = *req.DisplayName
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
			return
		}
		set["password_hash"] = hash
	}
	if req.RoleIDs != nil {
		set["role_ids"] = req.RoleIDs
	}
	if req.TeamIDs != nil {
		set["team_ids"] = req.TeamIDs
	}
	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	result, err := h.db.C("users").UpdateByID(ctx, c.Param("userId"), updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	h.audit(c, "user.update", "user", c.Param("userId"), set)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) deleteUser(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	now := time.Now().UTC()
	result, err := h.db.C("users").UpdateByID(ctx, c.Param("userId"), bson.M{"$set": bson.M{"status": "disabled", "deleted_at": now, "updated_at": now}})
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	h.audit(c, "user.delete", "user", c.Param("userId"), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) listRoles(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	cursor, err := h.db.C("roles").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "code", Value: 1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list roles"})
		return
	}
	defer cursor.Close(ctx)
	var roles []models.Role
	if err := cursor.All(ctx, &roles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

type roleRequest struct {
	Name            string   `json:"name" binding:"required"`
	Code            string   `json:"code" binding:"required"`
	PermissionCodes []string `json:"permission_codes"`
}

func (h *Handler) createRole(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := models.Role{Base: newBase(), Name: req.Name, Code: req.Code, PermissionCodes: req.PermissionCodes}
	ctx, cancel := timeout(c)
	defer cancel()
	if _, err := h.db.C("roles").InsertOne(ctx, role); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "could not create role"})
		return
	}
	h.audit(c, "role.create", "role", role.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (h *Handler) updateRole(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	set := bson.M{"name": req.Name, "code": req.Code, "permission_codes": req.PermissionCodes}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("roles").UpdateByID(ctx, c.Param("roleId"), updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	h.audit(c, "role.update", "role", c.Param("roleId"), set)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type assignIDsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

func (h *Handler) assignRoles(c *gin.Context) {
	h.assignUserIDs(c, "role_ids", "user.role_assign")
}

func (h *Handler) assignTeams(c *gin.Context) {
	h.assignUserIDs(c, "team_ids", "user.team_assign")
}

func (h *Handler) assignUserIDs(c *gin.Context, field string, action string) {
	var req assignIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("users").UpdateByID(ctx, c.Param("userId"), updateTimeSet(bson.M{field: req.IDs}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	h.audit(c, action, "user", c.Param("userId"), bson.M{field: req.IDs})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) permissionMatrix(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	rolesCursor, err := h.db.C("roles").Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load roles"})
		return
	}
	defer rolesCursor.Close(ctx)
	var roles []models.Role
	_ = rolesCursor.All(ctx, &roles)

	permsCursor, err := h.db.C("permissions").Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load permissions"})
		return
	}
	defer permsCursor.Close(ctx)
	var permissions []models.Permission
	_ = permsCursor.All(ctx, &permissions)
	c.JSON(http.StatusOK, gin.H{"roles": roles, "permissions": permissions})
}

type teamRequest struct {
	Name           string   `json:"name" binding:"required"`
	ParentTeamID   string   `json:"parent_team_id"`
	ManagerUserIDs []string `json:"manager_user_ids"`
	Status         string   `json:"status"`
}

func (h *Handler) listTeams(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	cursor, err := h.db.C("teams").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list teams"})
		return
	}
	defer cursor.Close(ctx)
	var teams []models.Team
	if err := cursor.All(ctx, &teams); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode teams"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": teams})
}

func (h *Handler) createTeam(c *gin.Context) {
	var req teamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	team := models.Team{Base: newBase(), Name: req.Name, ParentTeamID: req.ParentTeamID, ManagerUserIDs: req.ManagerUserIDs, Status: req.Status}
	ctx, cancel := timeout(c)
	defer cancel()
	if _, err := h.db.C("teams").InsertOne(ctx, team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create team"})
		return
	}
	h.audit(c, "team.create", "team", team.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"data": team})
}

func (h *Handler) listChannels(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	cursor, err := h.db.C("channels").Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "code", Value: 1}}))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list channels"})
		return
	}
	defer cursor.Close(ctx)
	var channels []models.Channel
	if err := cursor.All(ctx, &channels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

type channelAccountRequest struct {
	ChannelID        string `json:"channel_id" binding:"required"`
	Name             string `json:"name" binding:"required"`
	OwnerTeamID      string `json:"owner_team_id"`
	CredentialRef    string `json:"credential_ref"`
	WebhookSecretRef string `json:"webhook_secret_ref"`
	Enabled          *bool  `json:"enabled"`
}

func (h *Handler) createChannelAccount(c *gin.Context) {
	var req channelAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	account := models.ChannelAccount{
		Base:             newBase(),
		ChannelID:        req.ChannelID,
		Name:             req.Name,
		OwnerTeamID:      req.OwnerTeamID,
		CredentialRef:    req.CredentialRef,
		WebhookSecretRef: req.WebhookSecretRef,
		SessionStatus:    "unknown",
		Enabled:          enabled,
	}
	ctx, cancel := timeout(c)
	defer cancel()
	if _, err := h.db.C("channel_accounts").InsertOne(ctx, account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create channel account"})
		return
	}
	h.audit(c, "channel_account.create", "channel_account", account.ID, nil)
	c.JSON(http.StatusCreated, gin.H{"data": account})
}

func (h *Handler) updateChannelAccount(c *gin.Context) {
	var req channelAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	set := bson.M{
		"channel_id":         req.ChannelID,
		"name":               req.Name,
		"owner_team_id":      req.OwnerTeamID,
		"credential_ref":     req.CredentialRef,
		"webhook_secret_ref": req.WebhookSecretRef,
	}
	if req.Enabled != nil {
		set["enabled"] = *req.Enabled
	}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("channel_accounts").UpdateByID(ctx, c.Param("accountId"), updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel account not found"})
		return
	}
	h.audit(c, "channel_account.update", "channel_account", c.Param("accountId"), set)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) enableChannelAccount(c *gin.Context) {
	h.setChannelAccountEnabled(c, true)
}

func (h *Handler) disableChannelAccount(c *gin.Context) {
	h.setChannelAccountEnabled(c, false)
}

func (h *Handler) setChannelAccountEnabled(c *gin.Context, enabled bool) {
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("channel_accounts").UpdateByID(ctx, c.Param("accountId"), updateTimeSet(bson.M{"enabled": enabled}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel account not found"})
		return
	}
	action := "channel_account.disable"
	if enabled {
		action = "channel_account.enable"
	}
	h.audit(c, action, "channel_account", c.Param("accountId"), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) channelAccountHealth(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	var account models.ChannelAccount
	err := h.db.C("channel_accounts").FindOne(ctx, bson.M{"_id": c.Param("accountId")}).Decode(&account)
	if errors.Is(err, mongo.ErrNoDocuments) {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel account not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load channel account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"channel_health":  account.SessionStatus,
		"enabled":         account.Enabled,
		"last_webhook_at": account.LastWebhookAt,
		"last_sync_at":    account.LastSyncAt,
		"last_error":      account.LastError,
		"session_status":  account.SessionStatus,
	})
}
