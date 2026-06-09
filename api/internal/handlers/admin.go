package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
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

type listMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func pagingOptions(c *gin.Context) (int, int, *options.FindOptions) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	opts := options.Find().SetSkip(int64((page - 1) * limit)).SetLimit(int64(limit))
	return page, limit, opts
}

func textFilter(field string, q string) bson.M {
	if strings.TrimSpace(q) == "" {
		return bson.M{}
	}
	return bson.M{field: bson.M{"$regex": strings.TrimSpace(q), "$options": "i"}}
}

func (h *Handler) listUsers(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()

	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter["status"] = status
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filter["$or"] = []bson.M{textFilter("email", q), textFilter("display_name", q)}
	}
	page, limit, opts := pagingOptions(c)
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.C("users").Find(ctx, filter, opts)
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
	total, _ := h.db.C("users").CountDocuments(ctx, filter)
	c.JSON(http.StatusOK, gin.H{"data": users, "meta": listMeta{Page: page, Limit: limit, Total: total}})
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
	filter := bson.M{}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filter["$or"] = []bson.M{textFilter("name", q), textFilter("code", q)}
	}
	page, limit, opts := pagingOptions(c)
	opts.SetSort(bson.D{{Key: "code", Value: 1}})
	cursor, err := h.db.C("roles").Find(ctx, filter, opts)
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
	total, _ := h.db.C("roles").CountDocuments(ctx, filter)
	c.JSON(http.StatusOK, gin.H{"data": roles, "meta": listMeta{Page: page, Limit: limit, Total: total}})
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
	var existing models.Role
	if err := h.db.C("roles").FindOne(c.Request.Context(), bson.M{"_id": c.Param("roleId")}).Decode(&existing); err == nil && existing.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "system roles cannot be edited"})
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
	filter := bson.M{}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter["status"] = status
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		filter = bson.M{"$and": []bson.M{filter, textFilter("name", q)}}
	}
	page, limit, opts := pagingOptions(c)
	opts.SetSort(bson.D{{Key: "name", Value: 1}})
	cursor, err := h.db.C("teams").Find(ctx, filter, opts)
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
	total, _ := h.db.C("teams").CountDocuments(ctx, filter)
	c.JSON(http.StatusOK, gin.H{"data": teams, "meta": listMeta{Page: page, Limit: limit, Total: total}})
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

func (h *Handler) updateTeam(c *gin.Context) {
	var req teamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	set := bson.M{
		"name":             req.Name,
		"parent_team_id":   req.ParentTeamID,
		"manager_user_ids": req.ManagerUserIDs,
		"status":           req.Status,
	}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("teams").UpdateByID(ctx, c.Param("teamId"), updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	h.audit(c, "team.update", "team", c.Param("teamId"), set)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) deleteTeam(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("teams").UpdateByID(ctx, c.Param("teamId"), updateTimeSet(bson.M{"status": "disabled"}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}
	h.audit(c, "team.delete", "team", c.Param("teamId"), nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

func (h *Handler) listChannelAccounts(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	filter := bson.M{}
	user, _ := currentUserOrAbort(c)
	isAdmin, _ := h.rbac.Has(ctx, user, "admin:manage")
	if !isAdmin {
		filter["$or"] = []bson.M{
			{"owner_team_id": bson.M{"$in": user.TeamIDs}},
			{"shared_team_ids": bson.M{"$in": user.TeamIDs}},
			{"shared_user_ids": user.ID},
		}
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		filter["enabled"] = enabled == "true" || enabled == "1"
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter["session_status"] = status
	}
	if q := strings.TrimSpace(c.Query("q")); q != "" {
		search := []bson.M{textFilter("name", q), textFilter("channel_id", q), textFilter("owner_team_id", q)}
		if existingOr, ok := filter["$or"]; ok {
			filter = bson.M{"$and": []bson.M{{"$or": existingOr}, {"$or": search}}}
		} else {
			filter["$or"] = search
		}
	}
	page, limit, opts := pagingOptions(c)
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.C("channel_accounts").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list channel accounts"})
		return
	}
	defer cursor.Close(ctx)
	var accounts []models.ChannelAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode channel accounts"})
		return
	}
	total, _ := h.db.C("channel_accounts").CountDocuments(ctx, filter)
	c.JSON(http.StatusOK, gin.H{"data": accounts, "meta": listMeta{Page: page, Limit: limit, Total: total}})
}

type channelAccountRequest struct {
	ChannelID        string                 `json:"channel_id" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	OwnerTeamID      string                 `json:"owner_team_id"`
	SharedTeamIDs    []string               `json:"shared_team_ids"`
	SharedUserIDs    []string               `json:"shared_user_ids"`
	CredentialRef    string                 `json:"credential_ref"`
	WebhookSecretRef string                 `json:"webhook_secret_ref"`
	Metadata         map[string]interface{} `json:"metadata"`
	Enabled          *bool                  `json:"enabled"`
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
	user, _ := currentUserOrAbort(c)
	ownerTeamID := req.OwnerTeamID
	if ownerTeamID == "" && len(user.TeamIDs) > 0 {
		ownerTeamID = user.TeamIDs[0]
	}
	account := models.ChannelAccount{
		Base:             newBase(),
		ChannelID:        req.ChannelID,
		Name:             req.Name,
		OwnerTeamID:      ownerTeamID,
		SharedTeamIDs:    req.SharedTeamIDs,
		SharedUserIDs:    req.SharedUserIDs,
		CredentialRef:    req.CredentialRef,
		WebhookSecretRef: req.WebhookSecretRef,
		Metadata:         defaultChannelMetadata(req.Metadata),
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
	user, _ := currentUserOrAbort(c)
	ownerTeamID := req.OwnerTeamID
	if ownerTeamID == "" && len(user.TeamIDs) > 0 {
		ownerTeamID = user.TeamIDs[0]
	}
	set := bson.M{
		"channel_id":         req.ChannelID,
		"name":               req.Name,
		"owner_team_id":      ownerTeamID,
		"shared_team_ids":    req.SharedTeamIDs,
		"shared_user_ids":    req.SharedUserIDs,
		"credential_ref":     req.CredentialRef,
		"webhook_secret_ref": req.WebhookSecretRef,
		"metadata":           defaultChannelMetadata(req.Metadata),
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

func defaultChannelMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	if _, ok := metadata["autoConnect"]; !ok {
		metadata["autoConnect"] = true
	}
	if _, ok := metadata["syncFullHistory"]; !ok {
		metadata["syncFullHistory"] = true
	}
	return metadata
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

func (h *Handler) whatsAppSession(c *gin.Context) {
	h.proxyWhatsAppAdapter(c, http.MethodGet, "/session/"+c.Param("accountId"), true)
}

func (h *Handler) whatsAppConnect(c *gin.Context) {
	h.proxyWhatsAppAdapter(c, http.MethodPost, "/connect/"+c.Param("accountId"), true)
}

func (h *Handler) whatsAppDisconnect(c *gin.Context) {
	h.proxyWhatsAppAdapter(c, http.MethodPost, "/disconnect/"+c.Param("accountId"), false)
}

func (h *Handler) whatsAppResetSession(c *gin.Context) {
	h.clearWhatsAppQRCache(c)
	h.proxyWhatsAppAdapter(c, http.MethodPost, "/reset-session/"+c.Param("accountId"), false)
}

func (h *Handler) whatsAppResync(c *gin.Context) {
	h.proxyWhatsAppAdapter(c, http.MethodPost, "/resync?account_id="+c.Param("accountId"), true)
}

func (h *Handler) proxyWhatsAppAdapter(c *gin.Context, method string, path string, cacheQR bool) {
	cacheKey := h.whatsAppQRCacheKey(c)
	if cacheQR && (method == http.MethodGet || (method == http.MethodPost && strings.Contains(path, "/connect/"))) {
		if cached, ok := h.cachedWhatsAppQR(cacheKey); ok {
			c.JSON(http.StatusOK, cached)
			return
		}
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, strings.TrimRight(h.cfg.WhatsAppAdapterURL, "/")+path, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build adapter request"})
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if method == http.MethodGet && strings.Contains(path, "/session/") {
			c.JSON(http.StatusOK, gin.H{
				"accountId":  c.Param("accountId"),
				"status":     "error",
				"lastError":  "whatsapp adapter unavailable",
				"adapter_up": false,
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "whatsapp adapter unavailable"})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if cacheQR && resp.StatusCode < 300 {
		body = h.mergeWhatsAppQRCache(cacheKey, body)
	}
	c.Data(resp.StatusCode, "application/json", body)
}

func (h *Handler) whatsAppQRCacheKey(c *gin.Context) string {
	user, _ := currentUserOrAbort(c)
	return user.ID + ":" + c.Param("accountId")
}

func (h *Handler) cachedWhatsAppQR(key string) (gin.H, bool) {
	if h.qrCache == nil {
		return nil, false
	}
	h.qrCache.mu.Lock()
	defer h.qrCache.mu.Unlock()
	entry, ok := h.qrCache.entries[key]
	if !ok || entry.QR == "" || time.Now().UTC().After(entry.ExpiresAt) {
		if ok {
			delete(h.qrCache.entries, key)
		}
		return nil, false
	}
	return gin.H{
		"accountId":     accountIDFromQRCacheKey(key),
		"status":        entry.Status,
		"qr":            entry.QR,
		"cached":        true,
		"qr_cached_at":  entry.UpdatedAt,
		"qr_expires_at": entry.ExpiresAt,
	}, true
}

func (h *Handler) mergeWhatsAppQRCache(key string, body []byte) []byte {
	if h.qrCache == nil {
		return body
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	status, _ := payload["status"].(string)
	qr, _ := payload["qr"].(string)
	qrExpiresAt := parseQRExpiresAt(payload["qrExpiresAt"], payload["qr_expires_at"])

	h.qrCache.mu.Lock()
	defer h.qrCache.mu.Unlock()
	if qr != "" {
		if !qrExpiresAt.After(time.Now().UTC()) {
			delete(h.qrCache.entries, key)
			delete(payload, "qr")
			payload["status"] = "connecting"
			payload["lastError"] = "QR expired, waiting for a new QR"
			return mustJSON(body, payload)
		}
		h.qrCache.entries[key] = qrCacheEntry{QR: qr, Status: status, UpdatedAt: time.Now().UTC(), ExpiresAt: qrExpiresAt}
		payload["cached"] = false
		payload["qr_cached_at"] = h.qrCache.entries[key].UpdatedAt
		payload["qr_expires_at"] = h.qrCache.entries[key].ExpiresAt
	} else if entry, ok := h.qrCache.entries[key]; ok && entry.QR != "" && status != "connected" {
		if time.Now().UTC().Before(entry.ExpiresAt) {
			payload["qr"] = entry.QR
			payload["cached"] = true
			payload["qr_cached_at"] = entry.UpdatedAt
			payload["qr_expires_at"] = entry.ExpiresAt
		} else {
			delete(h.qrCache.entries, key)
		}
	} else if status == "connected" || status == "disconnected" {
		delete(h.qrCache.entries, key)
	}
	return mustJSON(body, payload)
}

func parseQRExpiresAt(values ...interface{}) time.Time {
	for _, value := range values {
		raw, ok := value.(string)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC().Add(25 * time.Second)
}

func mustJSON(fallback []byte, payload map[string]interface{}) []byte {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fallback
	}
	return encoded
}

func accountIDFromQRCacheKey(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return key
}

func (h *Handler) clearWhatsAppQRCache(c *gin.Context) {
	if h.qrCache == nil {
		return
	}
	h.qrCache.mu.Lock()
	defer h.qrCache.mu.Unlock()
	delete(h.qrCache.entries, h.whatsAppQRCacheKey(c))
}

func (h *Handler) listAuditLogs(c *gin.Context) {
	ctx, cancel := timeout(c)
	defer cancel()
	filter := bson.M{}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		filter["action"] = bson.M{"$regex": action, "$options": "i"}
	}
	if actor := strings.TrimSpace(c.Query("actor_user_id")); actor != "" {
		filter["actor_user_id"] = actor
	}
	if resourceType := strings.TrimSpace(c.Query("resource_type")); resourceType != "" {
		filter["resource_type"] = resourceType
	}
	if resourceID := strings.TrimSpace(c.Query("resource_id")); resourceID != "" {
		filter["resource_id"] = resourceID
	}
	page, limit, opts := pagingOptions(c)
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := h.db.C("audit_logs").Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list audit logs"})
		return
	}
	defer cursor.Close(ctx)
	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not decode audit logs"})
		return
	}
	total, _ := h.db.C("audit_logs").CountDocuments(ctx, filter)
	c.JSON(http.StatusOK, gin.H{"data": logs, "meta": listMeta{Page: page, Limit: limit, Total: total}})
}
