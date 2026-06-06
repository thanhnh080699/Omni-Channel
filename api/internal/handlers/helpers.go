package handlers

import (
	"context"
	"net/http"
	"time"

	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func newBase() models.Base {
	now := time.Now().UTC()
	return models.Base{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now}
}

func timeout(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), 10*time.Second)
}

func (h *Handler) requirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
			return
		}
		allowed, err := h.rbac.Has(c.Request.Context(), user, permission)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}

func (h *Handler) requireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
			return
		}
		for _, permission := range permissions {
			allowed, err := h.rbac.Has(c.Request.Context(), user, permission)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
				return
			}
			if allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	}
}

func (h *Handler) audit(c *gin.Context, action string, resourceType string, resourceID string, metadata map[string]interface{}) {
	user, _ := middleware.CurrentUser(c)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	log := models.AuditLog{
		Base:         newBase(),
		ActorUserID:  user.ID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	}
	_, _ = h.db.C("audit_logs").InsertOne(ctx, log)
}

func currentUserOrAbort(c *gin.Context) (models.User, bool) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return models.User{}, false
	}
	return user, true
}

func updateTimeSet(fields bson.M) bson.M {
	fields["updated_at"] = time.Now().UTC()
	return bson.M{"$set": fields}
}
