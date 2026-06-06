package handlers

import (
	"net/http"
	"strings"
	"time"

	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := timeout(c)
	defer cancel()

	var user models.User
	if err := h.db.C("users").FindOne(ctx, bson.M{"email": strings.ToLower(req.Email), "status": "active"}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, expiresAt, err := h.tokens.Generate(user.ID, user.Email, user.RoleIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}

	now := time.Now().UTC()
	_, _ = h.db.C("users").UpdateByID(ctx, user.ID, bson.M{"$set": bson.M{"last_login_at": now, "updated_at": now}})
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_at": expiresAt, "user": user})
}

func (h *Handler) profile(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	permissions, err := h.rbac.PermissionCodes(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "permissions": permissions})
}

func (h *Handler) updateProfile(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()

	set := bson.M{
		"display_name": strings.TrimSpace(req.DisplayName),
		"email":        strings.ToLower(strings.TrimSpace(req.Email)),
	}
	result, err := h.db.C("users").UpdateByID(ctx, user.ID, updateTimeSet(set))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "could not update profile"})
		return
	}
	user.DisplayName = set["display_name"].(string)
	user.Email = set["email"].(string)
	h.audit(c, "user.profile_update", "user", user.ID, bson.M{"display_name": user.DisplayName, "email": user.Email})
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *Handler) changePassword(c *gin.Context) {
	user, ok := currentUserOrAbort(c)
	if !ok {
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is invalid"})
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}
	ctx, cancel := timeout(c)
	defer cancel()
	result, err := h.db.C("users").UpdateByID(ctx, user.ID, updateTimeSet(bson.M{"password_hash": hash}))
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	h.audit(c, "user.password_change", "user", user.ID, nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) refresh(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	token, expiresAt, err := h.tokens.Generate(user.ID, user.Email, user.RoleIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": token, "expires_at": expiresAt})
}
