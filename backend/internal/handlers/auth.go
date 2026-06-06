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
